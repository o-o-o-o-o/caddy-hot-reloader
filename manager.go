package hotreloader

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// SiteManager manages multiple sites and their watchers
type SiteManager struct {
	logger     *zap.Logger
	config     *HotReloader
	sites      map[string]*Site // domain -> site
	mu         sync.RWMutex
	wg         sync.WaitGroup
	shutdownCh chan struct{}
}

// Site represents a single site being watched
type Site struct {
	Domain       string
	Path         string
	Watcher      *FileWatcher
	Clients      map[*websocket.Conn]bool
	ClientsMu    sync.RWMutex
	BroadcastCh  chan *ReloadMessage
	StopCh       chan struct{}
	LastActivity time.Time
}

// ReloadMessage is sent to clients when files change
type ReloadMessage struct {
	Type string `json:"type"` // "css" or "reload"
	File string `json:"file"` // relative file path
}

// NewSiteManager creates a new site manager
func NewSiteManager(logger *zap.Logger, config *HotReloader) *SiteManager {
	sm := &SiteManager{
		logger:     logger,
		config:     config,
		sites:      make(map[string]*Site),
		shutdownCh: make(chan struct{}),
	}

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		sm.idleCleanupLoop()
	}()

	return sm
}

// EnsureSiteWatched ensures a site is being watched
func (sm *SiteManager) EnsureSiteWatched(domain, sitePath string) {
	domain = strings.Split(domain, ":")[0]

	sm.mu.Lock()
	if existingSite, exists := sm.sites[domain]; exists {
		existingSite.LastActivity = time.Now()
		sm.mu.Unlock()
		return // already watching
	}

	sm.logger.Info("discovered new site",
		zap.String("domain", domain),
		zap.String("path", sitePath),
	)

	// Create site
	site := &Site{
		Domain:       domain,
		Path:         sitePath,
		Clients:      make(map[*websocket.Conn]bool),
		BroadcastCh:  make(chan *ReloadMessage, 10),
		StopCh:       make(chan struct{}),
		LastActivity: time.Now(),
	}

	// Create file watcher
	watcher, err := NewFileWatcher(sitePath, sm.config, sm.logger)
	if err != nil {
		sm.mu.Unlock()
		sm.logger.Error("failed to create watcher",
			zap.String("domain", domain),
			zap.Error(err),
		)
		return
	}

	site.Watcher = watcher
	sm.sites[domain] = site
	sm.mu.Unlock()

	// Start watcher
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		site.Watcher.Watch(site.BroadcastCh, site.StopCh)
	}()

	// Start broadcaster
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		sm.broadcastLoop(site)
	}()

	sm.logger.Info("site watcher started",
		zap.String("domain", domain),
		zap.Int("watch_dirs", len(site.Watcher.watchedDirs)),
	)
}

// broadcastLoop listens for reload messages and broadcasts to all clients
func (sm *SiteManager) broadcastLoop(site *Site) {
	for {
		select {
		case msg := <-site.BroadcastCh:
			sm.mu.Lock()
			site.LastActivity = time.Now()
			sm.mu.Unlock()

			site.ClientsMu.RLock()
			clientCount := len(site.Clients)
			site.ClientsMu.RUnlock()

			if clientCount == 0 {
				continue // no clients connected
			}

			sm.logger.Debug("broadcasting reload",
				zap.String("domain", site.Domain),
				zap.String("type", msg.Type),
				zap.String("file", msg.File),
				zap.Int("clients", clientCount),
			)

			site.ClientsMu.RLock()
			for client := range site.Clients {
				// Send in goroutine to avoid blocking
				go func(c *websocket.Conn) {
					if err := c.WriteJSON(msg); err != nil {
						sm.logger.Debug("failed to send message to client",
							zap.Error(err),
						)
					}
				}(client)
			}
			site.ClientsMu.RUnlock()

		case <-site.StopCh:
			return
		case <-sm.shutdownCh:
			return
		}
	}
}

