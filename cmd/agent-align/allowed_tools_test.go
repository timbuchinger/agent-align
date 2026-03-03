package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-align/internal/config"
)

func TestBuildWrapperScript(t *testing.T) {
	tools := []string{"shell(git fetch)", "shell(git pull)"}
	copilotPath := "/usr/bin/copilot"

	script := buildWrapperScript(copilotPath, tools)

	// Check that the script starts with shebang
	if !strings.HasPrefix(script, "#!/bin/bash\n") {
		t.Errorf("script should start with shebang, got: %q", script[:20])
	}

	// Check that it contains the copilot path
	if !strings.Contains(script, copilotPath) {
		t.Errorf("script should contain copilot path %q", copilotPath)
	}

	// Check that both tools are present
	if !strings.Contains(script, "shell(git fetch)") {
		t.Error("script should contain shell(git fetch)")
	}
	if !strings.Contains(script, "shell(git pull)") {
		t.Error("script should contain shell(git pull)")
	}

	// Check that it passes through arguments
	if !strings.Contains(script, `"$@"`) {
		t.Error("script should pass through arguments with $@")
	}

	// Check that it has proper flag format
	if !strings.Contains(script, "--allow-tool") {
		t.Error("script should contain --allow-tool flags")
	}
}

func TestBuildWrapperScriptEscapesQuotes(t *testing.T) {
	tools := []string{"shell(echo 'hello')"}
	copilotPath := "/usr/bin/copilot"

	script := buildWrapperScript(copilotPath, tools)

	// Single quotes should be escaped as '\''
	if !strings.Contains(script, `'\''`) {
		t.Error("script should escape single quotes in tool names")
	}
}

func TestGenerateCopilotWrapperWithNoTools(t *testing.T) {
	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			AlwaysAllowedTools: []string{},
		},
	}

	// Should return nil error when no tools are configured
	err := generateCopilotWrapper(cfg)
	if err != nil {
		t.Errorf("should not error when no tools are configured, got: %v", err)
	}
}

func TestGenerateCopilotWrapperCopilotNotFound(t *testing.T) {
	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			AlwaysAllowedTools: []string{"shell(git fetch)"},
		},
	}

	// When copilot is not found in PATH, should return nil silently
	// This is tested implicitly by the fact that exec.LookPath returns an error
	// and we handle it gracefully
	err := generateCopilotWrapper(cfg)
	if err != nil {
		// Should be nil since we skip silently when copilot is not found
		t.Errorf("should skip silently when copilot is not found, got: %v", err)
	}
}

func TestConvertToolToClaudePermission(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"shell(git fetch)", "Bash(git fetch:*)"},
		{"shell(git pull)", "Bash(git pull:*)"},
		{"shell(npm install)", "Bash(npm install:*)"},
		{"other(tool)", "other(tool)"},
		{"plaintext", "plaintext"},
	}

	for _, tt := range tests {
		got := convertToolToClaudePermission(tt.input)
		if got != tt.expected {
			t.Errorf("convertToolToClaudePermission(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestConvertToolToCodexRule(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"shell(git fetch)", `prefix_rule(pattern=["git", "fetch"], decision="allow")`},
		{"shell(git pull)", `prefix_rule(pattern=["git", "pull"], decision="allow")`},
		{"shell(npm install --save)", `prefix_rule(pattern=["npm", "install", "--save"], decision="allow")`},
		{"other(tool)", "other(tool)"},
		{"plaintext", "plaintext"},
	}

	for _, tt := range tests {
		got := convertToolToCodexRule(tt.input)
		if got != tt.expected {
			t.Errorf("convertToolToCodexRule(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestGenerateClaudePermissionsWithNoTools(t *testing.T) {
	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			AlwaysAllowedTools: []string{},
		},
	}

	err := generateClaudePermissions(cfg)
	if err != nil {
		t.Errorf("should not error when no tools are configured, got: %v", err)
	}
}

func TestGenerateClaudePermissionsWritesFile(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			AlwaysAllowedTools: []string{"shell(git fetch)", "shell(git pull)"},
			Targets: config.AllowedToolsTargets{
				Agents: []config.AllowedToolsAgent{
					{Name: "claude", Path: settingsPath},
				},
			},
		},
	}

	err := generateClaudePermissions(cfg)
	if err != nil {
		t.Fatalf("generateClaudePermissions returned error: %v", err)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings file: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse settings JSON: %v", err)
	}

	permissions, ok := result["permissions"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected permissions node in settings, got: %v", result)
	}

	allow, ok := permissions["allow"].([]interface{})
	if !ok {
		t.Fatalf("expected permissions.allow array, got: %v", permissions["allow"])
	}

	if len(allow) != 2 {
		t.Fatalf("expected 2 allow entries, got %d", len(allow))
	}
	if allow[0].(string) != "Bash(git fetch:*)" {
		t.Errorf("expected Bash(git fetch:*), got %s", allow[0])
	}
	if allow[1].(string) != "Bash(git pull:*)" {
		t.Errorf("expected Bash(git pull:*), got %s", allow[1])
	}
}

