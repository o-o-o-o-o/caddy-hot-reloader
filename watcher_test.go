package hotreloader

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMatchesExtensionEmptyAllowsAll(t *testing.T) {
	fw := &FileWatcher{config: &HotReloader{}}

	cases := []string{"index.html", "style.css", "noext"}
	for _, name := range cases {
		if !fw.matchesExtension(name) {
			t.Errorf("matchesExtension(%q) = false, want true (empty Extensions should allow all)", name)
		}
	}
}

func TestMatchesExtensionFilters(t *testing.T) {
	fw := &FileWatcher{config: &HotReloader{Extensions: []string{"html", "css"}}}

	allowed := []string{"a.html", "a.CSS"}
	for _, name := range allowed {
		if !fw.matchesExtension(name) {
			t.Errorf("matchesExtension(%q) = false, want true", name)
		}
	}

	if fw.matchesExtension("a.php") {
		t.Errorf("matchesExtension(%q) = true, want false", "a.php")
	}
}

func TestIsEditorJunk(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"4913", true},
		{".DS_Store", true},
		{"index.html~", true},
		{"foo.swp", true},
		{"FOO.SWO", true},
		{"#index.html#", true},
		{".#index.html", true},
		{"index.html", false},
		{"style.css", false},
		{"4913.html", false},
	}

	for _, c := range cases {
		if got := isEditorJunk(c.name); got != c.want {
			t.Errorf("isEditorJunk(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestCoalescePending(t *testing.T) {
	t.Run("empty map returns nil", func(t *testing.T) {
		if msg := coalescePending(map[string]string{}); msg != nil {
			t.Errorf("coalescePending(empty) = %+v, want nil", msg)
		}
	})

	t.Run("reload wins over css", func(t *testing.T) {
		pending := map[string]string{
			"a.html": "reload",
			"b.css":  "css",
		}
		msg := coalescePending(pending)
		if msg == nil {
			t.Fatal("coalescePending(...) = nil, want non-nil")
		}
		if msg.Type != "reload" || msg.File != "a.html" {
			t.Errorf("coalescePending(...) = %+v, want {Type: reload, File: a.html}", msg)
		}
	})

	t.Run("single css entry", func(t *testing.T) {
		pending := map[string]string{"a.css": "css"}
		msg := coalescePending(pending)
		if msg == nil {
			t.Fatal("coalescePending(...) = nil, want non-nil")
		}
		if msg.Type != "css" || msg.File != "a.css" {
			t.Errorf("coalescePending(...) = %+v, want {Type: css, File: a.css}", msg)
		}
	})

	t.Run("multiple css entries", func(t *testing.T) {
		pending := map[string]string{
			"a.css": "css",
			"b.css": "css",
		}
		msg := coalescePending(pending)
		if msg == nil {
			t.Fatal("coalescePending(...) = nil, want non-nil")
		}
		if msg.Type != "css" || msg.File != "" {
			t.Errorf("coalescePending(...) = %+v, want {Type: css, File: \"\"}", msg)
		}
	})
}

func TestWatcherDebouncesBurst(t *testing.T) {
	tmp := t.TempDir()
	cfg := &HotReloader{}

	fw, err := NewFileWatcher(tmp, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	ch := make(chan *ReloadMessage, 10)
	stop := make(chan struct{})

	go fw.Watch(ch, stop)

	// Let the watch loop start before generating events.
	time.Sleep(200 * time.Millisecond)

	for i := 0; i < 3; i++ {
		name := filepath.Join(tmp, "file"+string(rune('a'+i))+".html")
		if err := os.WriteFile(name, []byte("content"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}

	select {
	case msg := <-ch:
		if msg.Type != "reload" {
			t.Errorf("message Type = %q, want %q", msg.Type, "reload")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for debounced reload message")
	}

	select {
	case msg := <-ch:
		t.Errorf("received unexpected second message: %+v", msg)
	case <-time.After(400 * time.Millisecond):
		// expected: no second message
	}

	close(stop)
	if err := fw.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestWatcherIgnoresEditorJunk(t *testing.T) {
	tmp := t.TempDir()
	cfg := &HotReloader{}

	fw, err := NewFileWatcher(tmp, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	ch := make(chan *ReloadMessage, 10)
	stop := make(chan struct{})

	go fw.Watch(ch, stop)

	time.Sleep(200 * time.Millisecond)

	junkNames := []string{"4913", "page.html~"}
	for _, name := range junkNames {
		path := filepath.Join(tmp, name)
		if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}

	select {
	case msg := <-ch:
		t.Errorf("received unexpected message for editor junk: %+v", msg)
	case <-time.After(500 * time.Millisecond):
		// expected: no message
	}

	close(stop)
	if err := fw.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestWatcherWatchesNewSubdirectory(t *testing.T) {
	tmp := t.TempDir()
	cfg := &HotReloader{}

	fw, err := NewFileWatcher(tmp, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}

	ch := make(chan *ReloadMessage, 10)
	stop := make(chan struct{})

	go fw.Watch(ch, stop)

	time.Sleep(200 * time.Millisecond)

	sub := filepath.Join(tmp, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir(%s) error = %v", sub, err)
	}

	// Give the watch loop time to pick up the new directory and add a watch on it.
	time.Sleep(300 * time.Millisecond)

	newFile := filepath.Join(sub, "new.html")
	if err := os.WriteFile(newFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", newFile, err)
	}

	select {
	case msg := <-ch:
		if msg.Type != "reload" {
			t.Errorf("message Type = %q, want %q", msg.Type, "reload")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for reload message from new subdirectory")
	}

	close(stop)
	if err := fw.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
