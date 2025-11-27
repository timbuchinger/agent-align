package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestLoadMalformedYAML(t *testing.T) {
	path := writeConfigFile(t, "mcpServers:\n  targets: [\n")

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
	if !strings.Contains(err.Error(), "failed to parse config") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestLoadNormalizesAndTrims(t *testing.T) {
	path := writeConfigFile(t, `mcpServers:
  targets:
    agents:
      -  CoPilot
      - GEMINI
      - copilot
      - ""
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(got.MCP.Targets.Agents) != 2 {
		t.Fatalf("unexpected agent count: %v", got.MCP.Targets.Agents)
	}
	if !reflect.DeepEqual(got.MCP.Targets.Agents, []AgentTarget{
		{Name: "copilot"},
		{Name: "gemini"},
	}) {
		t.Fatalf("unexpected agents: %#v", got.MCP.Targets.Agents)
	}
}
