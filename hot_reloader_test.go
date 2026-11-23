package hotreloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// serveHTML runs an HTML body through the response wrapper and returns the
// written body plus the plugin warnings.
func serveHTML(t *testing.T, h *HotReloader, html string) (string, []observer.LoggedEntry) {
	t.Helper()

	core, logs := observer.New(zapcore.WarnLevel)
	h.logger = zap.New(core)

	rec := httptest.NewRecorder()
	rw := &responseWrapper{ResponseWriter: rec, host: "test.example", hotReloader: h, expectsHTML: true}
	rw.Header().Set("Content-Type", "text/html")
	rw.WriteHeader(200)
	rw.Write([]byte(html))
	rw.Flush()

	return rec.Body.String(), logs.All()
}

func TestHeadScriptFirstInHead(t *testing.T) {
	h := &HotReloader{HeadScript: "/faults.js"}
	got, _ := serveHTML(t, h, `<html><head><script src="/app.js"></script></head><body>x</body></html>`)
	want := `<html><head><script src="/faults.js"></script><script src="/app.js"></script></head><body>x` + clientScript + `</body></html>`
	if got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestHeadScriptWithHeadAttributes(t *testing.T) {
	h := &HotReloader{HeadScript: "/faults.js"}
	got, _ := serveHTML(t, h, `<HEAD lang="x"><title>t</title></HEAD><body></body>`)
	if !strings.HasPrefix(got, `<HEAD lang="x"><script src="/faults.js"></script><title>`) {
		t.Fatalf("head script not first in head: %q", got)
	}
}

func TestNoHeadScriptDirectiveLeavesBodyUnchanged(t *testing.T) {
	h := &HotReloader{}
	html := `<html><head><script src="/app.js"></script></head><body>x</body></html>`
	got, logs := serveHTML(t, h, html)
	want := strings.Replace(html, "</body>", clientScript+"</body>", 1)
	if got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if len(logs) != 0 {
		t.Fatalf("unexpected warnings: %v", logs)
	}
}

func TestHeadScriptWithoutHeadWarnsAndKeepsClient(t *testing.T) {
	h := &HotReloader{HeadScript: "/faults.js"}
	got, logs := serveHTML(t, h, `<p>fragment</p><body>x</body>`)
	if strings.Contains(got, "/faults.js") {
		t.Fatalf("head script injected without <head>: %q", got)
	}
	if !strings.Contains(got, clientScript) {
		t.Fatalf("reload client missing: %q", got)
	}
	if len(logs) != 1 || !strings.Contains(logs[0].Message, "head_script") {
		t.Fatalf("want one head_script warning, got %v", logs)
	}
}

func TestDiscoverSiteFromRequestRoot(t *testing.T) {
	t.Helper()

	h := &HotReloader{}
	req := httptest.NewRequest("GET", "https://test.laac-acc.why/", nil)
	repl := caddy.NewReplacer()
	root := t.TempDir()
	repl.Set("http.vars.root", root)

	ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
	req = req.WithContext(ctx)

	got := h.discoverSiteFromRequest(req)
	want := root
	if got != want {
		t.Fatalf("discoverSiteFromRequest() = %q, want %q", got, want)
	}
}

func TestDiscoverSiteFromRequestRootMissing(t *testing.T) {
	t.Helper()

	h := &HotReloader{}
	req := httptest.NewRequest("GET", "https://test.laac-acc.why/", nil)

	got := h.discoverSiteFromRequest(req)
	if got != "" {
		t.Fatalf("discoverSiteFromRequest() = %q, want empty", got)
	}
}

func TestDiscoverSiteWithBaseDir(t *testing.T) {
	t.Helper()

	h := &HotReloader{BaseDir: "/Users/why/Test Sites"}
	got := h.discoverSite("test.laac-acc.why")
	want := "/Users/why/Test Sites/test/laac-acc/www"
	if got != want {
		t.Fatalf("discoverSite() = %q, want %q", got, want)
	}
}

func TestHeadScriptRejectsExtraArgs(t *testing.T) {
	d := caddyfile.NewTestDispenser("hot_reloader {\n head_script /a.js /b.js\n}")
	if err := (&HotReloader{}).UnmarshalCaddyfile(d); err == nil {
		t.Fatal("want error for extra head_script argument")
	}
}

// headerSpy records the status passed to WriteHeader on the real writer.
type headerSpy struct {
	http.ResponseWriter
	status int
}

func (s *headerSpy) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func TestFlushEmptyHTMLBodyWritesHeader(t *testing.T) {
	h := &HotReloader{logger: zap.NewNop()}
	spy := &headerSpy{ResponseWriter: httptest.NewRecorder()}
	rw := &responseWrapper{ResponseWriter: spy, host: "test.example", hotReloader: h, expectsHTML: true}
	rw.Header().Set("Content-Type", "text/html")
	rw.WriteHeader(200)
	rw.Flush()

	if spy.status != 200 {
		t.Fatalf("WriteHeader not forwarded for empty body, status = %d", spy.status)
	}
	if got := rw.Header().Get("X-Hot-Reload"); got != "enabled" {
		t.Errorf("X-Hot-Reload = %q, want enabled", got)
	}
}

func headFileHotReloader(t *testing.T, js string) (*HotReloader, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "capture.js")
	if err := os.WriteFile(path, []byte(js), 0o644); err != nil {
		t.Fatal(err)
	}
	if !headScriptFile(path) {
		t.Fatalf("%s not detected as file", path)
	}
	return &HotReloader{HeadScript: path, headFile: &headFileCache{path: path}}, path
}

