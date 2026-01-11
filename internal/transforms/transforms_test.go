package transforms

import (
	"strings"
	"testing"
)

func TestGetTransformer(t *testing.T) {
	tests := []struct {
		name         string
		agent        string
		wantCopilot  bool
		wantCodex    bool
		wantClaude   bool
		wantGemini   bool
		wantOpenCode bool
	}{
		{"copilot", "copilot", true, false, false, false, false},
		{"copilot spaced", " copilot ", true, false, false, false, false},
		{"codex", "codex", false, true, false, false, false},
		{"codex spaced", " codex ", false, true, false, false, false},
		{"claude", "claudecode", false, false, true, false, false},
		{"gemini", "gemini", false, false, false, true, false},
		{"gemini spaced", " gemini ", false, false, false, true, false},
		{"opencode", "opencode", false, false, false, false, true},
		{"opencode spaced", " opencode ", false, false, false, false, true},
		{"default", "vscode", false, false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTransformer(tt.agent)
			_, isCopilot := got.(*CopilotTransformer)
			_, isCodex := got.(*CodexTransformer)
			_, isClaude := got.(*ClaudeTransformer)
			_, isGemini := got.(*GeminiTransformer)
			_, isOpenCode := got.(*OpenCodeTransformer)

			if isCopilot != tt.wantCopilot {
				t.Fatalf("expected copilot=%v, got %v", tt.wantCopilot, isCopilot)
			}
			if isCodex != tt.wantCodex {
				t.Fatalf("expected codex=%v, got %v", tt.wantCodex, isCodex)
			}
			if isClaude != tt.wantClaude {
				t.Fatalf("expected claude=%v, got %v", tt.wantClaude, isClaude)
			}
			if isGemini != tt.wantGemini {
				t.Fatalf("expected gemini=%v, got %v", tt.wantGemini, isGemini)
			}
			if isOpenCode != tt.wantOpenCode {
				t.Fatalf("expected opencode=%v, got %v", tt.wantOpenCode, isOpenCode)
			}
		})
	}
}