// HandleWebSocket handles WebSocket connections
func (sm *SiteManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) error {
	domain := strings.Split(r.Host, ":")[0]

	sm.mu.RLock()
	site, exists := sm.sites[domain]
	sm.mu.RUnlock()

	if !exists {
		sm.logger.Debug("websocket request for unwatched site",
			zap.String("domain", domain),
		)
		http.Error(w, "Site not found", http.StatusNotFound)
		return nil
	}

	// Upgrade connection
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in dev mode
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		sm.logger.Error("failed to upgrade websocket",
			zap.Error(err),
		)
		return err
	}

	// Register client
	site.ClientsMu.Lock()
	site.Clients[conn] = true
	clientCount := len(site.Clients)
	site.ClientsMu.Unlock()

	sm.mu.Lock()
	site.LastActivity = time.Now()
	sm.mu.Unlock()

	sm.logger.Info("client connected",
		zap.String("domain", domain),
		zap.Int("total_clients", clientCount),
	)

	// Wait for client to disconnect
	go func() {
		defer func() {
			site.ClientsMu.Lock()
			delete(site.Clients, conn)
			clientCount := len(site.Clients)
			site.ClientsMu.Unlock()

			sm.mu.Lock()
			site.LastActivity = time.Now()
			sm.mu.Unlock()

			conn.Close()

			sm.logger.Info("client disconnected",
				zap.String("domain", domain),
				zap.Int("remaining_clients", clientCount),
			)
		}()

		// Read messages (we don't expect any, but need to detect disconnect)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()

	return nil
}

func (sm *SiteManager) idleCleanupLoop() {
	interval := time.Minute
	if sm.config.idleTimeoutDur > 0 && sm.config.idleTimeoutDur < 3*time.Minute {
		interval = sm.config.idleTimeoutDur / 3
		if interval < 10*time.Second {
			interval = 10 * time.Second
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanupIdleSites()
		case <-sm.shutdownCh:
			return
		}
	}
}

func (sm *SiteManager) cleanupIdleSites() {
	now := time.Now()
	idleTimeout := sm.config.idleTimeoutDur
	if idleTimeout <= 0 {
		return
	}

	type candidate struct {
		domain string
		idle   time.Duration
	}
	candidates := make([]candidate, 0)

	sm.mu.RLock()
	for domain, site := range sm.sites {
		site.ClientsMu.RLock()
		clientCount := len(site.Clients)
		site.ClientsMu.RUnlock()

		if clientCount > 0 {
			continue
		}

		idleFor := now.Sub(site.LastActivity)
		if idleFor >= idleTimeout {
			candidates = append(candidates, candidate{domain: domain, idle: idleFor})
		}
	}
	sm.mu.RUnlock()

	for _, c := range candidates {
		sm.removeSite(c.domain, "idle timeout", c.idle)
	}
}

func (sm *SiteManager) removeSite(domain, reason string, idleFor time.Duration) {
	sm.mu.Lock()
	site, exists := sm.sites[domain]
	if !exists {
		sm.mu.Unlock()
		return
	}
	delete(sm.sites, domain)
	sm.mu.Unlock()

	close(site.StopCh)

	if site.Watcher != nil {
		site.Watcher.Close()
	}

	site.ClientsMu.Lock()
	for conn := range site.Clients {
		conn.Close()
	}
	site.ClientsMu.Unlock()

	sm.logger.Info("site watcher stopped",
		zap.String("domain", domain),
		zap.String("reason", reason),
		zap.Duration("idle_for", idleFor),
	)
}

// Shutdown gracefully shuts down all watchers
func (sm *SiteManager) Shutdown() error {
	sm.logger.Info("shutting down site manager")

	close(sm.shutdownCh)

	sm.mu.RLock()
	domains := make([]string, 0, len(sm.sites))
	for domain := range sm.sites {
		domains = append(domains, domain)
	}
	sm.mu.RUnlock()

	for _, domain := range domains {
		sm.removeSite(domain, "shutdown", 0)
	}

	// Wait for all goroutines to finish
	sm.wg.Wait()

	sm.logger.Info("site manager shutdown complete")
	return nil
}
