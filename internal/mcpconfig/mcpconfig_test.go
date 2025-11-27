package mcpconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.yml")
	content := `servers:
  test:
    command: npx
    args: ["tool"]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 server, got %d", len(got))
	}
	if _, ok := got["test"]; !ok {
		t.Fatalf("expected test server present")
	}
}

func TestLoadMissingServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.yml")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected error for missing servers")
	}
}
