package syncer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncerSync(t *testing.T) {
	targets := []AgentTarget{
		{Name: "copilot"},
		{Name: "vscode"},
		{Name: "codex", PathOverride: "/custom/codex.toml"},
	}
	servers := map[string]interface{}{
		"command-server": map[string]interface{}{
			"command": "npx",
			"args":    []interface{}{"tool"},
		},
		"http-server": map[string]interface{}{
			"type": "streamable-http",
			"url":  "https://example.test",
		},
	}

	s := New(targets)
	result, err := s.Sync(servers)
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	if len(result.Agents) != len(targets) {
		t.Fatalf("expected %d agents, got %d", len(targets), len(result.Agents))
	}

	copilot := result.Agents["copilot"]
	var copilotData map[string]interface{}
	if err := json.Unmarshal([]byte(copilot.Content), &copilotData); err != nil {
		t.Fatalf("copilot output not valid JSON: %v", err)
	}
	mcpServers, ok := copilotData["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("copilot output missing mcpServers: %v", copilotData)
	}
	for name, srv := range mcpServers {
		server := srv.(map[string]interface{})
		if _, ok := server["tools"]; !ok {
			t.Fatalf("copilot server %s missing tools array", name)
		}
	}

	vscode := result.Agents["vscode"]
	var vscodeData map[string]interface{}
	if err := json.Unmarshal([]byte(vscode.Content), &vscodeData); err != nil {
		t.Fatalf("vscode output not valid JSON: %v", err)
	}
	if _, ok := vscodeData["servers"]; !ok {
		t.Fatalf("vscode output missing servers node: %v", vscodeData)
	}
	if server := vscodeData["servers"].(map[string]interface{})["command-server"].(map[string]interface{}); server["tools"] != nil {
		t.Fatalf("vscode server should not have tools added: %v", server)
	}

	codex := result.Agents["codex"]
	if codex.Config.FilePath != "/custom/codex.toml" {
		t.Fatalf("codex override not applied, got %s", codex.Config.FilePath)
	}
	if !strings.Contains(codex.Content, "[mcp_servers.command-server]") {
		t.Fatalf("codex output missing server block: %s", codex.Content)
	}
}

func TestSupportedAgents(t *testing.T) {
	agents := SupportedAgents()
	expected := []string{"copilot", "vscode", "codex", "claudecode", "gemini", "kilocode"}
	if len(agents) != len(expected) {
		t.Fatalf("expected %d agents, got %d", len(expected), len(agents))
	}
	for i, name := range expected {
		if agents[i] != name {
			t.Fatalf("agent[%d] = %s, want %s", i, agents[i], name)
		}
	}
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
	cfg := AgentConfig{Name: "codex", FilePath: path, Format: "toml"}
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

func TestParseServersFromJSON_WholePayload(t *testing.T) {
	payload := `{"server1": {"command": "npx"}}`
	got, err := parseServersFromJSON("", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 server, got %d", len(got))
	}
	if _, ok := got["server1"]; !ok {
		t.Fatalf("expected server1 key present")
	}
}

func TestParseServersFromJSON_InvalidJSON(t *testing.T) {
	payload := `not valid json`
	_, err := parseServersFromJSON("mcpServers", payload)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
