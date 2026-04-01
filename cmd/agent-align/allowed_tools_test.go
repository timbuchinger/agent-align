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

func TestConvertClaudePermissionToTool(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Bash(git fetch:*)", "shell(git fetch)"},
		{"Bash(git pull:*)", "shell(git pull)"},
		{"Bash(npm install:*)", "shell(npm install)"},
		{"Bash(npm install --save:*)", "shell(npm install --save)"},
		// Non-matching strings should be returned unchanged.
		{"other(tool)", "other(tool)"},
		{"plaintext", "plaintext"},
		// Bash without the trailing :*) should not be converted.
		{"Bash(git fetch)", "Bash(git fetch)"},
	}

	for _, tt := range tests {
		got := convertClaudePermissionToTool(tt.input)
		if got != tt.expected {
			t.Errorf("convertClaudePermissionToTool(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestConvertCodexRuleToTool(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`prefix_rule(pattern=["git", "fetch"], decision="allow")`, "shell(git fetch)"},
		{`prefix_rule(pattern=["git", "pull"], decision="allow")`, "shell(git pull)"},
		{`prefix_rule(pattern=["npm", "install", "--save"], decision="allow")`, "shell(npm install --save)"},
		// Single-word command.
		{`prefix_rule(pattern=["make"], decision="allow")`, "shell(make)"},
		// Non-matching strings should be returned unchanged.
		{"other(tool)", "other(tool)"},
		{"plaintext", "plaintext"},
		// Malformed (no closing bracket) should be returned unchanged.
		{`prefix_rule(pattern=["git", "fetch"`, `prefix_rule(pattern=["git", "fetch"`},
		// Whitespace variants (spaces around '=') should be handled correctly.
		{`prefix_rule(pattern = ["gh", "pr", "view"],decision = "allow")`, "shell(gh pr view)"},
		{`prefix_rule(pattern = ["git", "fetch"], decision = "allow")`, "shell(git fetch)"},
	}

	for _, tt := range tests {
		got := convertCodexRuleToTool(tt.input)
		if got != tt.expected {
			t.Errorf("convertCodexRuleToTool(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestImportClaudeToolsFileNotFound(t *testing.T) {
	dir := t.TempDir()
	tools, err := importClaudeTools(filepath.Join(dir, "nonexistent.json"))
	if err != nil {
		t.Errorf("expected nil error for missing file, got: %v", err)
	}
	if tools != nil {
		t.Errorf("expected nil tools for missing file, got: %v", tools)
	}
}

func TestImportClaudeToolsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(p, []byte("not json"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	_, err := importClaudeTools(p)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestImportClaudeToolsNoPermissions(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	data, _ := json.Marshal(map[string]interface{}{"theme": "dark"})
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	tools, err := importClaudeTools(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected no tools when permissions node is absent, got: %v", tools)
	}
}

func TestImportClaudeToolsNoAllowList(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	data, _ := json.Marshal(map[string]interface{}{
		"permissions": map[string]interface{}{"deny": []string{"Bash(rm:*)"}},
	})
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	tools, err := importClaudeTools(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected no tools when allow list is absent, got: %v", tools)
	}
}

func TestImportClaudeToolsReadsPermissions(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	settings := map[string]interface{}{
		"theme": "dark",
		"permissions": map[string]interface{}{
			"allow": []string{"Bash(git fetch:*)", "Bash(git pull:*)"},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	tools, err := importClaudeTools(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d: %v", len(tools), tools)
	}
	if tools[0] != "shell(git fetch)" {
		t.Errorf("expected shell(git fetch), got %q", tools[0])
	}
	if tools[1] != "shell(git pull)" {
		t.Errorf("expected shell(git pull), got %q", tools[1])
	}
}

func TestImportClaudeToolsUnknownPermissionsPassedThrough(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	settings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{"mcp__github__create_issue"},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	tools, err := importClaudeTools(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 1 || tools[0] != "mcp__github__create_issue" {
		t.Errorf("expected unknown permission to pass through, got: %v", tools)
	}
}

func TestImportCodexToolsFileNotFound(t *testing.T) {
	dir := t.TempDir()
	tools, err := importCodexTools(filepath.Join(dir, "nonexistent.md"))
	if err != nil {
		t.Errorf("expected nil error for missing file, got: %v", err)
	}
	if tools != nil {
		t.Errorf("expected nil tools for missing file, got: %v", tools)
	}
}

func TestImportCodexToolsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "instructions.md")
	if err := os.WriteFile(p, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	tools, err := importCodexTools(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected no tools for empty file, got: %v", tools)
	}
}

func TestImportCodexToolsIgnoresNonRuleLines(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "instructions.md")
	content := "# Some heading\n\nSome prose text.\n\n" +
		`prefix_rule(pattern=["git", "fetch"], decision="allow")` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	tools, err := importCodexTools(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d: %v", len(tools), tools)
	}
	if tools[0] != "shell(git fetch)" {
		t.Errorf("expected shell(git fetch), got %q", tools[0])
	}
}

func TestImportCodexToolsReadsRules(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "instructions.md")
	content := `prefix_rule(pattern=["git", "fetch"], decision="allow")` + "\n" +
		`prefix_rule(pattern=["npm", "install", "--save"], decision="allow")` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	tools, err := importCodexTools(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d: %v", len(tools), tools)
	}
	if tools[0] != "shell(git fetch)" {
		t.Errorf("expected shell(git fetch), got %q", tools[0])
	}
	if tools[1] != "shell(npm install --save)" {
		t.Errorf("expected shell(npm install --save), got %q", tools[1])
	}
}

func TestImportCodexToolsHandlesSpacesAroundEquals(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "agent-align.rules")
	// Simulate a rules file written with spaces around '=' as described in the issue.
	content := `prefix_rule(pattern = ["gh", "pr", "view"],decision = "allow")` + "\n" +
		`prefix_rule(pattern = ["git", "fetch"], decision = "allow")` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	tools, err := importCodexTools(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d: %v", len(tools), tools)
	}
	if tools[0] != "shell(gh pr view)" {
		t.Errorf("expected shell(gh pr view), got %q", tools[0])
	}
	if tools[1] != "shell(git fetch)" {
		t.Errorf("expected shell(git fetch), got %q", tools[1])
	}
}

func TestCollectAllowedToolsNoClaude(t *testing.T) {
	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			Targets: config.AllowedToolsTargets{
				Agents: []config.AllowedToolsAgent{},
			},
		},
	}
	tools, err := collectAllowedTools(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected no tools, got: %v", tools)
	}
}

func TestCollectAllowedToolsSkipsCopilot(t *testing.T) {
	dir := t.TempDir()
	// A Codex file with one tool.
	codexPath := filepath.Join(dir, "instructions.md")
	if err := os.WriteFile(codexPath, []byte(`prefix_rule(pattern=["git", "fetch"], decision="allow")`+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			Targets: config.AllowedToolsTargets{
				Agents: []config.AllowedToolsAgent{
					{Name: "copilot", Path: dir}, // should be skipped
					{Name: "codex", Path: codexPath},
				},
			},
		},
	}
	tools, err := collectAllowedTools(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 1 || tools[0] != "shell(git fetch)" {
		t.Errorf("expected [shell(git fetch)], got: %v", tools)
	}
}

func TestCollectAllowedToolsMergesAndDeduplicates(t *testing.T) {
	dir := t.TempDir()

	// Claude settings with two tools, one duplicate.
	claudePath := filepath.Join(dir, "settings.json")
	settings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{"Bash(git fetch:*)", "Bash(git pull:*)"},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	if err := os.WriteFile(claudePath, data, 0o644); err != nil {
		t.Fatalf("failed to write claude file: %v", err)
	}

	// Codex file with one overlapping tool and one new tool.
	codexPath := filepath.Join(dir, "instructions.md")
	content := `prefix_rule(pattern=["git", "fetch"], decision="allow")` + "\n" +
		`prefix_rule(pattern=["npm", "install"], decision="allow")` + "\n"
	if err := os.WriteFile(codexPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write codex file: %v", err)
	}

	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			Targets: config.AllowedToolsTargets{
				Agents: []config.AllowedToolsAgent{
					{Name: "claude", Path: claudePath},
					{Name: "codex", Path: codexPath},
				},
			},
		},
	}

	tools, err := collectAllowedTools(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: 3 unique tools sorted alphabetically.
	expected := []string{"shell(git fetch)", "shell(git pull)", "shell(npm install)"}
	if len(tools) != len(expected) {
		t.Fatalf("expected %d tools, got %d: %v", len(expected), len(tools), tools)
	}
	for i, want := range expected {
		if tools[i] != want {
			t.Errorf("tools[%d] = %q, want %q", i, tools[i], want)
		}
	}
}

func TestCollectAllowedToolsSortedAlphabetically(t *testing.T) {
	dir := t.TempDir()

	claudePath := filepath.Join(dir, "settings.json")
	settings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{"Bash(npm install:*)", "Bash(git fetch:*)", "Bash(go build:*)"},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	if err := os.WriteFile(claudePath, data, 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			Targets: config.AllowedToolsTargets{
				Agents: []config.AllowedToolsAgent{
					{Name: "claude", Path: claudePath},
				},
			},
		},
	}

	tools, err := collectAllowedTools(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"shell(git fetch)", "shell(go build)", "shell(npm install)"}
	if len(tools) != len(expected) {
		t.Fatalf("expected %d tools, got %d: %v", len(expected), len(tools), tools)
	}
	for i, want := range expected {
		if tools[i] != want {
			t.Errorf("tools[%d] = %q, want %q", i, tools[i], want)
		}
	}
}

func TestCollectAllowedToolsMissingFilesLogWarning(t *testing.T) {
	dir := t.TempDir()
	// Both paths point to nonexistent files; should not error.
	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			Targets: config.AllowedToolsTargets{
				Agents: []config.AllowedToolsAgent{
					{Name: "claude", Path: filepath.Join(dir, "no-settings.json")},
					{Name: "codex", Path: filepath.Join(dir, "no-instructions.md")},
				},
			},
		},
	}
	tools, err := collectAllowedTools(cfg)
	if err != nil {
		t.Fatalf("unexpected error for missing files: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected no tools when all files are missing, got: %v", tools)
	}
}

