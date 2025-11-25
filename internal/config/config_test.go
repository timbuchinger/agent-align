package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	path := writeConfigFile(t, `source: codex
targets:
  - gemini
  - copilot
template: ./template.json
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	want := Config{Source: "codex", Targets: []string{"gemini", "copilot"}, Template: "./template.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected config: %#v", got)
	}
}

func TestLoadRejectsMissingTargets(t *testing.T) {
	path := writeConfigFile(t, `source: codex
targets: []
template: ./tmpl.yml
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
	path := writeConfigFile(t, `source: copilot
targets:
  - copilot
template: ./tmpl.yml
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when target matches source")
	}
	if !strings.Contains(err.Error(), "both source and target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRequiresTemplate(t *testing.T) {
	path := writeConfigFile(t, `source: codex
targets:
  - gemini
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when template missing")
	}
	if !strings.Contains(err.Error(), "template") {
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
