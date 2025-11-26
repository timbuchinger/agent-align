package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"agent-align/internal/config"
)

func buildAdditionalJSONContent(target config.AdditionalJSONTarget, servers map[string]interface{}) (string, error) {
	pathSegments := jsonPathSegments(target.JSONPath)
	if len(pathSegments) == 0 {
		return marshalJSON(servers)
	}

	root, err := loadJSONFile(target.FilePath)
	if err != nil {
		return "", err
	}

	mergeJSONValue(root, pathSegments, servers)
	return marshalJSON(root)
}

func loadJSONFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return make(map[string]interface{}), nil
	}

	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", path, err)
	}
	if out == nil {
		out = make(map[string]interface{})
	}
	return out, nil
}

func marshalJSON(value interface{}) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data) + "\n", nil
}

func mergeJSONValue(root map[string]interface{}, path []string, value interface{}) {
	current := root
	for i, segment := range path {
		if i == len(path)-1 {
			current[segment] = value
			return
		}
		next, ok := current[segment].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			current[segment] = next
		}
		current = next
	}
}

func jsonPathSegments(path string) []string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil
	}

	segments := strings.Split(trimmed, ".")
	var out []string
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		out = append(out, segment)
	}
	return out
}

func displayJSONPath(path string) string {
	if trimmed := strings.TrimSpace(path); trimmed != "" {
		return trimmed
	}
	return "<root>"
}
