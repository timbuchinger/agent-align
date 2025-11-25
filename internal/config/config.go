package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config describes the source and target agents defined in the YAML file.
type Config struct {
	Source   string   `yaml:"source"`
	Targets  []string `yaml:"targets"`
	Template string   `yaml:"template"`
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

	cfg.Source = normalizeAgent(cfg.Source)
	if cfg.Source == "" {
		return Config{}, fmt.Errorf("config at %q must define source", path)
	}
	cfg.Template = strings.TrimSpace(cfg.Template)
	if cfg.Template == "" {
		return Config{}, fmt.Errorf("config at %q must define template path", path)
	}

	cleanTargets := make([]string, 0, len(cfg.Targets))
	for _, target := range cfg.Targets {
		trimmed := normalizeAgent(target)
		if trimmed == "" {
			continue
		}
		cleanTargets = append(cleanTargets, trimmed)
	}

	if len(cleanTargets) == 0 {
		return Config{}, fmt.Errorf("config at %q must define at least one target", path)
	}
	for _, target := range cleanTargets {
		if target == cfg.Source {
			return Config{}, fmt.Errorf("config at %q lists %q as both source and target; remove it from targets", path, target)
		}
	}

	cfg.Targets = cleanTargets
	return cfg, nil
}

func normalizeAgent(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
