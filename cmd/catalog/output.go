// Package main provides the catalog CLI binary for managing the catalog server.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// outputFormat specifies how to render CLI output.
type outputFormat string

const (
	outputTable outputFormat = "table"
	outputJSON  outputFormat = "json"
	outputYAML  outputFormat = "yaml"
)

// parseOutputFormat parses and validates the output format flag.
func parseOutputFormat(s string) (outputFormat, error) {
	switch strings.ToLower(s) {
	case "table", "":
		return outputTable, nil
	case "json":
		return outputJSON, nil
	case "yaml":
		return outputYAML, nil
	default:
		return "", fmt.Errorf("unsupported output format %q (supported: table, json, yaml)", s)
	}
}

// printOutput renders data in the requested format.
// For table output, headers and rows must be provided.
// For json/yaml output, data is serialized directly.
func printOutput(w io.Writer, format outputFormat, data any, headers []string, rows [][]string) error {
	switch format {
	case outputJSON:
		return printJSON(w, data)
	case outputYAML:
		return printYAML(w, data)
	default:
		return printTable(w, headers, rows)
	}
}

// printJSON writes pretty-printed JSON to the writer.
func printJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// printYAML writes YAML to the writer.
func printYAML(w io.Writer, data any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer enc.Close()
	return enc.Encode(data)
}

// printTable writes aligned columnar output to the writer.
func printTable(w io.Writer, headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	// Print header
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(tw, "\t")
		}
		fmt.Fprint(tw, strings.ToUpper(h))
	}
	fmt.Fprintln(tw)

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			fmt.Fprint(tw, cell)
		}
		fmt.Fprintln(tw)
	}

	return tw.Flush()
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
