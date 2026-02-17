package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

func printOutput(v any) error {
	switch outputFmt {
	case "json":
		return printJSON(v)
	case "yaml":
		return printYAML(v)
	default:
		return fmt.Errorf("unsupported output format for structured data: %s (use json or yaml)", outputFmt)
	}
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printYAML(v any) error {
	// Convert through JSON to get consistent keys (json tags).
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var m any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	return enc.Encode(m)
}

func printTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print headers in uppercase.
	upperHeaders := make([]string, len(headers))
	for i, h := range headers {
		upperHeaders[i] = strings.ToUpper(h)
	}
	fmt.Fprintln(w, strings.Join(upperHeaders, "\t"))

	// Print rows.
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	w.Flush()
}

// truncate shortens a string to max length, appending "..." if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
