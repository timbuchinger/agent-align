package main

import (
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
