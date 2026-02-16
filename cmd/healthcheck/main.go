// Package main provides a minimal HTTP healthcheck binary.
// It performs a GET request to a specified URL and exits with
// code 0 on success (2xx) or code 1 on failure.
// Usage: healthcheck http://localhost:8080/readyz
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: healthcheck <url>\n")
		os.Exit(1)
	}

	url := os.Args[1]
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "healthcheck failed: status %d\n", resp.StatusCode)
	os.Exit(1)
}
