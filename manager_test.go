package hotreloader

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

type hijackWriter struct{ http.ResponseWriter }

func (hijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) { return nil, nil, nil }

type unwrapOnlyWriter struct{ inner http.ResponseWriter }

func (u unwrapOnlyWriter) Header() http.Header         { return u.inner.Header() }
func (u unwrapOnlyWriter) Write(b []byte) (int, error) { return u.inner.Write(b) }
func (u unwrapOnlyWriter) WriteHeader(code int)        { u.inner.WriteHeader(code) }
func (u unwrapOnlyWriter) Unwrap() http.ResponseWriter { return u.inner }

func TestHijackableWriterUnwraps(t *testing.T) {
	base := hijackWriter{httptest.NewRecorder()}
	wrapped := unwrapOnlyWriter{unwrapOnlyWriter{base}}

	if _, ok := http.ResponseWriter(wrapped).(http.Hijacker); ok {
		t.Fatal("test wrapper must not implement http.Hijacker")
	}
	if _, ok := hijackableWriter(wrapped).(http.Hijacker); !ok {
		t.Fatal("expected the wrapped Hijacker")
	}
}

func TestHijackableWriterNoHijacker(t *testing.T) {
	rec := httptest.NewRecorder()
	if got := hijackableWriter(rec); got != http.ResponseWriter(rec) {
		t.Fatal("expected the original writer when no Hijacker exists")
	}
}
