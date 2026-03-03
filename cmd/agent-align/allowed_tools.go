package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"agent-align/internal/config"
)

// generateCopilotWrapper creates wrapper scripts for each copilot agent at
// their specified paths (or ~/.local/bin by default) that pre-approves
// the configured allowed tools before invoking the real copilot CLI.
func generateCopilotWrapper(cfg config.Config) error {
	if len(cfg.AllowedTools.AlwaysAllowedTools) == 0 {
		// No allowed tools configured, skip wrapper generation
		return nil
	}

	// Find the real copilot binary
	copilotPath, err := exec.LookPath("copilot")
	if err != nil {
		// Copilot not installed, skip silently
		return nil
	}

	// Build the wrapper script content once
	script := buildWrapperScript(copilotPath, cfg.AllowedTools.AlwaysAllowedTools)

	// Get home directory for default path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	defaultBinDir := filepath.Join(homeDir, ".local", "bin")

	// Generate wrapper for each configured copilot agent
	for _, agent := range cfg.AllowedTools.Targets.Agents {
		if agent.Name != "copilot" {
			continue
		}

		// Use agent-specific path if provided, otherwise use default
		binDir := defaultBinDir
		if agent.Path != "" {
			binDir = agent.Path
		}

		// Create bin directory if it doesn't exist
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", binDir, err)
		}

		// Write the wrapper script with agent name as filename
		wrapperPath := filepath.Join(binDir, "acp")
		if err := os.WriteFile(wrapperPath, []byte(script), 0o755); err != nil {
			return fmt.Errorf("failed to write wrapper script to %s: %w", wrapperPath, err)
		}
	}

	return nil
}

// convertToolToClaudePermission converts a tool string like "shell(git fetch)"
// to the Claude permissions format "Bash(git fetch:*)".
func convertToolToClaudePermission(tool string) string {
	if strings.HasPrefix(tool, "shell(") && strings.HasSuffix(tool, ")") {
		inner := tool[len("shell(") : len(tool)-1]
		return "Bash(" + inner + ":*)"
	}
	return tool
}

// generateClaudePermissions merges allowed tools into the Claude settings.json
// by replacing the permissions.allow node while preserving other nodes.
func generateClaudePermissions(cfg config.Config) error {
	if len(cfg.AllowedTools.AlwaysAllowedTools) == 0 {
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	defaultPath := filepath.Join(homeDir, ".claude", "settings.json")

	for _, agent := range cfg.AllowedTools.Targets.Agents {
		if agent.Name != "claude" {
			continue
		}

		settingsPath := defaultPath
		if agent.Path != "" {
			settingsPath = agent.Path
		}

		// Build the permissions allow list
		allowList := make([]string, 0, len(cfg.AllowedTools.AlwaysAllowedTools))
		for _, tool := range cfg.AllowedTools.AlwaysAllowedTools {
			allowList = append(allowList, convertToolToClaudePermission(tool))
		}

		// Read existing settings if present
		var existing map[string]interface{}
		if data, err := os.ReadFile(settingsPath); err == nil {
			if err := json.Unmarshal(data, &existing); err != nil {
				log.Printf("warning: failed to parse existing Claude settings %q: %v; overwriting permissions node", settingsPath, err)
				existing = make(map[string]interface{})
			}
		}
		if existing == nil {
			existing = make(map[string]interface{})
		}

		// Replace permissions.allow while preserving other keys in permissions
		permissions, _ := existing["permissions"].(map[string]interface{})
		if permissions == nil {
			permissions = make(map[string]interface{})
		}
		permissions["allow"] = allowList
		existing["permissions"] = permissions

		data, err := json.MarshalIndent(existing, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal Claude settings: %w", err)
		}

		if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", settingsPath, err)
		}
		if err := os.WriteFile(settingsPath, append(data, '\n'), 0o644); err != nil {
			return fmt.Errorf("failed to write Claude settings to %s: %w", settingsPath, err)
		}
	}

	return nil
}

// convertToolToCodexRule converts a tool string like "shell(git fetch)"
// to the Codex prefix_rule format `prefix_rule(pattern=["git", "fetch"], decision="allow")`.
func convertToolToCodexRule(tool string) string {
	if strings.HasPrefix(tool, "shell(") && strings.HasSuffix(tool, ")") {
		inner := tool[len("shell(") : len(tool)-1]
		parts := strings.Fields(inner)
		quoted := make([]string, len(parts))
		for i, p := range parts {
			quoted[i] = `"` + p + `"`
		}
		return `prefix_rule(pattern=[` + strings.Join(quoted, ", ") + `], decision="allow")`
	}
	return tool
}

// generateCodexRules overwrites a Codex rules file with prefix_rule entries
// derived from the configured allowed tools.
func generateCodexRules(cfg config.Config) error {
	if len(cfg.AllowedTools.AlwaysAllowedTools) == 0 {
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	defaultPath := filepath.Join(homeDir, ".codex", "instructions.md")

	for _, agent := range cfg.AllowedTools.Targets.Agents {
		if agent.Name != "codex" {
			continue
		}

		rulesPath := defaultPath
		if agent.Path != "" {
			rulesPath = agent.Path
		}

		var sb strings.Builder
		for _, tool := range cfg.AllowedTools.AlwaysAllowedTools {
			sb.WriteString(convertToolToCodexRule(tool))
			sb.WriteByte('\n')
		}

		if err := os.MkdirAll(filepath.Dir(rulesPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", rulesPath, err)
		}
		if err := os.WriteFile(rulesPath, []byte(sb.String()), 0o644); err != nil {
			return fmt.Errorf("failed to write Codex rules to %s: %w", rulesPath, err)
		}
	}

	return nil
}

// buildWrapperScript generates the shell script content for the copilot wrapper.
func buildWrapperScript(copilotPath string, tools []string) string {
	var sb strings.Builder

	// Write the shebang and header
	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("# Generated by agent-align - do not edit manually\n")
	sb.WriteString(copilotPath)

	// Add each allowed tool as a --allow-tool flag
	for _, tool := range tools {
		sb.WriteString(" --allow-tool '")
		// Escape single quotes in the tool string
		escaped := strings.ReplaceAll(tool, "'", "'\\''")
		sb.WriteString(escaped)
		sb.WriteString("'")
	}

	// Pass through all remaining arguments
	sb.WriteString(` "$@"`)
	sb.WriteString("\n")

	return sb.String()
}
