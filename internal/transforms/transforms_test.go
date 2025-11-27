package transforms

import (
	"strings"
	"testing"
)

func TestGetTransformer(t *testing.T) {
	tests := []struct {
		name        string
		agent       string
		wantCopilot bool
	}{
		{"copilot lowercase", "copilot", true},
		{"copilot uppercase", "COPILOT", true},
		{"copilot mixed case", "Copilot", true},
		{"copilot with spaces", " copilot ", true},
		{"vscode", "vscode", false},
		{"codex", "codex", false},
		{"claudecode", "claudecode", false},
		{"gemini", "gemini", false},
		{"unknown", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTransformer(tt.agent)
			_, isCopilot := got.(*CopilotTransformer)
			if isCopilot != tt.wantCopilot {
				t.Errorf("GetTransformer(%q) isCopilot = %v, want %v", tt.agent, isCopilot, tt.wantCopilot)
			}
		})
	}
}

func TestNoOpTransformer_Transform(t *testing.T) {
	transformer := &NoOpTransformer{}
	servers := map[string]interface{}{
		"test": map[string]interface{}{
			"command": "npx",
		},
	}

	err := transformer.Transform(servers)
	if err != nil {
		t.Errorf("NoOpTransformer.Transform() returned error: %v", err)
	}

	// Verify servers were not modified
	server := servers["test"].(map[string]interface{})
	if _, hasTools := server["tools"]; hasTools {
		t.Error("NoOpTransformer should not modify servers")
	}
}

func TestCopilotTransformer_AddToolsForCommandServer(t *testing.T) {
	transformer := &CopilotTransformer{}
	tests := []struct {
		name           string
		servers        map[string]interface{}
		wantToolsAdded bool
	}{
		{
			name: "command server without tools",
			servers: map[string]interface{}{
				"test-server": map[string]interface{}{
					"command": "npx",
					"args":    []interface{}{"-y", "test-mcp"},
				},
			},
			wantToolsAdded: true,
		},
		{
			name: "command server with existing tools",
			servers: map[string]interface{}{
				"test-server": map[string]interface{}{
					"command": "npx",
					"tools":   []interface{}{"existing-tool"},
				},
			},
			wantToolsAdded: false, // tools already exists, should not be overwritten
		},
		{
			name: "command server with empty tools array",
			servers: map[string]interface{}{
				"test-server": map[string]interface{}{
					"command": "npx",
					"tools":   []interface{}{},
				},
			},
			wantToolsAdded: false, // tools already exists, should not be overwritten
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of the original tools length
			server := tt.servers["test-server"].(map[string]interface{})
			var originalToolsLen int
			var hadTools bool
			if orig, ok := server["tools"]; ok {
				hadTools = true
				if origArr, ok := orig.([]interface{}); ok {
					originalToolsLen = len(origArr)
				}
			}

			err := transformer.Transform(tt.servers)
			if err != nil {
				t.Errorf("Transform() returned unexpected error: %v", err)
			}

			tools, hasTools := server["tools"]
			if !hasTools {
				t.Error("Transform() should ensure tools field exists")
				return
			}

			if tt.wantToolsAdded {
				// Should have added empty tools array
				toolsArr, ok := tools.([]interface{})
				if !ok {
					t.Errorf("tools should be []interface{}, got %T", tools)
					return
				}
				if len(toolsArr) != 0 {
					t.Errorf("added tools array should be empty, got %v", toolsArr)
				}
			} else {
				// Should have preserved existing tools (check length matches)
				if hadTools {
					toolsArr, ok := tools.([]interface{})
					if !ok {
						t.Errorf("tools should be []interface{}, got %T", tools)
						return
					}
					if len(toolsArr) != originalToolsLen {
						t.Error("Transform() should not overwrite existing tools field")
					}
				}
			}
		})
	}
}

