package hotreloader

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(HotReloader{})
	httpcaddyfile.RegisterHandlerDirective("hot_reloader", parseCaddyfile)
}

// HotReloader implements hot-reloading functionality for Caddy
type HotReloader struct {
	// Configuration
	BaseDir          string   `json:"base_dir,omitempty"`
	Watch            []string `json:"watch,omitempty"`
	Exclude          []string `json:"exclude,omitempty"`
	Extensions       []string `json:"extensions,omitempty"`
	RespectGitignore bool     `json:"respect_gitignore,omitempty"`
	IdleTimeout      string   `json:"idle_timeout,omitempty"`

	// Runtime state
	logger         *zap.Logger
	manager        *SiteManager
	mu             sync.RWMutex
	idleTimeoutDur time.Duration
}

// CaddyModule returns the Caddy module information
func (HotReloader) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.hot_reloader",
		New: func() caddy.Module { return new(HotReloader) },
	}
}

// Provision sets up the hot reloader
func (h *HotReloader) Provision(ctx caddy.Context) error {
	h.logger = ctx.Logger(h)

	// Set defaults
	if len(h.Watch) == 0 {
		h.Watch = []string{}  // Empty = watch entire base directory (more flexible)
	}
	if len(h.Exclude) == 0 {
		h.Exclude = []string{"**.cache", "**/vendor/**", "**/node_modules/**", "**/.DS_Store"}
	}
	if len(h.Extensions) == 0 {
		h.Extensions = []string{}  // Empty = all extensions trigger reload
	}
	if !h.RespectGitignore {
		h.RespectGitignore = true // default to true
	}
	if h.IdleTimeout == "" {
		h.IdleTimeout = "30m"
	}
	parsedIdleTimeout, parseErr := time.ParseDuration(h.IdleTimeout)
	if parseErr != nil || parsedIdleTimeout <= 0 {
		h.logger.Warn("invalid idle_timeout, falling back to 30m",
			zap.String("idle_timeout", h.IdleTimeout),
			zap.Error(parseErr),
		)
		parsedIdleTimeout = 30 * time.Minute
		h.IdleTimeout = "30m"
	}
	h.idleTimeoutDur = parsedIdleTimeout

	// Initialize site manager
	h.manager = NewSiteManager(h.logger, h)

	h.logger.Info("hot_reloader initialized",
		zap.String("base_dir", h.BaseDir),
		zap.Strings("watch", h.Watch),
		zap.String("watch_behavior", func() string {
			if len(h.Watch) == 0 {
				return "watching entire directory (no patterns specified)"
			}
			return "watching specific patterns"
		}()),
		zap.Strings("exclude", h.Exclude),
		zap.Strings("extensions", h.Extensions),
		zap.Bool("respect_gitignore", h.RespectGitignore),
		zap.String("idle_timeout", h.IdleTimeout),
	)

	return nil
}

// Validate validates the configuration
func (h *HotReloader) Validate() error {
	// base_dir is optional - only needed if using domain-based site discovery
	// Users providing explicit roots in Caddyfile don't need it
	return nil
}

// Cleanup cleans up resources when Caddy is shutting down
func (h *HotReloader) Cleanup() error {
	h.logger.Info("hot_reloader shutting down")
	if h.manager != nil {
		return h.manager.Shutdown()
	}
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler
func (h *HotReloader) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Handle WebSocket upgrade for hot-reload endpoint
	if r.URL.Path == "/hot-reload" {
		return h.handleWebSocket(w, r)
	}

	// Discover site from request (request-time discovery)
	sitePath := h.discoverSite(r.Host)
	if sitePath != "" {
		h.manager.EnsureSiteWatched(r.Host, sitePath)
	}

	// Wrap response writer to inject client script
	wrapper := &responseWrapper{
		ResponseWriter: w,
		host:           r.Host,
		hotReloader:    h,
		expectsHTML:    strings.Contains(r.Header.Get("Accept"), "text/html") || strings.Contains(r.Header.Get("Accept"), "application/xhtml+xml"),
		method:         r.Method,
	}

	// Flush buffered response after handler completes
	defer wrapper.Flush()

	return next.ServeHTTP(wrapper, r)
}

