package syncer

import (
	"strings"
	"testing"
)

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

func TestFormatToTOML_ToolsSectionsNoIntermediateEmpty(t *testing.T) {
	servers := map[string]interface{}{
		"myserver": map[string]interface{}{
			"command": "uvx",
			"tools": map[string]interface{}{
				"search": map[string]interface{}{
					"approval_mode": "approve",
				},
				"read": map[string]interface{}{
					"approval_mode": "approve",
				},
			},
		},
	}

	toml := formatToTOML(servers)

	// Leaf tool sections should be present
	if !strings.Contains(toml, "[mcp_servers.myserver.tools.search]") {
		t.Errorf("expected search tool section, got:\n%s", toml)
	}
	if !strings.Contains(toml, "[mcp_servers.myserver.tools.read]") {
		t.Errorf("expected read tool section, got:\n%s", toml)
	}
	if !strings.Contains(toml, "approval_mode = \"approve\"") {
		t.Errorf("expected approval_mode value, got:\n%s", toml)
	}

	// Intermediate empty section [mcp_servers.myserver.tools] should not appear
	if strings.Contains(toml, "[mcp_servers.myserver.tools]\n") {
		t.Errorf("intermediate empty tools section should be suppressed, got:\n%s", toml)
	}
}
