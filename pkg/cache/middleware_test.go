package cache

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCacheMiddleware(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"GETCachedOnSecondCall", testGETCachedOnSecondCall},
		{"POSTNotCached", testPOSTNotCached},
		{"Non200NotCached", testNon200NotCached},
		{"XCacheHeaderSet", testXCacheHeaderSet},
		{"DifferentURLsCachedSeparately", testDifferentURLsCachedSeparately},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func testGETCachedOnSecondCall(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":"hello"}`))
	})

	c := NewLRUCache(10, 5*time.Second)
	wrapped := CacheMiddleware(c)(handler)

	// First request: MISS.
	req1 := httptest.NewRequest(http.MethodGet, "/api/plugins", nil)
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)

	if callCount != 1 {
		t.Fatalf("expected handler called once, got %d", callCount)
	}
	if rec1.Header().Get("X-Cache") != "MISS" {
		t.Fatalf("expected X-Cache: MISS, got %q", rec1.Header().Get("X-Cache"))
	}

	// Second request: HIT.
	req2 := httptest.NewRequest(http.MethodGet, "/api/plugins", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	if callCount != 1 {
		t.Fatalf("expected handler not called again, got %d", callCount)
	}
	if rec2.Header().Get("X-Cache") != "HIT" {
		t.Fatalf("expected X-Cache: HIT, got %q", rec2.Header().Get("X-Cache"))
	}

	body, _ := io.ReadAll(rec2.Result().Body)
	if string(body) != `{"data":"hello"}` {
		t.Fatalf("expected cached body, got %q", string(body))
	}
}

func testPOSTNotCached(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	})

	c := NewLRUCache(10, 5*time.Second)
	wrapped := CacheMiddleware(c)(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/plugins", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if callCount != 1 {
		t.Fatalf("expected handler called once, got %d", callCount)
	}

	// Cache should be empty since POST requests are not cached.
	if c.Size() != 0 {
		t.Fatalf("expected cache size 0 for POST, got %d", c.Size())
	}

	// No X-Cache header on non-GET.
	if rec.Header().Get("X-Cache") != "" {
		t.Fatalf("expected no X-Cache header on POST, got %q", rec.Header().Get("X-Cache"))
	}
}

func testNon200NotCached(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	})

	c := NewLRUCache(10, 5*time.Second)
	wrapped := CacheMiddleware(c)(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/unknown/capabilities", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	// Cache should be empty since non-200 responses are not cached.
	if c.Size() != 0 {
		t.Fatalf("expected cache size 0 for non-200, got %d", c.Size())
	}

	// Second request should still call the handler.
	req2 := httptest.NewRequest(http.MethodGet, "/api/plugins/unknown/capabilities", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	if callCount != 2 {
		t.Fatalf("expected handler called twice, got %d", callCount)
	}
}

func testXCacheHeaderSet(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`ok`))
	})

	c := NewLRUCache(10, 5*time.Second)
	wrapped := CacheMiddleware(c)(handler)

	// First: MISS.
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)

	if rec1.Header().Get("X-Cache") != "MISS" {
		t.Fatalf("expected X-Cache: MISS on first call, got %q", rec1.Header().Get("X-Cache"))
	}

	// Second: HIT.
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	if rec2.Header().Get("X-Cache") != "HIT" {
		t.Fatalf("expected X-Cache: HIT on second call, got %q", rec2.Header().Get("X-Cache"))
	}
}

func testDifferentURLsCachedSeparately(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.URL.Path))
	})

	c := NewLRUCache(10, 5*time.Second)
	wrapped := CacheMiddleware(c)(handler)

	// Request to /a.
	req1 := httptest.NewRequest(http.MethodGet, "/a", nil)
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)

	// Request to /b.
	req2 := httptest.NewRequest(http.MethodGet, "/b", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	// Both should be MISS.
	if rec1.Header().Get("X-Cache") != "MISS" || rec2.Header().Get("X-Cache") != "MISS" {
		t.Fatal("expected both first requests to be MISS")
	}

	// Request /a again: HIT with correct body.
	req3 := httptest.NewRequest(http.MethodGet, "/a", nil)
	rec3 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec3, req3)

	body, _ := io.ReadAll(rec3.Result().Body)
	if string(body) != "/a" {
		t.Fatalf("expected cached body /a, got %q", string(body))
	}

	if c.Size() != 2 {
		t.Fatalf("expected 2 cached entries, got %d", c.Size())
	}
}
