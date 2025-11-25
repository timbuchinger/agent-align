package syncer

import (
	"reflect"
	"testing"
)

func TestParseServersFromJSON_WholePayload(t *testing.T) {
    payload := `{"server1": {"command": "npx"}}`
    got, err := parseServersFromJSON("", payload)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(got) != 1 {
        t.Fatalf("expected 1 server, got %d", len(got))
    }
    if _, ok := got["server1"]; !ok {
        t.Fatalf("expected server1 key present")
    }
}

func TestParseServersFromJSON_InvalidJSON(t *testing.T) {
    payload := `not valid json`
    _, err := parseServersFromJSON("mcpServers", payload)
    if err == nil {
        t.Fatal("expected error for invalid JSON")
    }
}

func TestFormatConfig_ReturnsEmptyOnParseError(t *testing.T) {
    // template payload is invalid JSON for a JSON-based source — formatConfig should return empty
    tpl := Template{Name: "broken", Payload: "not-json"}
    out := formatConfig("copilot", "copilot", tpl)
    if out != "" {
        t.Fatalf("expected empty output on parse failure, got %q", out)
    }
}

func TestUniqueAgents_NormalizeAndDedup(t *testing.T) {
    input := []string{" Copilot", "", "copilot", "Gemini", "gemini", "  "}
    got := uniqueAgents(input)
    want := []string{"copilot", "gemini"}
    if !reflect.DeepEqual(got, want) {
        t.Fatalf("uniqueAgents(%v) = %v, want %v", input, got, want)
    }
}

