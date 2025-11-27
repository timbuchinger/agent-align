package syncer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"agent-align/internal/transforms"
)

// AgentTarget allows overrides for an agent destination.
type AgentTarget struct {
	Name         string
	PathOverride string
}

// AgentConfig holds information about an agent's configuration file.
type AgentConfig struct {
	Name     string // Normalized agent name
	FilePath string // Path to the config file
	NodeName string // Name of the node where servers are stored
	Format   string // "json" or "toml"
}

// AgentResult is the rendered output for a single agent.
type AgentResult struct {
	Config  AgentConfig
	Content string
}

// SupportedAgents returns a list of supported agent names.
func SupportedAgents() []string {
	return []string{"copilot", "vscode", "codex", "claudecode", "gemini", "kilocode"}
}

// GetAgentConfig returns the configuration information for a given agent.
func GetAgentConfig(agent, overridePath string) (AgentConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return AgentConfig{}, fmt.Errorf("failed to get home directory: %w", err)
	}

	name := normalizeAgent(agent)
	switch name {
	case "copilot":
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, filepath.Join(homeDir, ".copilot", "mcp-config.json")),
			NodeName: "mcpServers",
			Format:   "json",
		}, nil
	case "vscode":
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, filepath.Join(homeDir, ".config", "Code", "User", "mcp.json")),
			NodeName: "servers",
			Format:   "json",
		}, nil
	case "codex":
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, filepath.Join(homeDir, ".codex", "config.toml")),
			NodeName: "",
			Format:   "toml",
		}, nil
	case "claudecode":
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, filepath.Join(homeDir, ".claude.json")),
			NodeName: "mcpServers",
			Format:   "json",
		}, nil
	case "gemini":
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, filepath.Join(homeDir, ".gemini", "settings.json")),
			NodeName: "mcpServers",
			Format:   "json",
		}, nil
	case "kilocode":
		var defaultPath string
		if runtime.GOOS == "windows" {
			defaultPath = filepath.Join(homeDir, "AppData", "Roaming", "Code", "user", "mcp.json")
		} else {
			defaultPath = filepath.Join(homeDir, ".config", "Code", "User", "globalStorage", "kilocode.kilo-code", "settings", "mcp_settings.json")
		}
		return AgentConfig{
			Name:     name,
			FilePath: applyOverride(overridePath, defaultPath),
			NodeName: "mcpServers",
			Format:   "json",
		}, nil
	default:
		return AgentConfig{}, fmt.Errorf("unsupported agent: %s", agent)
	}
}

// Syncer renders MCP server definitions into the supported agent formats.
type Syncer struct {
	Agents []AgentTarget
}

func New(agents []AgentTarget) *Syncer {
	return &Syncer{Agents: dedupeTargets(agents)}
}

// SyncResult contains the output per agent plus the parsed server data.
type SyncResult struct {
	Agents  map[string]AgentResult
	Servers map[string]interface{}
}

func (s *Syncer) Sync(servers map[string]interface{}) (SyncResult, error) {
	if len(servers) == 0 {
		return SyncResult{}, fmt.Errorf("server list cannot be empty")
	}

	outputs := make(map[string]AgentResult, len(s.Agents))
	for _, agent := range s.Agents {
		cfg, err := GetAgentConfig(agent.Name, agent.PathOverride)
		if err != nil {
			return SyncResult{}, fmt.Errorf("target agent %q not supported: %w", agent.Name, err)
		}

		agentServers, err := deepCopyServers(servers)
		if err != nil {
			return SyncResult{}, err
		}

		transformer := transforms.GetTransformer(cfg.Name)
		if err := transformer.Transform(agentServers); err != nil {
			return SyncResult{}, err
		}

		outputs[cfg.Name] = AgentResult{
			Config:  cfg,
			Content: formatConfig(cfg, agentServers),
		}
	}

	return SyncResult{Agents: outputs, Servers: servers}, nil
}

// deepCopyServers creates a deep copy of the servers map to avoid
// transformations from one agent affecting another.
func deepCopyServers(servers map[string]interface{}) (map[string]interface{}, error) {
	// Use JSON marshal/unmarshal for deep copy
	data, err := json.Marshal(servers)
	if err != nil {
		return nil, fmt.Errorf("failed to copy server configuration: %w", err)
	}
	var copy map[string]interface{}
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, fmt.Errorf("failed to copy server configuration: %w", err)
	}
	return copy, nil
}

func formatConfig(config AgentConfig, servers map[string]interface{}) string {
	if config.Format == "toml" {
		return formatCodexConfig(config, servers)
	}

	switch config.Name {
	case "gemini":
		return formatGeminiConfig(config, servers)
	default:
		return formatToJSON(config.NodeName, servers)
	}
}

