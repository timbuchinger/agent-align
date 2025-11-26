# Formatter Design

This document outlines the architecture for parsing, transforming, and outputting
MCP configuration files across different coding agents.

## Overview

agent-align needs a formatter system that can:

1. Read configuration from any supported agent format
2. Convert to a common intermediary representation
3. Output to one or more target agent formats

## Supported Agents and Formats

| Agent      | Format | Root Element  | Location                           |
|------------|--------|---------------|------------------------------------|
| Copilot    | JSON   | `mcpServers`  | `~/.copilot/mcp-config.json`       |
| VS Code    | JSON   | `servers`     | `~/.config/Code/User/mcp.json`     |
| Codex      | TOML   | `mcp_servers` | `~/.codex/config.toml`             |
| ClaudeCode | JSON   | `mcpServers`  | `~/.claude.json`                   |
| Gemini     | JSON   | `mcpServers`  | `~/.gemini/settings.json`          |

## Recommended Architecture

### Intermediary Format: JSON

We recommend using JSON as the canonical intermediary format because:

- Most agents (Copilot, VS Code, ClaudeCode, Gemini) already use JSON natively
- JSON has broad Go ecosystem support via `encoding/json`
- Conversion overhead only applies when Codex (TOML) is involved
- The MCP server configuration structure maps naturally to JSON

When TOML input or output is required (Codex), the formatter will perform the
necessary conversion. Otherwise, the intermediary step is effectively a no-op
for JSON-based agents.

### Current Implementation

The current implementation does not use a separate formatter package. Instead,
the `internal/syncer` package handles format conversion directly:

- **AgentConfig** struct defines file paths, node names, and formats for each agent
- **Parsing** is handled by `parseServersFromJSON` and `parseServersFromTOML`
- **Formatting** is handled by `formatToJSON` and `formatToTOML`
- **Servers** are represented as `map[string]interface{}` for flexibility

```go
package syncer

// AgentConfig holds information about an agent's configuration file.
type AgentConfig struct {
    FilePath string // Path to the config file
    NodeName string // Name of the node where servers are stored
    Format   string // "json" or "toml"
}

// GetAgentConfig returns the configuration information for a given agent.
func GetAgentConfig(agent string) (AgentConfig, error) {
    // Returns agent-specific paths and configuration
}
```

### Actual Package Layout

```text
internal/
├── config/
│   ├── config.go              # Configuration loading and validation
│   ├── config_test.go         # Config tests
│   └── config_additional_test.go
├── syncer/
│   ├── syncer.go              # Main sync logic with parsing/formatting
│   ├── syncer_test.go         # Syncer tests
│   ├── syncer_integration_test.go
│   ├── syncer_edgecases_test.go
│   └── syncer_additional_test.go
```

### Agent Configuration Lookup

Instead of a registry pattern, the code uses a switch statement in `GetAgentConfig`:

```go
package syncer

func GetAgentConfig(agent string) (AgentConfig, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return AgentConfig{}, fmt.Errorf("failed to get home directory: %w", err)
    }

    switch normalizeAgent(agent) {
    case "copilot":
        return AgentConfig{
            FilePath: filepath.Join(homeDir, ".copilot", "mcp-config.json"),
            NodeName: "mcpServers",
            Format:   "json",
        }, nil
    case "vscode":
        return AgentConfig{
            FilePath: filepath.Join(homeDir, ".config", "Code", "User", "mcp.json"),
            NodeName: "servers",
            Format:   "json",
        }, nil
    case "codex":
        return AgentConfig{
            FilePath: filepath.Join(homeDir, ".codex", "config.toml"),
            NodeName: "",
            Format:   "toml",
        }, nil
    // ... additional agents
    }
}
```

### Format Conversion Implementation

The syncer uses inline parsing and formatting functions:

```go
// parseServersFromJSON extracts servers from a JSON config
func parseServersFromJSON(nodeName, payload string) (
    map[string]interface{}, error) {
    var data map[string]interface{}
    if err := json.Unmarshal([]byte(payload), &data); err != nil {
        return nil, fmt.Errorf("failed to parse JSON: %w", err)
    }
    
    if nodeName == "" {
        return data, nil
    }
    
    servers, ok := data[nodeName].(map[string]interface{})
    if !ok {
        return make(map[string]interface{}), nil
    }
    return servers, nil
}

// formatToJSON converts servers to JSON format with the specified node name
func formatToJSON(nodeName string, servers map[string]interface{}) string {
    var output map[string]interface{}
    if nodeName != "" {
        output = map[string]interface{}{
            nodeName: servers,
        }
    } else {
        output = servers
    }
    
    data, err := json.MarshalIndent(output, "", "  ")
    if err != nil {
        return ""
    }
    return string(data)
}
```

