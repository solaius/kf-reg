package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeflow/model-registry/pkg/catalog/plugin"
)

func TestValidateMcpContent(t *testing.T) {
	tests := []struct {
		name     string
		input    plugin.SourceConfigInput
		wantErrs int
		wantMsg  string
	}{
		{
			name: "valid content with all known fields",
			input: plugin.SourceConfigInput{
				Properties: map[string]any{
					"content": `mcpservers:
- name: test-server
  externalId: ext-1
  description: A test MCP server
  serverUrl: http://example.com
  transportType: stdio
  toolCount: 5
  resourceCount: 3
  promptCount: 2
  deploymentMode: local
  image: myimage:latest
  endpoint: http://endpoint.example.com
  supportedTransports: stdio,sse
  license: Apache-2.0
  verified: true
  certified: false
  provider: Acme Corp
  logo: https://example.com/logo.png
  category: tools
  customProperties:
    key1: value1
`,
				},
			},
			wantErrs: 0,
		},
		{
			name: "valid content with minimal fields",
			input: plugin.SourceConfigInput{
				Properties: map[string]any{
					"content": `mcpservers:
- name: minimal-server
  serverUrl: http://example.com
`,
				},
			},
			wantErrs: 0,
		},
		{
			name: "content with unknown field",
			input: plugin.SourceConfigInput{
				Properties: map[string]any{
					"content": `mcpservers:
- name: test-server
  serverUrl: http://example.com
  unknownField: true
`,
				},
			},
			wantErrs: 1,
			wantMsg:  "unknown or invalid fields in content",
		},
		{
			name: "content with unknown top-level field",
			input: plugin.SourceConfigInput{
				Properties: map[string]any{
					"content": `mcpservers:
- name: test-server
  serverUrl: http://example.com
bogusTopLevel: oops
`,
				},
			},
			wantErrs: 1,
			wantMsg:  "unknown or invalid fields in content",
		},
		{
			name: "empty content passes",
			input: plugin.SourceConfigInput{
				Properties: map[string]any{
					"content": "",
				},
			},
			wantErrs: 0,
		},
		{
			name: "no content property passes",
			input: plugin.SourceConfigInput{
				Properties: map[string]any{
					"other": "value",
				},
			},
			wantErrs: 0,
		},
		{
			name: "nil properties passes",
			input: plugin.SourceConfigInput{
				Properties: nil,
			},
			wantErrs: 0,
		},
		{
			name: "content is not a string passes",
			input: plugin.SourceConfigInput{
				Properties: map[string]any{
					"content": 42,
				},
			},
			wantErrs: 0,
		},
		{
			name: "invalid YAML returns error",
			input: plugin.SourceConfigInput{
				Properties: map[string]any{
					"content": "not: [valid: yaml: {{",
				},
			},
			wantErrs: 1,
			wantMsg:  "unknown or invalid fields in content",
		},
		{
			name: "empty mcpservers list passes",
			input: plugin.SourceConfigInput{
				Properties: map[string]any{
					"content": "mcpservers: []\n",
				},
			},
			wantErrs: 0,
		},
		{
			name: "multiple entries with one unknown field fails",
			input: plugin.SourceConfigInput{
				Properties: map[string]any{
					"content": `mcpservers:
- name: server1
  serverUrl: http://example.com
- name: server2
  serverUrl: http://example2.com
  badField: oops
`,
				},
			},
			wantErrs: 1,
			wantMsg:  "unknown or invalid fields in content",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := validateMcpContent(tc.input)
			require.Len(t, errs, tc.wantErrs)
			if tc.wantErrs > 0 {
				assert.Equal(t, "properties.content", errs[0].Field)
				assert.Contains(t, errs[0].Message, tc.wantMsg)
			}
		})
	}
}