// parseServersFromSource extracts MCP server definitions from the source template
func parseServersFromSource(source, payload string) (map[string]interface{}, error) {
	sourceConfig, err := GetAgentConfig(source, "")
	if err != nil {
		return nil, err
	}

	if sourceConfig.Format == "toml" {
		return parseServersFromTOML(payload)
	}
	return parseServersFromJSON(sourceConfig.NodeName, payload)
}

// parseServersFromJSON extracts servers from a JSON config
func parseServersFromJSON(nodeName, payload string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// If nodeName is empty, assume the entire payload is the servers map
	if nodeName == "" {
		return data, nil
	}

	servers, ok := data[nodeName].(map[string]interface{})
	if !ok {
		// Return empty map if node doesn't exist
		return make(map[string]interface{}), nil
	}
	return servers, nil
}

// parseServersFromTOML extracts servers from a TOML config (Codex format)
func parseServersFromTOML(payload string) (map[string]interface{}, error) {
	servers := make(map[string]interface{})
	lines := strings.Split(payload, "\n")

	var currentServer string
	var serverData map[string]interface{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for server section header [mcp_servers.servername] or [mcp_servers.servername.subsection]
		if strings.HasPrefix(line, "[mcp_servers.") && strings.HasSuffix(line, "]") {
			if currentServer != "" && serverData != nil {
				servers[currentServer] = serverData
			}
			currentServer = strings.TrimPrefix(line, "[mcp_servers.")
			currentServer = strings.TrimSuffix(currentServer, "]")
			serverData = make(map[string]interface{})
			continue
		}

		// Parse key-value pairs within a server section
		if currentServer != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Handle string values
				if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
					value = strings.Trim(value, "\"")
					serverData[key] = value
				} else if strings.HasPrefix(value, "[") {
					// Handle array values
					arr := parseTOMLArray(value)
					serverData[key] = arr
				} else {
					serverData[key] = value
				}
			}
		}
	}

	// Don't forget the last server
	if currentServer != "" && serverData != nil {
		servers[currentServer] = serverData
	}

	// Merge nested sections (e.g., "github.env" becomes nested under "github" as "env")
	servers = mergeNestedTOMLSections(servers)

	return servers, nil
}

// mergeNestedTOMLSections converts flat dotted section names into nested structures.
// For example, "github.env" with data {KEY: "value"} gets merged into
// "github" as {env: {KEY: "value"}}.
func mergeNestedTOMLSections(servers map[string]interface{}) map[string]interface{} {
	// Collect all keys that contain dots (subsections)
	var subsectionNames []string
	for name := range servers {
		if strings.Contains(name, ".") {
			subsectionNames = append(subsectionNames, name)
		}
	}

	// If no subsections, return as-is
	if len(subsectionNames) == 0 {
		return servers
	}

	// Sort subsection names for deterministic processing order
	sort.Strings(subsectionNames)

	// Process subsections and merge them into parent servers
	for _, subsectionName := range subsectionNames {
		parts := strings.SplitN(subsectionName, ".", 2)
		if len(parts) != 2 {
			continue
		}

		parentName := parts[0]
		childKey := parts[1]
		subsectionData := servers[subsectionName]

		// Ensure parent exists
		parent, parentExists := servers[parentName]
		if !parentExists {
			parent = make(map[string]interface{})
			servers[parentName] = parent
		}

		// Convert parent to map if needed (this handles edge case where parent
		// was defined but is not a map - in TOML context this shouldn't happen
		// for well-formed configs, but we handle it gracefully)
		parentMap, ok := parent.(map[string]interface{})
		if !ok {
			parentMap = make(map[string]interface{})
			servers[parentName] = parentMap
		}

		// Handle nested subsections (e.g., "github.env.nested" -> github.env.nested)
		if strings.Contains(childKey, ".") {
			// Recursively build the nested structure
			setNestedValue(parentMap, childKey, subsectionData)
		} else {
			// Simple case: set the child key directly
			parentMap[childKey] = subsectionData
		}

		// Remove the flat subsection key
		delete(servers, subsectionName)
	}

	return servers
}

// setNestedValue sets a value at a nested path within a map.
// For example, setNestedValue(m, "env.nested", data) creates m["env"]["nested"] = data
func setNestedValue(m map[string]interface{}, path string, value interface{}) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 1 {
		m[parts[0]] = value
		return
	}

	// Ensure intermediate map exists
	child, exists := m[parts[0]]
	if !exists {
		child = make(map[string]interface{})
		m[parts[0]] = child
	}

	// If existing value is not a map, replace it (this handles edge cases
	// where a simple value exists at an intermediate path level)
	childMap, ok := child.(map[string]interface{})
	if !ok {
		childMap = make(map[string]interface{})
		m[parts[0]] = childMap
	}

	setNestedValue(childMap, parts[1], value)
}

