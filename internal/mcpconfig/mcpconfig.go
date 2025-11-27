package mcpconfig

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads the MCP server definitions from a YAML file.
// It accepts either a top-level "servers" or "mcpServers" mapping.
func Load(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Servers    map[string]interface{} `yaml:"servers"`
		MCPServers map[string]interface{} `yaml:"mcpServers"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config at %q: %w", path, err)
	}

	servers := raw.Servers
	if len(servers) == 0 {
		servers = raw.MCPServers
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("no MCP servers found in %s", path)
	}

	for name, server := range servers {
		if _, ok := server.(map[string]interface{}); !ok {
			return nil, fmt.Errorf("server %q must be a mapping", name)
		}
	}

	return servers, nil
}