func TestCollectAllowedToolsDefaultPathsUsedWhenNoPathSet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Write a Claude settings file at the default location.
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("failed to create .claude dir: %v", err)
	}
	settings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{"Bash(git fetch:*)"},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cfg := config.Config{
		AllowedTools: config.AllowedToolsConfig{
			Targets: config.AllowedToolsTargets{
				Agents: []config.AllowedToolsAgent{
					{Name: "claude"}, // no explicit path → uses default
				},
			},
		},
	}

	tools, err := collectAllowedTools(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 1 || tools[0] != "shell(git fetch)" {
		t.Errorf("expected [shell(git fetch)], got: %v", tools)
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

func TestMergeAllowedToolsCombinesAndDeduplicates(t *testing.T) {
	collected := []string{"shell(git fetch)", "shell(npm install)"}
	existing := []string{"shell(git pull)", "shell(git fetch)"} // git fetch is duplicate

	result := mergeAllowedTools(collected, existing)

	expected := []string{"shell(git fetch)", "shell(git pull)", "shell(npm install)"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d tools, got %d: %v", len(expected), len(result), result)
	}
	for i, want := range expected {
		if result[i] != want {
			t.Errorf("result[%d] = %q, want %q", i, result[i], want)
		}
	}
}

func TestMergeAllowedToolsPreservesExistingWhenNoCollected(t *testing.T) {
	collected := []string{}
	existing := []string{"shell(git pull)", "shell(git fetch)"}

	result := mergeAllowedTools(collected, existing)

	expected := []string{"shell(git fetch)", "shell(git pull)"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d tools, got %d: %v", len(expected), len(result), result)
	}
	for i, want := range expected {
		if result[i] != want {
			t.Errorf("result[%d] = %q, want %q", i, result[i], want)
		}
	}
}

func TestMergeAllowedToolsReturnsEmptyWhenBothEmpty(t *testing.T) {
	result := mergeAllowedTools([]string{}, []string{})
	if len(result) != 0 {
		t.Errorf("expected empty result, got: %v", result)
	}
}

func TestMergeAllowedToolsSortedAlphabetically(t *testing.T) {
	collected := []string{"shell(npm install)", "shell(go build)"}
	existing := []string{"shell(git fetch)"}

	result := mergeAllowedTools(collected, existing)

	expected := []string{"shell(git fetch)", "shell(go build)", "shell(npm install)"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d tools, got %d: %v", len(expected), len(result), result)
	}
	for i, want := range expected {
		if result[i] != want {
			t.Errorf("result[%d] = %q, want %q", i, result[i], want)
		}
	}
}
