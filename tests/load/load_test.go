// Package load provides load tests for validating SLO targets.
// These tests require a running catalog server (CATALOG_SERVER_URL env var)
// and are intended to be run manually or in a CI load testing stage.
//
// Run with: CATALOG_SERVER_URL=http://localhost:8080 go test ./tests/load/... -v -count=1 -timeout 5m
package load

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"testing"
	"time"
)

var serverURL = os.Getenv("CATALOG_SERVER_URL")

func waitForReady(t *testing.T) {
	t.Helper()
	for i := 0; i < 30; i++ {
		resp, err := http.Get(serverURL + "/readyz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("server did not become ready within 15 seconds")
}

// latencyStats collects request latencies and computes percentiles.
type latencyStats struct {
	mu        sync.Mutex
	latencies []time.Duration
	errors    int
}

func (s *latencyStats) record(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latencies = append(s.latencies, d)
}

func (s *latencyStats) recordError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors++
}

func (s *latencyStats) percentile(p float64) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.latencies) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(s.latencies))
	copy(sorted, s.latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func (s *latencyStats) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.latencies)
}

func (s *latencyStats) errorCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.errors
}

func (s *latencyStats) report() string {
	return fmt.Sprintf(
		"total=%d errors=%d p50=%v p95=%v p99=%v",
		s.count(), s.errorCount(),
		s.percentile(0.50),
		s.percentile(0.95),
		s.percentile(0.99),
	)
}

// runLoadTest executes concurrent requests against a URL and collects latency.
func runLoadTest(t *testing.T, url string, concurrency, totalRequests int) *latencyStats {
	t.Helper()
	stats := &latencyStats{}
	requests := make(chan struct{}, totalRequests)
	for i := 0; i < totalRequests; i++ {
		requests <- struct{}{}
	}
	close(requests)

	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{Timeout: 10 * time.Second}
			for range requests {
				start := time.Now()
				resp, err := client.Get(url)
				elapsed := time.Since(start)
				if err != nil {
					stats.recordError()
					continue
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					stats.record(elapsed)
				} else {
					stats.recordError()
				}
			}
		}()
	}

	wg.Wait()
	return stats
}

// TestLoadPluginsList validates p95 latency SLO for /api/plugins.
// SLO target: p95 <= 300ms.
func TestLoadPluginsList(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	stats := runLoadTest(t, serverURL+"/api/plugins", 10, 200)
	t.Logf("/api/plugins load: %s", stats.report())

	p95 := stats.percentile(0.95)
	if p95 > 300*time.Millisecond {
		t.Errorf("p95 latency %v exceeds 300ms SLO", p95)
	}
	if stats.errorCount() > 0 {
		t.Errorf("had %d errors out of %d requests", stats.errorCount(), stats.count()+stats.errorCount())
	}
}

