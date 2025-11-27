package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	content := `mcpServers:
  configPath: ~/agent-align-mcp.yml
  targets:
    agents:
      - copilot
      - name: codex
        path: ~/custom.toml
    additionalTargets:
      json:
        - filePath: ~/extra.json
          jsonPath: .mcpServers
extraTargets:
  files:
    - source: ~/AGENTS.md
      destinations:
        - ~/dest/AGENTS.md
`

	path := writeConfigFile(t, content)
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got.MCP.ConfigPath != filepath.Join(dir, "agent-align-mcp.yml") {
		t.Fatalf("configPath not expanded, got %s", got.MCP.ConfigPath)
	}

	if len(got.MCP.Targets.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(got.MCP.Targets.Agents))
	}
	if got.MCP.Targets.Agents[0].Name != "copilot" {
		t.Fatalf("unexpected agent name: %s", got.MCP.Targets.Agents[0].Name)
	}
	if got.MCP.Targets.Agents[1].Path != filepath.Join(dir, "custom.toml") {
		t.Fatalf("agent override path not expanded, got %s", got.MCP.Targets.Agents[1].Path)
	}

	if got.ExtraTargets.IsZero() {
		t.Fatalf("expected extraTargets to be populated")
	}
	if got.MCP.Targets.Additional.IsZero() {
		t.Fatalf("expected additional targets to be populated")
	}
}

func TestLoadRejectsMissingTargets(t *testing.T) {
	path := writeConfigFile(t, `mcpServers:
  targets:
    agents: []
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing targets")
	}
	if !strings.Contains(err.Error(), "at least one target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsInvalidAdditionalTarget(t *testing.T) {
	path := writeConfigFile(t, `mcpServers:
  targets:
    agents: [copilot]
    additionalTargets:
      json:
        - filePath: ""
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing filePath")
	}
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	return path
}
