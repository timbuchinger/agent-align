package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"agent-align/internal/config"
)

func TestJSONPathSegments(t *testing.T) {
	cases := []struct {
		path string
		want []string
	}{
		{"", nil},
		{".", nil},
		{".mcpServers", []string{"mcpServers"}},
		{"root.value", []string{"root", "value"}},
		{"nested..value", []string{"nested", "value"}},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got := jsonPathSegments(tc.path)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("jsonPathSegments(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestBuildAdditionalJSONContent_MergesWithExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "extra.json")
	if err := os.WriteFile(path, []byte(`{"keep": true}`), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	target := config.AdditionalJSONTarget{FilePath: path, JSONPath: ".mcpServers"}
	servers := map[string]interface{}{
		"beta": map[string]interface{}{"command": "node"},
	}

	content, err := buildAdditionalJSONContent(target, servers)
	if err != nil {
		t.Fatalf("buildAdditionalJSONContent returned error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := parsed["keep"]; !ok {
		t.Fatal("expected existing keys to be preserved")
	}
	if _, ok := parsed["mcpServers"]; !ok {
		t.Fatal("expected new node to be inserted")
	}
}

func TestBuildAdditionalJSONContent_RootPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "root.json")

	target := config.AdditionalJSONTarget{FilePath: path, JSONPath: ""}
	servers := map[string]interface{}{
		"delta": map[string]interface{}{"command": "npm"},
	}

	content, err := buildAdditionalJSONContent(target, servers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := parsed["delta"]; !ok {
		t.Fatal("expected root object to contain the servers map")
	}
}

func TestBuildAdditionalJSONContent_InvalidJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(path, []byte("{ invalid"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	target := config.AdditionalJSONTarget{FilePath: path, JSONPath: ".mcpServers"}
	_, err := buildAdditionalJSONContent(target, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