// TestLoadCapabilities validates p95 latency SLO for /api/plugins/{name}/capabilities.
// SLO target: p95 <= 300ms.
func TestLoadCapabilities(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	// Discover a plugin.
	resp, err := http.Get(serverURL + "/api/plugins")
	if err != nil {
		t.Fatalf("GET /api/plugins failed: %v", err)
	}
	defer resp.Body.Close()

	var response struct {
		Plugins []struct {
			Name string `json:"name"`
		} `json:"plugins"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(response.Plugins) == 0 {
		t.Skip("no plugins available")
	}

	capURL := serverURL + "/api/plugins/" + response.Plugins[0].Name + "/capabilities"
	stats := runLoadTest(t, capURL, 10, 200)
	t.Logf("capabilities load: %s", stats.report())

	p95 := stats.percentile(0.95)
	if p95 > 300*time.Millisecond {
		t.Errorf("p95 latency %v exceeds 300ms SLO", p95)
	}
}

// TestLoadEntityList validates p95 latency SLO for entity list endpoints.
// SLO target: p95 <= 300ms.
func TestLoadEntityList(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	// Discover entity list endpoints.
	resp, err := http.Get(serverURL + "/api/plugins")
	if err != nil {
		t.Fatalf("GET /api/plugins failed: %v", err)
	}
	defer resp.Body.Close()

	var response struct {
		Plugins []struct {
			Name           string `json:"name"`
			CapabilitiesV2 *struct {
				Entities []struct {
					Kind      string `json:"kind"`
					Endpoints struct {
						List string `json:"list"`
					} `json:"endpoints"`
				} `json:"entities"`
			} `json:"capabilitiesV2"`
		} `json:"plugins"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	tested := 0
	for _, p := range response.Plugins {
		if p.CapabilitiesV2 == nil {
			continue
		}
		for _, entity := range p.CapabilitiesV2.Entities {
			t.Run(p.Name+"/"+entity.Kind, func(t *testing.T) {
				url := serverURL + entity.Endpoints.List
				stats := runLoadTest(t, url, 10, 100)
				t.Logf("entity list %s/%s load: %s", p.Name, entity.Kind, stats.report())

				p95 := stats.percentile(0.95)
				if p95 > 300*time.Millisecond {
					t.Errorf("p95 latency %v exceeds 300ms SLO", p95)
				}
			})
			tested++
		}
		if tested >= 3 {
			break // Limit to 3 entity types.
		}
	}
}

// TestLoadEntityGet validates p95 latency SLO for entity get endpoints.
// SLO target: p95 <= 300ms.
func TestLoadEntityGet(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	// Discover entity list endpoints and get first entity.
	resp, err := http.Get(serverURL + "/api/plugins")
	if err != nil {
		t.Fatalf("GET /api/plugins failed: %v", err)
	}
	defer resp.Body.Close()

	var response struct {
		Plugins []struct {
			Name           string `json:"name"`
			CapabilitiesV2 *struct {
				Entities []struct {
					Kind      string `json:"kind"`
					Endpoints struct {
						List string `json:"list"`
						Get  string `json:"get"`
					} `json:"endpoints"`
				} `json:"entities"`
			} `json:"capabilitiesV2"`
		} `json:"plugins"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	for _, p := range response.Plugins {
		if p.CapabilitiesV2 == nil {
			continue
		}
		for _, entity := range p.CapabilitiesV2.Entities {
			if entity.Endpoints.Get == "" {
				continue
			}

			// Get the first entity to have a valid URL.
			listResp, err := http.Get(serverURL + entity.Endpoints.List)
			if err != nil {
				continue
			}
			var listData struct {
				Items []struct {
					Name string `json:"name"`
				} `json:"items"`
			}
			if err := json.NewDecoder(listResp.Body).Decode(&listData); err != nil {
				listResp.Body.Close()
				continue
			}
			listResp.Body.Close()

			if len(listData.Items) == 0 {
				continue
			}

			// Test GET for the first entity.
			t.Run(p.Name+"/"+entity.Kind, func(t *testing.T) {
				// The get endpoint template is like /api/{plugin}_catalog/v1alpha1/{kind}/{name}
				// Use the list URL but append the first item name.
				getURL := serverURL + entity.Endpoints.List + "/" + listData.Items[0].Name
				stats := runLoadTest(t, getURL, 10, 100)
				t.Logf("entity get %s/%s load: %s", p.Name, entity.Kind, stats.report())

				p95 := stats.percentile(0.95)
				if p95 > 300*time.Millisecond {
					t.Errorf("p95 latency %v exceeds 300ms SLO", p95)
				}
			})
			return // Test one entity type.
		}
	}
}

// TestLoadHealthEndpoints validates health endpoint latency under load.
// SLO target: p95 <= 100ms for health endpoints.
func TestLoadHealthEndpoints(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	for _, path := range []string{"/livez", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			stats := runLoadTest(t, serverURL+path, 10, 200)
			t.Logf("health %s load: %s", path, stats.report())

			p95 := stats.percentile(0.95)
			if p95 > 100*time.Millisecond {
				t.Errorf("p95 latency %v exceeds 100ms SLO", p95)
			}
		})
	}
}

// TestLoadConcurrentMixed validates that the server handles concurrent
// requests to different endpoints without degradation.
func TestLoadConcurrentMixed(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	endpoints := []string{
		"/api/plugins",
		"/livez",
		"/readyz",
	}

	// Discover a capability endpoint.
	resp, err := http.Get(serverURL + "/api/plugins")
	if err != nil {
		t.Fatalf("GET /api/plugins failed: %v", err)
	}
	defer resp.Body.Close()
	var response struct {
		Plugins []struct {
			Name string `json:"name"`
		} `json:"plugins"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err == nil && len(response.Plugins) > 0 {
		endpoints = append(endpoints, "/api/plugins/"+response.Plugins[0].Name+"/capabilities")
	}

	stats := &latencyStats{}
	const totalRequests = 400
	const concurrency = 20

	var wg sync.WaitGroup
	reqChan := make(chan int, totalRequests)
	for i := 0; i < totalRequests; i++ {
		reqChan <- i
	}
	close(reqChan)

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{Timeout: 10 * time.Second}
			for i := range reqChan {
				endpoint := endpoints[i%len(endpoints)]
				start := time.Now()
				resp, err := client.Get(serverURL + endpoint)
				elapsed := time.Since(start)
				if err != nil {
					stats.recordError()
					continue
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					stats.record(elapsed)
				} else {
					stats.recordError()
				}
			}
		}()
	}

	wg.Wait()
	t.Logf("mixed concurrent load: %s", stats.report())

	p95 := stats.percentile(0.95)
	if p95 > 300*time.Millisecond {
		t.Errorf("p95 latency %v exceeds 300ms SLO under concurrent load", p95)
	}
	errorRate := float64(stats.errorCount()) / float64(stats.count()+stats.errorCount())
	if errorRate > 0.01 {
		t.Errorf("error rate %.2f%% exceeds 1%% SLO", errorRate*100)
	}
}
