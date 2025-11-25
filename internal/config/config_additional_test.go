package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestLoadMalformedYAML(t *testing.T) {
    path := writeConfigFile(t, "source: codex\ntargets: [\n")

    _, err := Load(path)
    if err == nil {
        t.Fatal("expected error for malformed YAML")
    }
    if !strings.Contains(err.Error(), "failed to parse config") {
        t.Fatalf("unexpected error message: %v", err)
    }
}

func TestLoadNormalizesAndTrims(t *testing.T) {
    path := writeConfigFile(t, `source:  CoPilot
targets:
  -  Gemini
  - CODEx
  - ""
`)

    got, err := Load(path)
    if err != nil {
        t.Fatalf("Load returned error: %v", err)
    }
    want := Config{Source: "copilot", Targets: []string{"gemini", "codex"}}
    if !reflect.DeepEqual(got, want) {
        t.Fatalf("unexpected config: %#v", got)
    }
}
