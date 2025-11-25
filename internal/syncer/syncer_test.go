package syncer

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestSyncer_Sync(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        s := New("Copilot", []string{"Copilot", "Codex"})
        template := Template{Name: "global-mcp", Payload: "payload"}

        got, err := s.Sync(template)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if len(got) != 2 {
            t.Fatalf("expected 2 agents, got %d", len(got))
        }
        tests := map[string]string{
            "copilot": "agent=copilot;source=copilot;name=global-mcp;payload=payload",
            "codex":   "agent=codex;source=copilot;name=global-mcp;payload=payload",
        }
        for agent, expected := range tests {
            if got[agent] != expected {
                t.Errorf("converted %s = %q, want %q", agent, got[agent], expected)
            }
        }
    })

    t.Run("missing template payload", func(t *testing.T) {
        s := New("copilot", []string{"copilot"})
        _, err := s.Sync(Template{Name: "empty", Payload: ""})
        if err == nil {
            t.Fatal("expected error for empty payload")
        }
    })

    t.Run("source agent not registered", func(t *testing.T) {
        s := New("claude", []string{"copilot"})
        _, err := s.Sync(Template{Name: "payload", Payload: "payload"})
        if err == nil || !strings.Contains(err.Error(), "source agent") {
            t.Fatalf("unexpected error state: %v", err)
        }
    })
}

func TestLoadTemplateFromFile(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "template.txt")
    if err := os.WriteFile(path, []byte("  contents with whitespace\n"), 0o644); err != nil {
        t.Fatalf("failed to write template: %v", err)
    }

    tpl, err := LoadTemplateFromFile(path)
    if err != nil {
        t.Fatalf("failed to load template: %v", err)
    }

    if tpl.Name != "template.txt" {
        t.Fatalf("unexpected template name %q", tpl.Name)
    }
    if tpl.Payload != "contents with whitespace" {
        t.Fatalf("unexpected payload %q", tpl.Payload)
    }
}