func TestGenerateClaudePermissionsMergesExistingConfig(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	// Write existing settings with other nodes
	existing := map[string]interface{}{
		"theme": "dark",
		"permissions": map[string]interface{}{
			"deny": []string{"Bash(rm -rf:*)"},
		},
	}
	existingData, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(settingsPath, existingData, 0o644); err != nil {
		t.Fatalf("failed to write existing settings: %v", err)
	}

	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			AlwaysAllowedTools: []string{"shell(git fetch)"},
			Targets: config.AllowedToolsTargets{
				Agents: []config.AllowedToolsAgent{
					{Name: "claude", Path: settingsPath},
				},
			},
		},
	}

	err := generateClaudePermissions(cfg)
	if err != nil {
		t.Fatalf("generateClaudePermissions returned error: %v", err)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings file: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse settings JSON: %v", err)
	}

	// Other nodes must be preserved
	if result["theme"] != "dark" {
		t.Errorf("expected theme to be preserved as dark, got: %v", result["theme"])
	}

	permissions, ok := result["permissions"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected permissions node, got: %v", result["permissions"])
	}

	// permissions.allow should be updated
	allow, ok := permissions["allow"].([]interface{})
	if !ok {
		t.Fatalf("expected permissions.allow array, got: %v", permissions["allow"])
	}
	if len(allow) != 1 || allow[0].(string) != "Bash(git fetch:*)" {
		t.Errorf("unexpected allow entries: %v", allow)
	}
}

func TestGenerateCodexRulesWithNoTools(t *testing.T) {
	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			AlwaysAllowedTools: []string{},
		},
	}

	err := generateCodexRules(cfg)
	if err != nil {
		t.Errorf("should not error when no tools are configured, got: %v", err)
	}
}

func TestGenerateCodexRulesWritesFile(t *testing.T) {
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, "agent-align.rules")

	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			AlwaysAllowedTools: []string{"shell(git fetch)", "shell(git pull)"},
			Targets: config.AllowedToolsTargets{
				Agents: []config.AllowedToolsAgent{
					{Name: "codex", Path: rulesPath},
				},
			},
		},
	}

	err := generateCodexRules(cfg)
	if err != nil {
		t.Fatalf("generateCodexRules returned error: %v", err)
	}

	data, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("failed to read rules file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `prefix_rule(pattern=["git", "fetch"], decision="allow")`) {
		t.Errorf("expected git fetch rule, got: %s", content)
	}
	if !strings.Contains(content, `prefix_rule(pattern=["git", "pull"], decision="allow")`) {
		t.Errorf("expected git pull rule, got: %s", content)
	}
}

func TestGenerateCodexRulesOverwritesFile(t *testing.T) {
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, "agent-align.rules")

	// Write existing content
	if err := os.WriteFile(rulesPath, []byte("old content\n"), 0o644); err != nil {
		t.Fatalf("failed to write existing rules: %v", err)
	}

	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			AlwaysAllowedTools: []string{"shell(git fetch)"},
			Targets: config.AllowedToolsTargets{
				Agents: []config.AllowedToolsAgent{
					{Name: "codex", Path: rulesPath},
				},
			},
		},
	}

	err := generateCodexRules(cfg)
	if err != nil {
		t.Fatalf("generateCodexRules returned error: %v", err)
	}

	data, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("failed to read rules file: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "old content") {
		t.Errorf("old content should have been overwritten, got: %s", content)
	}
	if !strings.Contains(content, `prefix_rule(pattern=["git", "fetch"], decision="allow")`) {
		t.Errorf("expected git fetch rule, got: %s", content)
	}
}

func TestGenerateCopilotWrapperSkipsNonCopilotAgents(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "settings.json")
	codexPath := filepath.Join(dir, "rules.txt")

	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			AlwaysAllowedTools: []string{"shell(git fetch)"},
			Targets: config.AllowedToolsTargets{
				Agents: []config.AllowedToolsAgent{
					{Name: "claude", Path: claudePath},
					{Name: "codex", Path: codexPath},
				},
			},
		},
	}

	// generateCopilotWrapper should skip claude and codex agents
	err := generateCopilotWrapper(cfg)
	if err != nil {
		t.Errorf("should not error for non-copilot agents, got: %v", err)
	}

	// No files should be created for claude/codex by the copilot wrapper
	if _, err := os.Stat(claudePath); err == nil {
		t.Error("generateCopilotWrapper should not create files for claude agents")
	}
	if _, err := os.Stat(codexPath); err == nil {
		t.Error("generateCopilotWrapper should not create files for codex agents")
	}
}
