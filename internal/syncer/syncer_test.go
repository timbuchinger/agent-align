package syncer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncer_Sync(t *testing.T) {
	t.Run("success with JSON source", func(t *testing.T) {
		// Create a valid JSON payload for copilot format
		payload := `{
            "mcpServers": {
                "test-server": {
                    "command": "npx",
                    "args": ["test-mcp"]
                }
            }
        }`
		s := New("Copilot", []string{"Copilot", "Codex", "VSCode", "ClaudeCode", "Gemini"})
		template := Template{Name: "test-config", Payload: payload}

		result, err := s.Sync(template)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Agents) != 5 {
			t.Fatalf("expected 5 agents, got %d", len(result.Agents))
		}
		if _, ok := result.Servers["test-server"]; !ok {
			t.Fatalf("expected servers map to include test-server")
		}

		// Verify copilot output (JSON with mcpServers)
		var copilotData map[string]interface{}
		if err := json.Unmarshal([]byte(result.Agents["copilot"]), &copilotData); err != nil {
			t.Fatalf("copilot output is not valid JSON: %v", err)
		}
		if _, ok := copilotData["mcpServers"]; !ok {
			t.Error("copilot output should have mcpServers node")
		}

		// Verify vscode output (JSON with servers)
		var vscodeData map[string]interface{}
		if err := json.Unmarshal([]byte(result.Agents["vscode"]), &vscodeData); err != nil {
			t.Fatalf("vscode output is not valid JSON: %v", err)
		}
		if _, ok := vscodeData["servers"]; !ok {
			t.Error("vscode output should have servers node")
		}

		// Verify codex output (TOML format)
		if !strings.Contains(result.Agents["codex"], "[mcp_servers.test-server]") {
			t.Errorf("codex output should be in TOML format, got: %s", result.Agents["codex"])
		}

		// Verify claudecode output (JSON with mcpServers)
		var claudeData map[string]interface{}
		if err := json.Unmarshal([]byte(result.Agents["claudecode"]), &claudeData); err != nil {
			t.Fatalf("claudecode output is not valid JSON: %v", err)
		}
		if _, ok := claudeData["mcpServers"]; !ok {
			t.Error("claudecode output should have mcpServers node")
		}

		// Verify gemini output (JSON with mcpServers)
		var geminiData map[string]interface{}
		if err := json.Unmarshal([]byte(result.Agents["gemini"]), &geminiData); err != nil {
			t.Fatalf("gemini output is not valid JSON: %v", err)
		}
		if _, ok := geminiData["mcpServers"]; !ok {
			t.Error("gemini output should have mcpServers node")
		}
	})

	t.Run("success with TOML source", func(t *testing.T) {
		// Create a valid TOML payload for codex format
		payload := `[mcp_servers.test-server]
command = "npx"
args = ["test-mcp"]`
		s := New("Codex", []string{"Copilot", "Codex"})
		template := Template{Name: "test-config", Payload: payload}

		result, err := s.Sync(template)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Agents) != 2 {
			t.Fatalf("expected 2 agents, got %d", len(result.Agents))
		}

		// Verify copilot output (JSON with mcpServers)
		var copilotData map[string]interface{}
		if err := json.Unmarshal([]byte(result.Agents["copilot"]), &copilotData); err != nil {
			t.Fatalf("copilot output is not valid JSON: %v", err)
		}
		if _, ok := copilotData["mcpServers"]; !ok {
			t.Error("copilot output should have mcpServers node")
		}

		// Verify codex output (TOML format)
		if !strings.Contains(result.Agents["codex"], "[mcp_servers.test-server]") {
			t.Errorf("codex output should be in TOML format, got: %s", result.Agents["codex"])
		}
	})

	t.Run("missing template payload", func(t *testing.T) {
		s := New("copilot", []string{"copilot"})
		_, err := s.Sync(Template{Name: "empty", Payload: ""})
		if err == nil {
			t.Fatal("expected error for empty payload")
		}
	})

	t.Run("source agent not registered", func(t *testing.T) {
		s := New("unknown", []string{"copilot"})
		_, err := s.Sync(Template{Name: "payload", Payload: `{"mcpServers": {}}`})
		if err == nil || !strings.Contains(err.Error(), "source agent") {
			t.Fatalf("unexpected error state: %v", err)
		}
	})
}

func TestLoadTemplateFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "template.txt")
	if err := os.WriteFile(path, []byte("  contents with whitespace\n"), 0o644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	tpl, err := LoadTemplateFromFile(path)
	if err != nil {
		t.Fatalf("failed to load template: %v", err)
	}

	if tpl.Name != "template.txt" {
		t.Fatalf("unexpected template name %q", tpl.Name)
	}
	if tpl.Payload != "contents with whitespace" {
		t.Fatalf("unexpected payload %q", tpl.Payload)
	}
}

func TestGetAgentConfig(t *testing.T) {
	tests := []struct {
		agent    string
		nodeName string
		format   string
	}{
		{"copilot", "mcpServers", "json"},
		{"vscode", "servers", "json"},
		{"codex", "", "toml"},
		{"claudecode", "mcpServers", "json"},
		{"gemini", "mcpServers", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			config, err := GetAgentConfig(tt.agent)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if config.NodeName != tt.nodeName {
				t.Errorf("expected node name %q, got %q", tt.nodeName, config.NodeName)
			}
			if config.Format != tt.format {
				t.Errorf("expected format %q, got %q", tt.format, config.Format)
			}
			if config.FilePath == "" {
				t.Error("expected non-empty file path")
			}
		})
	}

	t.Run("unsupported agent", func(t *testing.T) {
		_, err := GetAgentConfig("unknown")
		if err == nil {
			t.Error("expected error for unsupported agent")
		}
	})
}

func TestSupportedAgents(t *testing.T) {
	agents := SupportedAgents()
	expected := []string{"copilot", "vscode", "codex", "claudecode", "gemini"}
	if len(agents) != len(expected) {
		t.Fatalf("expected %d agents, got %d", len(expected), len(agents))
	}
	for i, agent := range expected {
		if agents[i] != agent {
			t.Errorf("expected agent %q at index %d, got %q", agent, i, agents[i])
		}
	}
}

func TestFormatConversion(t *testing.T) {
	t.Run("JSON to TOML conversion", func(t *testing.T) {
		payload := `{
            "mcpServers": {
                "server1": {
                    "command": "npx",
                    "args": ["arg1", "arg2"]
                }
            }
        }`
		servers, err := parseServersFromJSON("mcpServers", payload)
		if err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		toml := formatToTOML(servers)
		if !strings.Contains(toml, "[mcp_servers.server1]") {
			t.Errorf("TOML should contain server section, got: %s", toml)
		}
		if !strings.Contains(toml, "command = \"npx\"") {
			t.Errorf("TOML should contain command, got: %s", toml)
		}
	})

	t.Run("TOML to JSON conversion", func(t *testing.T) {
		payload := `[mcp_servers.server1]
command = "npx"
args = ["arg1", "arg2"]`
		servers, err := parseServersFromTOML(payload)
		if err != nil {
			t.Fatalf("failed to parse TOML: %v", err)
		}

		jsonOutput := formatToJSON("mcpServers", servers)
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(jsonOutput), &data); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}
		if _, ok := data["mcpServers"]; !ok {
			t.Error("JSON should have mcpServers node")
		}
	})
}

func TestFormatCodexConfigPreservesExistingSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	existing := `# Codex configuration
[general]
theme = "dark"

[mcp_servers.old]
command = "node"
args = ["--flag"]

[editor]
font_size = 12
`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("failed to write existing config: %v", err)
	}

	servers := map[string]interface{}{
		"new": map[string]interface{}{
			"command": "npx",
			"args":    []interface{}{"tool"},
		},
	}
	cfg := AgentConfig{FilePath: path, Format: "toml"}
	result := formatCodexConfig(cfg, servers)

	if !strings.Contains(result, "[general]") {
		t.Fatal("general section should remain in output")
	}
	if !strings.Contains(result, "[editor]") {
		t.Fatal("editor section should remain in output")
	}
	if strings.Contains(result, "[mcp_servers.old]") {
		t.Fatal("old MCP blocks should be removed")
	}
	if !strings.Contains(result, "[mcp_servers.new]") {
		t.Fatal("new MCP block should be present")
	}
}

func TestFormatCodexConfigCreatesFileWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	servers := map[string]interface{}{
		"server": map[string]interface{}{
			"command": "node",
		},
	}
	cfg := AgentConfig{FilePath: path, Format: "toml"}
	result := formatCodexConfig(cfg, servers)

	if result == "" {
		t.Fatal("expected MCP servers to be rendered even without existing file")
	}
	if !strings.Contains(result, "[mcp_servers.server]") {
		t.Fatal("MCP section should be rendered for missing file")
	}
}

func TestParseTOMLArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"simple array", `["a", "b", "c"]`, []string{"a", "b", "c"}},
		{"array with spaces", `[ "a" , "b" , "c" ]`, []string{"a", "b", "c"}},
		{"array with comma in value", `["a,b", "c"]`, []string{"a,b", "c"}},
		{"empty array", `[]`, nil},
		{"single element", `["only"]`, []string{"only"}},
		{"path arguments", `["-y", "@modelcontextprotocol/server-github"]`, []string{"-y", "@modelcontextprotocol/server-github"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTOMLArray(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d elements, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("element[%d] = %q, want %q", i, result[i], expected)
				}
			}
		})
	}
}

func TestSyncer_CopilotTransformAddToolsArray(t *testing.T) {
	// Test that copilot output includes tools array for command servers
	payload := `{
		"mcpServers": {
			"test-server": {
				"command": "npx",
				"args": ["test-mcp"]
			}
		}
	}`
	s := New("copilot", []string{"copilot", "vscode"})
	template := Template{Name: "test-config", Payload: payload}

	result, err := s.Sync(template)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse copilot output and verify tools array was added
	var copilotData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Agents["copilot"]), &copilotData); err != nil {
		t.Fatalf("copilot output is not valid JSON: %v", err)
	}

	mcpServers, ok := copilotData["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("copilot output missing mcpServers")
	}

	server, ok := mcpServers["test-server"].(map[string]interface{})
	if !ok {
		t.Fatal("copilot output missing test-server")
	}

	tools, hasTools := server["tools"]
	if !hasTools {
		t.Fatal("copilot server should have tools array added")
	}

	toolsArr, ok := tools.([]interface{})
	if !ok {
		t.Fatalf("tools should be an array, got %T", tools)
	}
	if len(toolsArr) != 0 {
		t.Errorf("tools array should be empty, got %v", toolsArr)
	}

	// Verify vscode output does NOT have tools array added (no transformation)
	var vscodeData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Agents["vscode"]), &vscodeData); err != nil {
		t.Fatalf("vscode output is not valid JSON: %v", err)
	}

	vscodeServers, ok := vscodeData["servers"].(map[string]interface{})
	if !ok {
		t.Fatal("vscode output missing servers")
	}

	vscodeServer, ok := vscodeServers["test-server"].(map[string]interface{})
	if !ok {
		t.Fatal("vscode output missing test-server")
	}

	if _, hasTools := vscodeServer["tools"]; hasTools {
		t.Error("vscode server should NOT have tools array (no transformation)")
	}
}

