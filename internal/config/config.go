package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config describes the source agent and potential target destinations.
type Config struct {
	SourceAgent string        `yaml:"sourceAgent"`
	Targets     TargetsConfig `yaml:"targets"`
}

// TargetsConfig groups agent targets and additional destinations.
type TargetsConfig struct {
	Agents     []string          `yaml:"agents"`
	Additional AdditionalTargets `yaml:"additional"`
}

// AdditionalTargets lists paths for JSON-style destinations.
type AdditionalTargets struct {
	JSON []AdditionalJSONTarget `yaml:"json"`
}

// AdditionalJSONTarget describes a JSON file that should receive the MCP payload.
type AdditionalJSONTarget struct {
	FilePath string `yaml:"filePath"`
	JSONPath string `yaml:"jsonPath"`
}

// Load reads the YAML configuration from the given path and validates it.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config at %q: %w", path, err)
	}

	cfg.SourceAgent = normalizeAgent(cfg.SourceAgent)
	if cfg.SourceAgent == "" {
		return Config{}, fmt.Errorf("config at %q must define a source agent", path)
	}

	cleanAgents := make([]string, 0, len(cfg.Targets.Agents))
	for _, target := range cfg.Targets.Agents {
		trimmed := normalizeAgent(target)
		if trimmed == "" {
			continue
		}
		cleanAgents = append(cleanAgents, trimmed)
	}

	if len(cleanAgents) == 0 && len(cfg.Targets.Additional.JSON) == 0 {
		return Config{}, fmt.Errorf("config at %q must define at least one target", path)
	}

	for _, target := range cleanAgents {
		if target == cfg.SourceAgent {
			return Config{}, fmt.Errorf("config at %q lists %q as both source and target; remove it from targets", path, target)
		}
	}

	cfg.Targets.Agents = cleanAgents
	for i := range cfg.Targets.Additional.JSON {
		cfg.Targets.Additional.JSON[i].FilePath = strings.TrimSpace(cfg.Targets.Additional.JSON[i].FilePath)
		cfg.Targets.Additional.JSON[i].JSONPath = strings.TrimSpace(cfg.Targets.Additional.JSON[i].JSONPath)
		if cfg.Targets.Additional.JSON[i].FilePath == "" {
			return Config{}, fmt.Errorf("config at %q has an additional JSON target without a filePath", path)
		}
	}

	return cfg, nil
}

func normalizeAgent(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// UnmarshalYAML supports the legacy `source` field beside the new `sourceAgent`.
func (c *Config) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		return nil
	}

	type rawConfig struct {
		Source      string        `yaml:"source"`
		SourceAgent string        `yaml:"sourceAgent"`
		Targets     TargetsConfig `yaml:"targets"`
	}

	var raw rawConfig
	if err := node.Decode(&raw); err != nil {
		return err
	}

	c.Targets = raw.Targets
	if raw.SourceAgent != "" {
		c.SourceAgent = raw.SourceAgent
	} else {
		c.SourceAgent = raw.Source
	}
	return nil
}

// UnmarshalYAML accepts both the legacy target list and the new mapping.
func (t *TargetsConfig) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.SequenceNode:
		var agents []string
		if err := node.Decode(&agents); err != nil {
			return err
		}
		t.Agents = agents
		return nil
	case yaml.MappingNode:
		type rawTargets struct {
			Agents     []string          `yaml:"agents"`
			Additional AdditionalTargets `yaml:"additional"`
		}
		var raw rawTargets
		if err := node.Decode(&raw); err != nil {
			return err
		}
		t.Agents = raw.Agents
		t.Additional = raw.Additional
		return nil
	default:
		return fmt.Errorf("unexpected targets format, expected sequence or mapping")
	}
}