// discoverSite extracts site path from domain using *.*.domain pattern and base_dir
// Example: test.subdomain.example.com -> /path/to/sites/test/subdomain/www
// Returns empty string if base_dir is not set or domain doesn't match pattern
func (h *HotReloader) discoverSite(host string) string {
	// base_dir is required for this feature
	if h.BaseDir == "" {
		return ""
	}

	// Remove port if present
	host = strings.Split(host, ":")[0]

	// Split by dots
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		// Only process multi-level subdomains
		return ""
	}

	// Extract labels: test.subdomain.example.com -> parts[0]=test, parts[1]=subdomain
	label1 := parts[0]
	label2 := parts[1]

	// Construct path: {base_dir}/{label1}/{label2}/www
	sitePath := filepath.Join(h.BaseDir, label1, label2, "www")

	return sitePath
}

// handleWebSocket handles WebSocket connections for hot reload
func (h *HotReloader) handleWebSocket(w http.ResponseWriter, r *http.Request) error {
	return h.manager.HandleWebSocket(w, r)
}

// responseWrapper wraps http.ResponseWriter to inject client script
// It buffers HTML responses to inject the script cleanly
type responseWrapper struct {
	http.ResponseWriter
	host         string
	hotReloader  *HotReloader
	expectsHTML  bool
	method       string
	wroteHeader  bool
	statusCode   int
	buffer       []byte
	shouldInject bool
}

func (rw *responseWrapper) WriteHeader(statusCode int) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
		rw.statusCode = statusCode

		// Only inject into HTML responses
		contentType := rw.Header().Get("Content-Type")
		if statusCode == 200 && (strings.Contains(contentType, "text/html") || (contentType == "" && rw.expectsHTML)) {
			rw.shouldInject = true
			// Remove Content-Length header since we'll be modifying the response body
			rw.Header().Del("Content-Length")
			// Add header to indicate script will be injected
			rw.Header().Set("X-Hot-Reload", "enabled")
			rw.hotReloader.logger.Debug("buffering HTML for injection",
				zap.String("host", rw.host),
				zap.String("content_type", contentType))
			// Don't write header yet - we'll do it after injection
			return
		}
	}
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWrapper) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(200)
	}

	// If we should inject, buffer the response
	if rw.shouldInject {
		rw.buffer = append(rw.buffer, b...)
		return len(b), nil
	}

	return rw.ResponseWriter.Write(b)
}

// Flush writes the buffered response with injected script
func (rw *responseWrapper) Flush() {
	if rw.shouldInject && len(rw.buffer) > 0 {
		body := string(rw.buffer)

		// Inject script before </body>
		if idx := strings.LastIndex(body, "</body>"); idx != -1 {
			injected := body[:idx] + clientScript + body[idx:]
			rw.hotReloader.logger.Info("injected hot reload script",
				zap.String("host", rw.host),
				zap.Int("original_size", len(rw.buffer)),
				zap.Int("injected_size", len(injected)))

			// Now write header and body
			rw.ResponseWriter.WriteHeader(rw.statusCode)
			rw.ResponseWriter.Write([]byte(injected))
		} else {
			// No </body> found, write as-is
			rw.hotReloader.logger.Debug("no </body> tag found, writing original response",
				zap.String("host", rw.host),
				zap.Int("size", len(rw.buffer)))
			rw.ResponseWriter.WriteHeader(rw.statusCode)
			rw.ResponseWriter.Write(rw.buffer)
		}
	} else if rw.shouldInject && len(rw.buffer) == 0 {
		if rw.method == http.MethodHead {
			return
		}
		// We were supposed to inject but buffer is empty - this is an error case
		rw.hotReloader.logger.Warn("expected to inject script but buffer is empty",
			zap.String("host", rw.host))
	}
	// If !shouldInject, the response was already written directly (non-HTML responses)
}

// parseCaddyfile unmarshals tokens from Caddyfile into the module config
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var hr HotReloader
	err := hr.UnmarshalCaddyfile(h.Dispenser)
	return &hr, err
}