func TestSyncer_CopilotTransformPreservesExistingTools(t *testing.T) {
	// Test that copilot output preserves existing tools array
	payload := `{
		"mcpServers": {
			"test-server": {
				"command": "npx",
				"args": ["test-mcp"],
				"tools": ["existing-tool"]
			}
		}
	}`
	s := New("copilot", []string{"copilot"})
	template := Template{Name: "test-config", Payload: payload}

	result, err := s.Sync(template)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var copilotData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Agents["copilot"]), &copilotData); err != nil {
		t.Fatalf("copilot output is not valid JSON: %v", err)
	}

	mcpServers, ok := copilotData["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("copilot output missing mcpServers")
	}

	server, ok := mcpServers["test-server"].(map[string]interface{})
	if !ok {
		t.Fatal("copilot output missing test-server")
	}

	tools, hasTools := server["tools"]
	if !hasTools {
		t.Fatal("copilot server should have tools array")
	}

	toolsArr, ok := tools.([]interface{})
	if !ok {
		t.Fatalf("tools should be an array, got %T", tools)
	}
	if len(toolsArr) != 1 {
		t.Errorf("tools array should have 1 element, got %d", len(toolsArr))
	}
}

func TestSyncer_CopilotNetworkServerValidation(t *testing.T) {
	// Test that copilot validation fails for network servers missing required fields
	testCases := []struct {
		name        string
		payload     string
		shouldError bool
		errContains string
	}{
		{
			name: "valid network server",
			payload: `{
				"mcpServers": {
					"network-server": {
						"type": "sse",
						"url": "https://example.com/sse"
					}
				}
			}`,
			shouldError: false,
		},
		{
			name: "network server missing url",
			payload: `{
				"mcpServers": {
					"network-server": {
						"type": "http"
					}
				}
			}`,
			shouldError: true,
			errContains: "url",
		},
		{
			name: "network server missing type",
			payload: `{
				"mcpServers": {
					"network-server": {
						"url": "https://example.com/api"
					}
				}
			}`,
			shouldError: true,
			errContains: "type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := New("copilot", []string{"copilot"})
			template := Template{Name: "test-config", Payload: tc.payload}

			_, err := s.Sync(template)
			if tc.shouldError {
				if err == nil {
					t.Fatal("expected error for invalid network server")
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.errContains)) {
					t.Errorf("error should mention %q, got: %v", tc.errContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSyncer_CopilotTransformDoesNotAffectOtherAgents(t *testing.T) {
	// Test that copilot transformation does not affect other agents
	payload := `{
		"mcpServers": {
			"cmd-server": {
				"command": "npx",
				"args": ["test"]
			}
		}
	}`
	s := New("copilot", []string{"copilot", "vscode", "claudecode"})
	template := Template{Name: "test-config", Payload: payload}

	result, err := s.Sync(template)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Copilot should have tools added
	var copilotData map[string]interface{}
	json.Unmarshal([]byte(result.Agents["copilot"]), &copilotData)
	copilotServers := copilotData["mcpServers"].(map[string]interface{})
	copilotServer := copilotServers["cmd-server"].(map[string]interface{})
	if _, hasTools := copilotServer["tools"]; !hasTools {
		t.Error("copilot should have tools array")
	}

	// VSCode should NOT have tools added
	var vscodeData map[string]interface{}
	json.Unmarshal([]byte(result.Agents["vscode"]), &vscodeData)
	vscodeServers := vscodeData["servers"].(map[string]interface{})
	vscodeServer := vscodeServers["cmd-server"].(map[string]interface{})
	if _, hasTools := vscodeServer["tools"]; hasTools {
		t.Error("vscode should NOT have tools array")
	}

	// ClaudeCode should NOT have tools added
	var claudeData map[string]interface{}
	json.Unmarshal([]byte(result.Agents["claudecode"]), &claudeData)
	claudeServers := claudeData["mcpServers"].(map[string]interface{})
	claudeServer := claudeServers["cmd-server"].(map[string]interface{})
	if _, hasTools := claudeServer["tools"]; hasTools {
		t.Error("claudecode should NOT have tools array")
	}
}
