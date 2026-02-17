package main

import (
	"fmt"
	"os"
)

func main() {
	// Try to discover plugins (non-fatal if server is unreachable).
	// Dynamic commands are only available when the server is reachable.
	if err := discoverPlugins(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not discover plugins: %v\n", err)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
