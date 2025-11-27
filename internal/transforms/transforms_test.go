package transforms

import (
	"strings"
	"testing"
)

func TestGetTransformer(t *testing.T) {
	tests := []struct {
		name        string
		agent       string
		wantCopilot bool
		wantCodex   bool
	}{
		{"copilot", "copilot", true, false},
		{"copilot spaced", " copilot ", true, false},
		{"codex", "codex", false, true},
		{"codex spaced", " codex ", false, true},
		{"default", "vscode", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTransformer(tt.agent)
			_, isCopilot := got.(*CopilotTransformer)
			_, isCodex := got.(*CodexTransformer)

			if isCopilot != tt.wantCopilot {
				t.Fatalf("expected copilot=%v, got %v", tt.wantCopilot, isCopilot)
			}
			if isCodex != tt.wantCodex {
				t.Fatalf("expected codex=%v, got %v", tt.wantCodex, isCodex)
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
