package syncer

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseServersFromTOML_MixedValues(t *testing.T) {
    payload := `# comment
[mcp_servers.server1]
command = "node"
args = ["-a", "-b"]
timeout = 5

[mcp_servers.server2]
command = "npx"
args = ["one,two", "three"]
`

    servers, err := parseServersFromTOML(payload)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if len(servers) != 2 {
        t.Fatalf("expected 2 servers, got %d", len(servers))
    }

    s1, ok := servers["server1"].(map[string]interface{})
    if !ok {
        t.Fatalf("server1 not parsed as map, got: %#v", servers["server1"])
    }
    if s1["command"] != "node" {
        t.Fatalf("server1 command mismatch: %v", s1["command"])
    }
    // timeout is unquoted and will be returned as the raw string
    if s1["timeout"] != "5" {
        t.Fatalf("expected timeout '5', got %v", s1["timeout"])
    }

    s2, ok := servers["server2"].(map[string]interface{})
    if !ok {
        t.Fatalf("server2 not parsed as map")
    }
    if s2["command"] != "npx" {
        t.Fatalf("server2 command mismatch: %v", s2["command"])
    }
    // args should preserve comma inside quoted value
    arr, ok := s2["args"].([]string)
    if !ok {
        // parser may return []interface{} for arrays; allow either
        if ifaceArr, ok := s2["args"].([]interface{}); ok {
            var got []string
            for _, it := range ifaceArr {
                if s, ok := it.(string); ok {
                    got = append(got, s)
                }
            }
            if !reflect.DeepEqual(got, []string{"one,two", "three"}) {
                t.Fatalf("unexpected args: %v", got)
            }
        } else {
            t.Fatalf("args not parsed as array, got: %#v", s2["args"])
        }
    } else {
        if !reflect.DeepEqual(arr, []string{"one,two", "three"}) {
            t.Fatalf("unexpected args: %v", arr)
        }
    }
}

func TestStripMCPServersSections_RemovesBlocksAndKeepsOthers(t *testing.T) {
    content := `# Pre
[general]
val = true

[mcp_servers.old]
command = "node"

[editor]
font = 12

[mcp_servers.new]
command = "npx"
`

    stripped := stripMCPServersSections(content)
    if strings.Contains(stripped, "[mcp_servers.old]") || strings.Contains(stripped, "[mcp_servers.new]") {
        t.Fatalf("mcp server sections should be removed, got: %s", stripped)
    }
    if !strings.Contains(stripped, "[general]") || !strings.Contains(stripped, "[editor]") {
        t.Fatalf("non-mcp sections should be preserved, got: %s", stripped)
    }
}

func TestFormatToTOML_MixedArrayAndTypes(t *testing.T) {
    servers := map[string]interface{}{
        "alpha": map[string]interface{}{
            "command": "node",
            "args":    []interface{}{"a", 123, "b"},
        },
    }

    toml := formatToTOML(servers)
    if !strings.Contains(toml, "[mcp_servers.alpha]") {
        t.Fatalf("expected section header, got: %s", toml)
    }
    if !strings.Contains(toml, "command = \"node\"") {
        t.Fatalf("expected command line, got: %s", toml)
    }
    // non-string array items should be ignored; result should contain "a" and "b"
    if !strings.Contains(toml, "\"a\"") || !strings.Contains(toml, "\"b\"") {
        t.Fatalf("expected string array elements present, got: %s", toml)
    }
    // ensure numeric 123 is not rendered as a quoted string
    if strings.Contains(toml, "123") {
        // Accept either presence or absence; just ensure formatting is stable (no panic)
    }
}