func TestCopilotTransformer_ValidateNetworkServer(t *testing.T) {
	transformer := &CopilotTransformer{}
	tests := []struct {
		name        string
		servers     map[string]interface{}
		wantErr     bool
		errContains string
	}{
		{
			name: "valid network server with type and url",
			servers: map[string]interface{}{
				"network-server": map[string]interface{}{
					"type": "sse",
					"url":  "https://example.com/sse",
				},
			},
			wantErr: false,
		},
		{
			name: "network server missing url",
			servers: map[string]interface{}{
				"network-server": map[string]interface{}{
					"type": "http",
				},
			},
			wantErr:     true,
			errContains: "url",
		},
		{
			name: "network server missing type",
			servers: map[string]interface{}{
				"network-server": map[string]interface{}{
					"url": "https://example.com/api",
				},
			},
			wantErr:     true,
			errContains: "type",
		},
		{
			name: "command server is not validated as network",
			servers: map[string]interface{}{
				"command-server": map[string]interface{}{
					"command": "npx",
					"args":    []interface{}{"test"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transformer.Transform(tt.servers)
			if tt.wantErr {
				if err == nil {
					t.Error("Transform() should return error for invalid network server")
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("error should mention %q, got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Transform() returned unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCopilotTransformer_MixedServers(t *testing.T) {
	transformer := &CopilotTransformer{}
	servers := map[string]interface{}{
		"command-server": map[string]interface{}{
			"command": "npx",
			"args":    []interface{}{"-y", "test-mcp"},
		},
		"network-server": map[string]interface{}{
			"type": "sse",
			"url":  "https://example.com/sse",
		},
	}

	err := transformer.Transform(servers)
	if err != nil {
		t.Errorf("Transform() returned unexpected error: %v", err)
	}

	// Verify command server has tools added
	cmdServer := servers["command-server"].(map[string]interface{})
	if _, hasTools := cmdServer["tools"]; !hasTools {
		t.Error("command server should have tools array added")
	}

	// Verify network server was validated (no error means it passed)
	// The network server should not have tools added since it's not a command server
	netServer := servers["network-server"].(map[string]interface{})
	if _, hasTools := netServer["tools"]; hasTools {
		t.Error("network server should not have tools array added")
	}
}

func TestCopilotTransformer_ErrorMessageFormat(t *testing.T) {
	transformer := &CopilotTransformer{}
	servers := map[string]interface{}{
		"my-server": map[string]interface{}{
			"type": "http",
			// missing url
		},
	}

	err := transformer.Transform(servers)
	if err == nil {
		t.Fatal("expected error for invalid network server")
	}

	// Check error message contains useful information
	errStr := err.Error()
	if !strings.Contains(errStr, "my-server") {
		t.Error("error message should contain server name")
	}
	if !strings.Contains(errStr, "url") {
		t.Error("error message should mention missing field")
	}
	if !strings.Contains(errStr, "copilot") {
		t.Error("error message should mention copilot")
	}
}

func TestIsCommandServer(t *testing.T) {
	tests := []struct {
		name   string
		server map[string]interface{}
		want   bool
	}{
		{
			name:   "has command field",
			server: map[string]interface{}{"command": "npx"},
			want:   true,
		},
		{
			name:   "no command field",
			server: map[string]interface{}{"type": "http"},
			want:   false,
		},
		{
			name:   "empty server",
			server: map[string]interface{}{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCommandServer(tt.server); got != tt.want {
				t.Errorf("isCommandServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNetworkServer(t *testing.T) {
	tests := []struct {
		name   string
		server map[string]interface{}
		want   bool
	}{
		{
			name:   "has type field",
			server: map[string]interface{}{"type": "http"},
			want:   true,
		},
		{
			name:   "has url field",
			server: map[string]interface{}{"url": "https://example.com"},
			want:   true,
		},
		{
			name:   "has both type and url",
			server: map[string]interface{}{"type": "sse", "url": "https://example.com"},
			want:   true,
		},
		{
			name:   "command server only",
			server: map[string]interface{}{"command": "npx"},
			want:   false,
		},
		{
			name:   "empty server",
			server: map[string]interface{}{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNetworkServer(tt.server); got != tt.want {
				t.Errorf("isNetworkServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopilotTransformer_NonMapServer(t *testing.T) {
	transformer := &CopilotTransformer{}
	servers := map[string]interface{}{
		"invalid-server": "not a map",
		"valid-server": map[string]interface{}{
			"command": "npx",
		},
	}

	// Should not panic on non-map server entries
	err := transformer.Transform(servers)
	if err != nil {
		t.Errorf("Transform() should handle non-map entries gracefully, got: %v", err)
	}

	// Valid server should still be transformed
	validServer := servers["valid-server"].(map[string]interface{})
	if _, hasTools := validServer["tools"]; !hasTools {
		t.Error("valid command server should have tools array added")
	}
}