func TestHeadScriptFileInlinedFirstInHead(t *testing.T) {
	h, _ := headFileHotReloader(t, "window.x=1;")
	got, _ := serveHTML(t, h, `<html><head><script src="/app.js"></script></head><body>x</body></html>`)
	want := `<html><head><script>window.x=1;</script><script src="/app.js"></script></head><body>x` + clientScript + `</body></html>`
	if got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestHeadScriptFileReloadsOnMtimeChange(t *testing.T) {
	h, path := headFileHotReloader(t, "window.x=1;")
	serveHTML(t, h, `<head></head><body></body>`)
	if err := os.WriteFile(path, []byte("window.x=22;"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	os.Chtimes(path, later, later)
	got, _ := serveHTML(t, h, `<head></head><body></body>`)
	if !strings.Contains(got, "<script>window.x=22;</script>") {
		t.Fatalf("file change not picked up: %q", got)
	}
}

func TestHeadScriptFileEscapesClosingScript(t *testing.T) {
	h, _ := headFileHotReloader(t, `var s="</script><b>"; var t="<!-- x";`)
	got, _ := serveHTML(t, h, `<head></head><body></body>`)
	if strings.Count(strings.ToLower(got), "</script>") != 2 { // inline + reload client
		t.Fatalf("literal </script> survived: %q", got)
	}
	if !strings.Contains(got, `"<!-- x"`) {
		t.Fatalf("<!-- must stay unchanged: %q", got)
	}
}

func TestHeadScriptFileModeSelection(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "c.js")
	os.WriteFile(file, []byte("x"), 0o644)
	cases := map[string]bool{
		file:                          true,
		dir:                           false, // directory
		filepath.Join(dir, "missing"): false,
		"relative/c.js":               false,
		"/faults.js":                  false, // root-relative URL
		"https://h.example/c.js":      false,
	}
	for v, want := range cases {
		if got := headScriptFile(v); got != want {
			t.Errorf("headScriptFile(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestProvisionSelectsFileMode(t *testing.T) {
	file := filepath.Join(t.TempDir(), "c.js")
	os.WriteFile(file, []byte("x"), 0o644)
	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()
	for v, wantFile := range map[string]bool{file: true, "/faults.js": false} {
		h := &HotReloader{HeadScript: v}
		if err := h.Provision(ctx); err != nil {
			t.Fatal(err)
		}
		if (h.headFile != nil) != wantFile {
			t.Errorf("head_script %q: file mode = %v, want %v", v, h.headFile != nil, wantFile)
		}
	}
}

func TestHeadScriptFileUnreadableWarnsOnce(t *testing.T) {
	h, path := headFileHotReloader(t, "x")
	os.Remove(path)
	core, logs := observer.New(zapcore.WarnLevel)
	h.logger = zap.New(core)
	for i := 0; i < 3; i++ {
		rw := &responseWrapper{ResponseWriter: httptest.NewRecorder(), host: "t", hotReloader: h, expectsHTML: true}
		rw.Header().Set("Content-Type", "text/html")
		rw.WriteHeader(200)
		rw.Write([]byte(`<head></head><body></body>`))
		rw.Flush()
	}
	if n := logs.Len(); n != 1 {
		t.Fatalf("want 1 warning for 3 responses, got %d", n)
	}
	os.WriteFile(path, []byte("y"), 0o644)
	if got, _ := serveHTML(t, h, `<head></head><body></body>`); !strings.Contains(got, "<script>y</script>") {
		t.Fatalf("file not injected after it became readable: %q", got)
	}
}

func TestHeadScriptFileMissingWarnsAndKeepsClient(t *testing.T) {
	h, path := headFileHotReloader(t, "x")
	os.Remove(path)
	got, logs := serveHTML(t, h, `<head></head><body></body>`)
	if len(logs) != 1 || !strings.Contains(logs[0].Message, "head_script") {
		t.Fatalf("want one head_script warning, got %v", logs)
	}
	if !strings.Contains(got, clientScript) {
		t.Fatal("reload client missing")
	}
}
