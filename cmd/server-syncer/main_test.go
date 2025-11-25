package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"server-syncer/internal/config"
)

type promptStub struct {
	responses []bool
	idx       int
	prompts   []string
}

func (s *promptStub) answer(prompt string, defaultYes bool) bool {
	s.prompts = append(s.prompts, prompt)
	if s.idx >= len(s.responses) {
		return defaultYes
	}
	resp := s.responses[s.idx]
	s.idx++
	return resp
}

func TestEnsureConfigFileCreatesDefault(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "conf", "server-syncer.yml")

	stub := &promptStub{responses: []bool{true}}
	orig := promptUser
	promptUser = stub.answer
	t.Cleanup(func() { promptUser = orig })
	origCollect := collectConfig
	collectConfig = func() (config.Config, error) {
		return config.Config{Source: "codex", Targets: []string{"gemini", "copilot"}, Template: "template.json"}, nil
	}
	t.Cleanup(func() { collectConfig = origCollect })

	if err := ensureConfigFile(path); err != nil {
		t.Fatalf("ensureConfigFile failed: %v", err)
	}

	createdCfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("failed to load created config: %v", err)
	}
	expected := config.Config{Source: "codex", Targets: []string{"gemini", "copilot"}, Template: "template.json"}
	if !reflect.DeepEqual(createdCfg, expected) {
		t.Fatalf("config mismatch. got %#v", createdCfg)
	}

	if len(stub.prompts) != 1 {
		t.Fatalf("expected one prompt, got %d", len(stub.prompts))
	}
}

func TestEnsureConfigFileDeclined(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")

	stub := &promptStub{responses: []bool{false}}
	orig := promptUser
	promptUser = stub.answer
	t.Cleanup(func() { promptUser = orig })

	if err := ensureConfigFile(path); err == nil {
		t.Fatal("expected error when user declines to create config")
	}
}

func TestRunInitCommandOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "server-syncer.yml")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	stub := &promptStub{responses: []bool{true}}
	orig := promptUser
	promptUser = stub.answer
	t.Cleanup(func() { promptUser = orig })
	origCollect := collectConfig
	collectConfig = func() (config.Config, error) {
		return config.Config{Source: "gemini", Targets: []string{"codex"}, Template: "template.json"}, nil
	}
	t.Cleanup(func() { collectConfig = origCollect })

	if err := runInitCommand([]string{"-config", path}); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	createdCfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	expected := config.Config{Source: "gemini", Targets: []string{"codex"}, Template: "template.json"}
	if !reflect.DeepEqual(createdCfg, expected) {
		t.Fatalf("config not overwritten. got %#v", createdCfg)
	}
}

func TestRunInitCommandCancelOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "server-syncer.yml")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	stub := &promptStub{responses: []bool{false}}
	orig := promptUser
	promptUser = stub.answer
	t.Cleanup(func() { promptUser = orig })

	if err := runInitCommand([]string{"-config", path}); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if string(data) != "old" {
		t.Fatalf("config should not have changed, got %q", string(data))
	}
}

func TestRunInitCommandCreatesMissing(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "server-syncer.yml")

	stub := &promptStub{responses: []bool{true}}
	orig := promptUser
	promptUser = stub.answer
	t.Cleanup(func() { promptUser = orig })
	origCollect := collectConfig
	collectConfig = func() (config.Config, error) {
		return config.Config{Source: "copilot", Targets: []string{"claudecode"}, Template: "template.json"}, nil
	}
	t.Cleanup(func() { collectConfig = origCollect })

	if err := runInitCommand([]string{"-config", path}); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	createdCfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	expected := config.Config{Source: "copilot", Targets: []string{"claudecode"}, Template: "template.json"}
	if !reflect.DeepEqual(createdCfg, expected) {
		t.Fatalf("config mismatch. got %#v", createdCfg)
	}
}
