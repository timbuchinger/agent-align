package syncer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSyncer_EndToEnd performs an integration-style test by overriding HOME
// to a temporary directory, creating a source Copilot config file, running
// Syncer.Sync and validating converted outputs for multiple agents.
func TestSyncer_EndToEnd(t *testing.T) {
    dir := t.TempDir()
    origHome := os.Getenv("HOME")
    defer os.Setenv("HOME", origHome)
    if err := os.Setenv("HOME", dir); err != nil {
        t.Fatalf("failed to set HOME: %v", err)
    }

    // Prepare a Copilot source file in the temporary HOME
    copilotCfg, err := GetAgentConfig("copilot")
    if err != nil {
        t.Fatalf("GetAgentConfig failed: %v", err)
    }
    if err := os.MkdirAll(filepath.Dir(copilotCfg.FilePath), 0o755); err != nil {
        t.Fatalf("failed to create directories: %v", err)
    }
    payload := `{"mcpServers": {"int-server": {"command": "npx", "args": ["a"]}}}`
    if err := os.WriteFile(copilotCfg.FilePath, []byte(payload), 0o644); err != nil {
        t.Fatalf("failed to write copilot source file: %v", err)
    }

    s := New("copilot", SupportedAgents())

    tpl, err := LoadTemplateFromFile(copilotCfg.FilePath)
    if err != nil {
        t.Fatalf("LoadTemplateFromFile failed: %v", err)
    }

    got, err := s.Sync(tpl)
    if err != nil {
        t.Fatalf("Sync returned error: %v", err)
    }

    // Expect outputs for all supported agents
    agents := SupportedAgents()
    if len(got) != len(agents) {
        t.Fatalf("expected %d agents, got %d", len(agents), len(got))
    }

    // Check codex output is TOML-like
    codexOut, ok := got["codex"]
    if !ok {
        t.Fatalf("missing codex output")
    }
    if !strings.Contains(codexOut, "[mcp_servers.int-server]") {
        t.Fatalf("codex output does not contain expected TOML server block: %s", codexOut)
    }

    // Check copilot output is valid JSON with mcpServers
    copilotOut, ok := got["copilot"]
    if !ok {
        t.Fatalf("missing copilot output")
    }
    var parsed map[string]interface{}
    if err := json.Unmarshal([]byte(copilotOut), &parsed); err != nil {
        t.Fatalf("copilot output is not valid JSON: %v", err)
    }
    if _, ok := parsed["mcpServers"]; !ok {
        t.Fatalf("copilot JSON missing mcpServers node")
    }
}
