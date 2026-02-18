package conformance

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// GetJSON performs a GET request and decodes the JSON response into v.
func GetJSON(t *testing.T, serverURL, path string, v any) {
	t.Helper()
	resp, err := http.Get(serverURL + path)
	if err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s returned %d: %s", path, resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("GET %s: decode error: %v", path, err)
	}
}

// WaitForReady waits until the server responds OK on /readyz.
func WaitForReady(t *testing.T, serverURL string) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 30; i++ {
		resp, err := client.Get(serverURL + "/readyz")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("server not ready after 30 seconds")
}
