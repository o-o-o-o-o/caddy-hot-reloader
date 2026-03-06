package hotreloader

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/caddyserver/caddy/v2"
)

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
