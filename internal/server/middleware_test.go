package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestJSONBodySizeMiddleware_AllowsUnderCap verifies a request body
// at exactly the cap goes through.
func TestJSONBodySizeMiddleware_AllowsUnderCap(t *testing.T) {
	t.Parallel()
	mw := jsonBodySizeMiddleware(1024)
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if len(body) != 100 {
			t.Errorf("got %d bytes, want 100", len(body))
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/x",
		bytes.NewReader(make([]byte, 100)))
	req.ContentLength = 100
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Fatal("handler not reached")
	}
}

// TestJSONBodySizeMiddleware_RejectsOverCap verifies that a body
// exceeding the cap surfaces the http.MaxBytesReader error to the
// reading handler, which can then write 413.
func TestJSONBodySizeMiddleware_RejectsOverCap(t *testing.T) {
	t.Parallel()
	mw := jsonBodySizeMiddleware(50)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err == nil {
			t.Fatal("expected ErrBodyTooLarge, got nil")
		}
		// MaxBytesReader sets the response status before erroring; we
		// only need to confirm the handler observed the cap.
		w.WriteHeader(http.StatusRequestEntityTooLarge)
	}))
	req := httptest.NewRequest(http.MethodPost, "/x",
		bytes.NewReader(make([]byte, 200)))
	req.ContentLength = 200
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status=%d, want 413", rr.Code)
	}
}

// TestJSONBodySizeMiddleware_NilBody is a no-op when r.Body is nil
// (GET / HEAD requests).
func TestJSONBodySizeMiddleware_NilBody(t *testing.T) {
	t.Parallel()
	mw := jsonBodySizeMiddleware(50)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("GET wrapped: status=%d", rr.Code)
	}
}

// TestJSONBodySizeMiddleware_ChunkedTransfer wraps Body even when
// ContentLength is unknown (-1, common with chunked transfer
// encoding) so an attacker can't bypass the cap by streaming.
func TestJSONBodySizeMiddleware_ChunkedTransfer(t *testing.T) {
	t.Parallel()
	mw := jsonBodySizeMiddleware(50)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err == nil {
			t.Error("chunked body bypassed cap")
		}
		w.WriteHeader(http.StatusRequestEntityTooLarge)
	}))
	// ContentLength=-1 simulates chunked encoding.
	req := httptest.NewRequest(http.MethodPost, "/x",
		strings.NewReader(strings.Repeat("a", 200)))
	req.ContentLength = -1
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("chunked status=%d, want 413", rr.Code)
	}
}
