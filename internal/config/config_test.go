package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	path := writeConfigFile(t, `sourceAgent: codex
targets:
  agents:
    - gemini
    - copilot
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	want := Config{
		SourceAgent: "codex",
		Targets: TargetsConfig{
			Agents: []string{"gemini", "copilot"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected config: %#v", got)
	}
}

func TestLoadRejectsMissingTargets(t *testing.T) {
	path := writeConfigFile(t, `sourceAgent: codex
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

func TestLoadRejectsSourceAsTarget(t *testing.T) {
	path := writeConfigFile(t, `sourceAgent: copilot
targets:
  agents:
    - copilot
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when target matches source")
	}
	if !strings.Contains(err.Error(), "both source and target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}