// parseTOMLArray parses a simple TOML array like ["a", "b", "c"]
// This handles basic quoted strings but does not support escaped quotes within values
func parseTOMLArray(value string) []string {
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	value = strings.TrimSpace(value)

	if value == "" {
		return nil
	}

	var result []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(value); i++ {
		ch := value[i]
		switch {
		case ch == '"' && !inQuotes:
			inQuotes = true
		case ch == '"' && inQuotes:
			inQuotes = false
			result = append(result, current.String())
			current.Reset()
		case ch == ',' && !inQuotes:
			// Skip commas outside quotes (between elements)
			continue
		case inQuotes:
			current.WriteByte(ch)
		}
	}

	return result
}

// formatToJSON converts servers to JSON format with the specified node name
func formatGeminiConfig(cfg AgentConfig, servers map[string]interface{}) string {
	var existing map[string]interface{}
	if data, err := os.ReadFile(cfg.FilePath); err == nil {
		if err := json.Unmarshal(data, &existing); err != nil {
			existing = make(map[string]interface{})
		}
	}
	if existing == nil {
		existing = make(map[string]interface{})
	}

	existing[cfg.NodeName] = servers
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

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

// formatToTOML converts servers to Codex TOML format
func formatToTOML(servers map[string]interface{}) string {
	var sb strings.Builder

	// Sort server names for consistent output
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		serverData, ok := servers[name].(map[string]interface{})
		if !ok {
			continue
		}

		formatServerToTOML(&sb, "mcp_servers."+name, serverData)
	}

	return strings.TrimRight(sb.String(), "\n")
}

// formatServerToTOML recursively formats a server and its nested sections to TOML
func formatServerToTOML(sb *strings.Builder, sectionPath string, data map[string]interface{}) {
	// Separate nested maps from simple values
	simpleValues := make(map[string]interface{})
	nestedMaps := make(map[string]map[string]interface{})

	for k, v := range data {
		if nested, ok := v.(map[string]interface{}); ok {
			nestedMaps[k] = nested
		} else {
			simpleValues[k] = v
		}
	}

	// Write the section header and simple values
	sb.WriteString(fmt.Sprintf("[%s]\n", sectionPath))

	// Sort keys for consistent output
	keys := make([]string, 0, len(simpleValues))
	for k := range simpleValues {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := simpleValues[k]
		switch val := v.(type) {
		case string:
			sb.WriteString(fmt.Sprintf("%s = \"%s\"\n", k, val))
		case []interface{}:
			arr := make([]string, 0, len(val))
			for _, item := range val {
				if s, ok := item.(string); ok {
					arr = append(arr, fmt.Sprintf("\"%s\"", s))
				}
			}
			sb.WriteString(fmt.Sprintf("%s = [%s]\n", k, strings.Join(arr, ", ")))
		case []string:
			arr := make([]string, 0, len(val))
			for _, s := range val {
				arr = append(arr, fmt.Sprintf("\"%s\"", s))
			}
			sb.WriteString(fmt.Sprintf("%s = [%s]\n", k, strings.Join(arr, ", ")))
		default:
			sb.WriteString(fmt.Sprintf("%s = %v\n", k, val))
		}
	}
	sb.WriteString("\n")

	// Sort nested map keys for consistent output
	nestedKeys := make([]string, 0, len(nestedMaps))
	for k := range nestedMaps {
		nestedKeys = append(nestedKeys, k)
	}
	sort.Strings(nestedKeys)

	// Recursively format nested maps as separate sections
	for _, k := range nestedKeys {
		formatServerToTOML(sb, sectionPath+"."+k, nestedMaps[k])
	}
}

func formatCodexConfig(cfg AgentConfig, servers map[string]interface{}) string {
	var existing string
	if data, err := os.ReadFile(cfg.FilePath); err == nil {
		existing = string(data)
	}

	preserved := strings.TrimRight(stripMCPServersSections(existing), "\r\n")
	newSections := strings.TrimRight(formatToTOML(servers), "\r\n")

	var parts []string
	if preserved != "" {
		parts = append(parts, preserved)
	}
	if newSections != "" {
		parts = append(parts, newSections)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n\n") + "\n"
}

func stripMCPServersSections(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	var sb strings.Builder
	insideMCP := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if strings.HasPrefix(trimmed, "[mcp_servers.") {
				insideMCP = true
				continue
			}
			insideMCP = false
		}
		if insideMCP {
			continue
		}
		sb.WriteString(line)
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func normalizeAgent(agent string) string {
	return strings.ToLower(strings.TrimSpace(agent))
}

func dedupeTargets(targets []AgentTarget) []AgentTarget {
	seen := make(map[string]struct{}, len(targets))
	var out []AgentTarget
	for _, target := range targets {
		name := normalizeAgent(target.Name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, AgentTarget{
			Name:         name,
			PathOverride: strings.TrimSpace(target.PathOverride),
		})
	}
	return out
}

func applyOverride(overridePath, defaultPath string) string {
	if trimmed := strings.TrimSpace(overridePath); trimmed != "" {
		return trimmed
	}
	return defaultPath
}