// UnmarshalCaddyfile sets up the hot reloader from Caddyfile tokens
func (h *HotReloader) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "base_dir":
				if !d.NextArg() {
					return d.ArgErr()
				}
				h.BaseDir = d.Val()
			case "watch":
				h.Watch = d.RemainingArgs()
				if len(h.Watch) == 0 {
					return d.ArgErr()
				}
			case "exclude":
				h.Exclude = d.RemainingArgs()
				if len(h.Exclude) == 0 {
					return d.ArgErr()
				}
			case "extensions":
				h.Extensions = d.RemainingArgs()
				if len(h.Extensions) == 0 {
					return d.ArgErr()
				}
			case "respect_gitignore":
				if !d.NextArg() {
					return d.ArgErr()
				}
				val := d.Val()
				h.RespectGitignore = val == "true" || val == "yes" || val == "1"
			case "idle_timeout":
				if !d.NextArg() {
					return d.ArgErr()
				}
				h.IdleTimeout = d.Val()
			default:
				return d.Errf("unrecognized subdirective: %s", d.Val())
			}
		}
	}
	return nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*HotReloader)(nil)
	_ caddy.Validator             = (*HotReloader)(nil)
	_ caddy.CleanerUpper          = (*HotReloader)(nil)
	_ caddyhttp.MiddlewareHandler = (*HotReloader)(nil)
	_ caddyfile.Unmarshaler       = (*HotReloader)(nil)
)

// clientScript is the JavaScript injected into HTML pages
const clientScript = `<script>
(function() {
	'use strict';
	var ws, reconnectTimeout, reconnectDelay = 1000;
	var maxReconnectDelay = 8000;
	var isTabHidden = false;

	// Track visibility to stop reconnecting when tab is closed/hidden
	document.addEventListener('visibilitychange', function() {
		isTabHidden = document.hidden;
		if (isTabHidden && ws) {
			ws.close();
		}
	});

	function connect() {
		if (isTabHidden) return;

		var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		var url = protocol + '//' + window.location.host + '/hot-reload';
		
		ws = new WebSocket(url);
		
		ws.onopen = function() {
			console.log('[Caddy 🔄] Connected');
			reconnectDelay = 1000; // reset delay on successful connection
		};
		
		ws.onmessage = function(event) {
			try {
				var data = JSON.parse(event.data);
				console.log('[Caddy 🔄] Received:', data);
				
				if (data.type === 'css') {
					// Inject CSS without full reload
					reloadCSS(data.file);
				} else if (data.type === 'reload') {
					// Full page reload
					console.log('[Caddy 🔄] Reloading page...');
					window.location.reload();
				}
			} catch (e) {
				console.error('[Caddy 🔄] Error parsing message:', e);
			}
		};
		
		ws.onclose = function() {
			console.log('[Caddy 🔄] Disconnected');
			if (!isTabHidden) {
				// Exponential backoff
				reconnectTimeout = setTimeout(connect, reconnectDelay);
				reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
			}
		};
		
		ws.onerror = function(error) {
			console.error('[Caddy 🔄] WebSocket error:', error);
		};
	}

	function reloadCSS(changedFile) {
		console.log('[Caddy 🔄] Reloading CSS:', changedFile);
		var links = document.querySelectorAll('link[rel="stylesheet"]');
		var reloaded = false;
		
		links.forEach(function(link) {
			var href = link.getAttribute('href');
			if (href && (href.indexOf(changedFile) !== -1 || !changedFile)) {
				// Add cache-busting parameter
				var url = new URL(href, window.location.origin);
				url.searchParams.set('_reload', Date.now());
				link.setAttribute('href', url.toString());
				reloaded = true;
			}
		});
		
		if (!reloaded) {
			// If we couldn't match the file, reload all CSS
			links.forEach(function(link) {
				var href = link.getAttribute('href');
				if (href) {
					var url = new URL(href, window.location.origin);
					url.searchParams.set('_reload', Date.now());
					link.setAttribute('href', url.toString());
				}
			});
		}
	}

	// Start connection
	connect();
})();
</script>`