func TestCopilotTransformer_AddsToolsAndNormalizesTypes(t *testing.T) {
	transformer := &CopilotTransformer{}
	servers := map[string]interface{}{
		"command": map[string]interface{}{
			"command": "npx",
		},
		"network-stdio": map[string]interface{}{
			"type": "stdio",
			"url":  "http://example.test",
		},
		"network-stream": map[string]interface{}{
			"type": "streamable-http",
			"url":  "http://example.test",
			"tools": []interface{}{
				"kept",
			},
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for name, serverRaw := range servers {
		server := serverRaw.(map[string]interface{})
		tools, ok := server["tools"]
		if !ok {
			t.Fatalf("server %s missing tools array", name)
		}
		if _, ok := tools.([]interface{}); !ok {
			t.Fatalf("server %s tools should be slice, got %T", name, tools)
		}
	}

	if servers["network-stdio"].(map[string]interface{})["type"] != "local" {
		t.Errorf("expected stdio to be normalized to local, got %v", servers["network-stdio"].(map[string]interface{})["type"])
	}
	if servers["network-stream"].(map[string]interface{})["type"] != "http" {
		t.Errorf("expected streamable-http to be normalized to http, got %v", servers["network-stream"].(map[string]interface{})["type"])
	}
}

func TestClaudeTransformer_NormalizesTypes(t *testing.T) {
	transformer := &ClaudeTransformer{}
	servers := map[string]interface{}{
		"network-stream": map[string]interface{}{
			"type": "streamable-http",
			"url":  "http://example.test",
		},
		"command": map[string]interface{}{
			"command": "npx",
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if servers["network-stream"].(map[string]interface{})["type"] != "http" {
		t.Errorf("expected streamable-http to be normalized to http, got %v", servers["network-stream"].(map[string]interface{})["type"])
	}
}

func TestCopilotTransformer_Validation(t *testing.T) {
	transformer := &CopilotTransformer{}
	t.Run("missing url for http", func(t *testing.T) {
		servers := map[string]interface{}{
			"broken": map[string]interface{}{
				"type": "http",
			},
		}

		err := transformer.Transform(servers)
		if err == nil {
			t.Fatal("expected validation error for missing url")
		}
		if !strings.Contains(err.Error(), "url") {
			t.Fatalf("expected error to mention url, got %v", err)
		}
	})

	t.Run("local without url allowed", func(t *testing.T) {
		servers := map[string]interface{}{
			"local-server": map[string]interface{}{
				"type": "local",
			},
		}
		if err := transformer.Transform(servers); err != nil {
			t.Fatalf("expected local transport without url to pass, got %v", err)
		}
	})
}

func TestCopilotTransformer_NonMapServer(t *testing.T) {
	transformer := &CopilotTransformer{}
	servers := map[string]interface{}{
		"invalid": "oops",
		"valid": map[string]interface{}{
			"type": "http",
			"url":  "https://example.test",
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := servers["valid"].(map[string]interface{})["tools"]; !ok {
		t.Fatalf("valid server missing tools array")
	}
}

func TestIsNetworkServer(t *testing.T) {
	tests := []struct {
		name   string
		server map[string]interface{}
		want   bool
	}{
		{"type only", map[string]interface{}{"type": "http"}, true},
		{"url only", map[string]interface{}{"url": "https://example.test"}, true},
		{"both", map[string]interface{}{"type": "http", "url": "https://example.test"}, true},
		{"command", map[string]interface{}{"command": "npx"}, false},
		{"empty", map[string]interface{}{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNetworkServer(tt.server); got != tt.want {
				t.Fatalf("isNetworkServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodexTransformerGithubToken(t *testing.T) {
	transformer := &CodexTransformer{}
	servers := map[string]interface{}{
		"github": map[string]interface{}{
			"type": "streamable-http",
			"url":  "https://api.example.test",
			"headers": map[string]interface{}{
				"Authorization": "Bearer ghp_example",
			},
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	github := servers["github"].(map[string]interface{})
	if github["bearer_token_env_var"] != "CODEX_GITHUB_PERSONAL_ACCESS_TOKEN" {
		t.Fatalf("expected bearer_token_env_var to be set, got %v", github["bearer_token_env_var"])
	}
	if headers, ok := github["headers"]; ok {
		if len(headers.(map[string]interface{})) != 0 {
			t.Fatalf("expected Authorization header to be removed, got %v", headers)
		}
	}
}

func TestGeminiTransformer_RemovesUnsupportedFields(t *testing.T) {
	transformer := &GeminiTransformer{}
	servers := map[string]interface{}{
		"server1": map[string]interface{}{
			"command":      "npx",
			"args":         []interface{}{"-y", "some-mcp-server"},
			"alwaysAllow":  []interface{}{"tool1", "tool2"},
			"autoApprove":  []interface{}{},
			"disabled":     false,
		},
		"server2": map[string]interface{}{
			"type":    "stdio",
			"command": "uvx",
			"gallery": true,
			"env": map[string]interface{}{
				"API_KEY": "test",
			},
		},
		"server3": map[string]interface{}{
			"command":     "node",
			"kept":        "value",
			"alwaysAllow": []interface{}{"monitor"},
			"disabled":    true,
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify server1 has alwaysAllow, autoApprove and disabled removed but other fields remain
	server1 := servers["server1"].(map[string]interface{})
	if _, exists := server1["alwaysAllow"]; exists {
		t.Error("alwaysAllow should be removed from server1")
	}
	if _, exists := server1["autoApprove"]; exists {
		t.Error("autoApprove should be removed from server1")
	}
	if _, exists := server1["disabled"]; exists {
		t.Error("disabled should be removed from server1")
	}
	if server1["command"] != "npx" {
		t.Error("command should be preserved in server1")
	}
	if server1["args"] == nil {
		t.Error("args should be preserved in server1")
	}

	// Verify server2 has type and gallery removed but other fields remain
	server2 := servers["server2"].(map[string]interface{})
	if _, exists := server2["type"]; exists {
		t.Error("type should be removed from server2")
	}
	if _, exists := server2["gallery"]; exists {
		t.Error("gallery should be removed from server2")
	}
	if server2["command"] != "uvx" {
		t.Error("command should be preserved in server2")
	}
	if server2["env"] == nil {
		t.Error("env should be preserved in server2")
	}

	// Verify server3 has alwaysAllow and disabled removed but kept field remains
	server3 := servers["server3"].(map[string]interface{})
	if _, exists := server3["alwaysAllow"]; exists {
		t.Error("alwaysAllow should be removed from server3")
	}
	if _, exists := server3["disabled"]; exists {
		t.Error("disabled should be removed from server3")
	}
	if server3["kept"] != "value" {
		t.Error("kept field should be preserved in server3")
	}
	if server3["command"] != "node" {
		t.Error("command should be preserved in server3")
	}
}

func TestGeminiTransformer_NonMapServer(t *testing.T) {
	transformer := &GeminiTransformer{}
	servers := map[string]interface{}{
		"invalid": "not-a-map",
		"valid": map[string]interface{}{
			"command":     "npx",
			"autoApprove": []interface{}{},
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Valid server should have autoApprove removed
	validServer := servers["valid"].(map[string]interface{})
	if _, exists := validServer["autoApprove"]; exists {
		t.Error("autoApprove should be removed from valid server")
	}

	// Invalid server should remain unchanged (string)
	if servers["invalid"] != "not-a-map" {
		t.Error("non-map server should remain unchanged")
	}
}

func TestOpenCodeTransformer_CommandArrayConversion(t *testing.T) {
	transformer := &OpenCodeTransformer{}
	servers := map[string]interface{}{
		"with-args": map[string]interface{}{
			"command": "npx",
			"args":    []interface{}{"-y", "some-mcp-server"},
		},
		"without-args": map[string]interface{}{
			"command": "uvx",
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check server with args - command should be array, args should be removed
	serverWithArgs := servers["with-args"].(map[string]interface{})
	cmdArray, ok := serverWithArgs["command"].([]interface{})
	if !ok {
		t.Fatalf("command should be array, got %T", serverWithArgs["command"])
	}
	if len(cmdArray) != 3 {
		t.Fatalf("expected 3 elements in command array, got %d", len(cmdArray))
	}
	if cmdArray[0] != "npx" || cmdArray[1] != "-y" || cmdArray[2] != "some-mcp-server" {
		t.Errorf("unexpected command array: %v", cmdArray)
	}
	if _, exists := serverWithArgs["args"]; exists {
		t.Error("args should be removed after conversion")
	}

	// Check server without args - command should still be array
	serverWithoutArgs := servers["without-args"].(map[string]interface{})
	cmdArray2, ok := serverWithoutArgs["command"].([]interface{})
	if !ok {
		t.Fatalf("command should be array, got %T", serverWithoutArgs["command"])
	}
	if len(cmdArray2) != 1 || cmdArray2[0] != "uvx" {
		t.Errorf("unexpected command array: %v", cmdArray2)
	}
}

func TestOpenCodeTransformer_EnvironmentRename(t *testing.T) {
	transformer := &OpenCodeTransformer{}
	servers := map[string]interface{}{
		"server": map[string]interface{}{
			"command": "npx",
			"env": map[string]interface{}{
				"API_KEY": "test-key",
				"DEBUG":   "true",
			},
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	server := servers["server"].(map[string]interface{})
	if _, exists := server["env"]; exists {
		t.Error("env should be removed")
	}

	environment, exists := server["environment"]
	if !exists {
		t.Fatal("environment should exist")
	}

	envMap, ok := environment.(map[string]interface{})
	if !ok {
		t.Fatalf("environment should be a map, got %T", environment)
	}
	if envMap["API_KEY"] != "test-key" {
		t.Error("API_KEY should be preserved in environment")
	}
	if envMap["DEBUG"] != "true" {
		t.Error("DEBUG should be preserved in environment")
	}
}

func TestOpenCodeTransformer_TypeNormalization(t *testing.T) {
	transformer := &OpenCodeTransformer{}
	servers := map[string]interface{}{
		"stdio-server": map[string]interface{}{
			"type":    "stdio",
			"command": "node",
		},
		"http-server": map[string]interface{}{
			"type": "http",
			"url":  "https://example.com",
		},
		"streamable-http-server": map[string]interface{}{
			"type": "streamable-http",
			"url":  "https://example.com",
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if servers["stdio-server"].(map[string]interface{})["type"] != "local" {
		t.Error("stdio should be converted to local")
	}
	if servers["http-server"].(map[string]interface{})["type"] != "remote" {
		t.Error("http should be converted to remote")
	}
	if servers["streamable-http-server"].(map[string]interface{})["type"] != "remote" {
		t.Error("streamable-http should be converted to remote")
	}
}

func TestOpenCodeTransformer_RemovesUnsupportedFields(t *testing.T) {
	transformer := &OpenCodeTransformer{}
	servers := map[string]interface{}{
		"server": map[string]interface{}{
			"command":     "npx",
			"alwaysAllow": []interface{}{"tool1", "tool2"},
			"autoApprove": []interface{}{},
			"disabled":    false,
			"gallery":     true,
			"tools":       []interface{}{"tool1"},
			"env": map[string]interface{}{
				"KEY": "value",
			},
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	server := servers["server"].(map[string]interface{})

	if _, exists := server["alwaysAllow"]; exists {
		t.Error("alwaysAllow should be removed")
	}
	if _, exists := server["autoApprove"]; exists {
		t.Error("autoApprove should be removed")
	}
	if _, exists := server["disabled"]; exists {
		t.Error("disabled should be removed")
	}
	if _, exists := server["gallery"]; exists {
		t.Error("gallery should be removed")
	}
	if _, exists := server["tools"]; exists {
		t.Error("tools should be removed")
	}
	
	// env should be renamed to environment, not removed
	if _, exists := server["environment"]; !exists {
		t.Error("environment should exist (renamed from env)")
	}
}

func TestOpenCodeTransformer_NonMapServer(t *testing.T) {
	transformer := &OpenCodeTransformer{}
	servers := map[string]interface{}{
		"invalid": "not-a-map",
		"valid": map[string]interface{}{
			"command": "npx",
			"args":    []interface{}{"-y", "tool"},
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Valid server should have command converted to array
	validServer := servers["valid"].(map[string]interface{})
	if _, ok := validServer["command"].([]interface{}); !ok {
		t.Error("command should be converted to array")
	}

	// Invalid server should remain unchanged (string)
	if servers["invalid"] != "not-a-map" {
		t.Error("non-map server should remain unchanged")
	}
}

func TestOpenCodeTransformer_AddsTypeWhenMissing(t *testing.T) {
	transformer := &OpenCodeTransformer{}
	servers := map[string]interface{}{
		"command-based": map[string]interface{}{
			"command": "npx",
			"args":    []interface{}{"-y", "@upstash/context7-mcp@latest"},
		},
		"url-based": map[string]interface{}{
			"url": "https://api.example.com/mcp",
			"headers": map[string]interface{}{
				"Authorization": "Bearer token",
			},
		},
		"docker-command": map[string]interface{}{
			"command": "docker",
			"args": []interface{}{
				"run",
				"-i",
				"--rm",
				"--init",
				"--pull=always",
				"mcr.microsoft.com/playwright/mcp",
			},
		},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Command-based server should have type "local" added
	commandServer := servers["command-based"].(map[string]interface{})
	if commandServer["type"] != "local" {
		t.Errorf("command-based server should have type 'local', got %v", commandServer["type"])
	}

	// URL-based server should have type "remote" added
	urlServer := servers["url-based"].(map[string]interface{})
	if urlServer["type"] != "remote" {
		t.Errorf("url-based server should have type 'remote', got %v", urlServer["type"])
	}

	// Docker command server should have type "local" added
	dockerServer := servers["docker-command"].(map[string]interface{})
	if dockerServer["type"] != "local" {
		t.Errorf("docker-command server should have type 'local', got %v", dockerServer["type"])
	}
}

func TestOpenCodeTransformer_AddsTypeForEdgeCases(t *testing.T) {
	transformer := &OpenCodeTransformer{}
	servers := map[string]interface{}{
		"no-command-or-url": map[string]interface{}{
			"someOtherField": "value",
		},
		"empty-server": map[string]interface{}{},
	}

	if err := transformer.Transform(servers); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Server with neither command nor url should default to "local"
	edgeCaseServer := servers["no-command-or-url"].(map[string]interface{})
	if edgeCaseServer["type"] != "local" {
		t.Errorf("server with no command or url should default to type 'local', got %v", edgeCaseServer["type"])
	}

	// Empty server should also default to "local"
	emptyServer := servers["empty-server"].(map[string]interface{})
	if emptyServer["type"] != "local" {
		t.Errorf("empty server should default to type 'local', got %v", emptyServer["type"])
	}
}
