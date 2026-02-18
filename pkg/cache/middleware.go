package cache

import (
	"bytes"
	"net/http"
)

// cacheResponseWriter wraps http.ResponseWriter to capture the response body
// and status code so they can be stored in the cache.
type cacheResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
	written    bool
}

func (w *cacheResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *cacheResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.statusCode = http.StatusOK
		w.written = true
	}
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// CacheMiddleware returns HTTP middleware that caches GET responses in the
// provided LRUCache. The cache key is the full request URL (path + query).
//
// Behavior:
//   - Only GET requests are cached; all other methods pass through.
//   - On cache hit: the cached body is written with its original Content-Type
//     and a 200 status. An X-Cache: HIT header is added.
//   - On cache miss: the handler is called; if it returns 200, the response
//     body is stored in the cache. An X-Cache: MISS header is added.
//   - Non-200 responses are never cached.
func CacheMiddleware(c *LRUCache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests.
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			key := r.URL.RequestURI()

			// Check cache.
			if cached, ok := c.Get(key); ok {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "HIT")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(cached)
				return
			}

			// Cache miss: capture response.
			crw := &cacheResponseWriter{
				ResponseWriter: w,
			}
			crw.Header().Set("X-Cache", "MISS")
			next.ServeHTTP(crw, r)

			// Only cache 200 responses.
			if crw.statusCode == http.StatusOK {
				c.Set(key, crw.body.Bytes())
			}
		})
	}
}