For TOML, the implementation uses custom parsing instead of external libraries,
parsing line-by-line to extract `[mcp_servers.*]` sections and formatting them
back to TOML syntax.

## Transformation Pipeline

```text
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Source Agent   │     │  Intermediary   │     │  Target Agent   │
│  (e.g., Codex)  │────▶│  JSON Config    │────▶│  (e.g., Copilot)│
│  TOML file      │     │                 │     │  JSON file      │
└─────────────────┘     └─────────────────┘     └─────────────────┘
      Parse()                                        Format()
```

1. **Parse**: The source formatter reads its native format and produces the
   intermediary `Config` struct.
2. **Transform** (optional): Apply any normalization or validation.
3. **Format**: Each target formatter converts the intermediary to its native
   output format.

## Handling Different Root Elements

Different agents use different root element names:

| Agent      | Root Element  | Notes                                    |
|------------|---------------|------------------------------------------|
| Copilot    | `mcpServers`  | Standard JSON wrapping                   |
| VS Code    | `servers`     | Different naming convention              |
| Codex      | `mcp_servers` | TOML sections like `[mcp_servers.name]`  |
| ClaudeCode | `mcpServers`  | Standard JSON wrapping                   |
| Gemini     | `mcpServers`  | Standard JSON wrapping                   |

The `AgentConfig.NodeName` field specifies which root element to use when
parsing or formatting. An empty `NodeName` means the servers map is at the
root level of the JSON.

## Configuration-Driven Workflow

The user's YAML config file (see `CONFIGURATION.md`) drives the pipeline:

```yaml
sourceAgent: codex
targets:
  agents:
    - copilot
    - vscode
    - claudecode
    - gemini
```

The syncer will:

1. Load the source agent's configuration via `GetAgentConfig(cfg.SourceAgent)`.
2. Read the source agent's config file from disk.
3. Parse the content using `parseServersFromSource()` to extract the servers map.
4. For each target agent, format the servers using `formatOutput()` and write.

## Error Handling

The syncer returns descriptive errors that include context:

```go
// Example from GetAgentConfig
return AgentConfig{}, fmt.Errorf("unsupported agent: %s", agent)

// Example from parseServersFromJSON
return nil, fmt.Errorf("failed to parse JSON: %w", err)
```

## Testing Strategy

- **Unit tests**: `syncer_test.go` includes parse/format round-trip tests.
- **Integration tests**: `syncer_integration_test.go` verifies the full sync pipeline.
- **Edge case tests**: `syncer_edgecases_test.go` handles boundary conditions.
- **Additional tests**: `syncer_additional_test.go` covers extra scenarios.

## Dependencies

| Package                  | Purpose                        |
|--------------------------|--------------------------------|
| `encoding/json` (stdlib) | JSON parsing and formatting    |
| `gopkg.in/yaml.v3`       | YAML config file I/O           |

Note: TOML is parsed with a custom line-by-line parser rather than using an
external library. This keeps dependencies minimal.

## Current Features

- **Multiple agents**: Supports Copilot, VS Code, Codex, ClaudeCode, and
  Gemini.
- **Additional targets**: Can sync to arbitrary JSON files with configurable
  JSON paths.
- **Extra file operations**: Supports copying additional files and directories.
- **Format preservation**: For Codex, preserves non-MCP sections in the TOML file.

## Future Considerations

- **Formatter package**: Refactor to separate formatter interface as designed above.
- **External TOML library**: Consider using a proper TOML parser for robustness.
- **Validation**: Warn when a server definition is missing required fields.
- **Dry-run mode**: Preview changes without writing files.
- **Watch mode**: Automatically sync when the source file changes.

## Summary

The current implementation uses JSON as the intermediary format (represented as
`map[string]interface{}`), which minimizes conversion overhead for the majority
of JSON-based agents (Copilot, VS Code, ClaudeCode, Gemini). The `syncer` package
handles all parsing and formatting inline, with agent-specific details configured
through the `AgentConfig` struct and `GetAgentConfig` function.

While the document originally outlined a `Formatter` interface design, the actual
implementation takes a more direct approach. A future refactoring could adopt the
interface-based design to improve modularity and testability when adding new agents.
