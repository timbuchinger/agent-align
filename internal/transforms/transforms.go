package transforms

import (
	"fmt"
	"strings"
)

// Transformer defines the interface for destination-specific transformations.
// Each target agent can have its own transformer that manipulates server
// configurations before they are written.
type Transformer interface {
	// Transform modifies servers in place and returns an error if validation fails.
	Transform(servers map[string]interface{}) error
}

// GetTransformer returns the appropriate transformer for a given agent.
// If no specific transformer exists, it returns a no-op transformer.
func GetTransformer(agent string) Transformer {
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case "copilot":
		return &CopilotTransformer{}
	case "codex":
		return &CodexTransformer{}
	default:
		return &NoOpTransformer{}
	}
}

// NoOpTransformer performs no transformations.
type NoOpTransformer struct{}

// Transform returns nil without modifying servers.
func (t *NoOpTransformer) Transform(servers map[string]interface{}) error {
	return nil
}

// CopilotTransformer handles Copilot-specific transformations and validations.
type CopilotTransformer struct{}

// Transform applies Copilot-specific modifications:
// - Adds an empty "tools" array to every server if not present
// - Normalizes network transport types to the values Copilot expects
// - Validates that network-based servers have both "type" and "url" fields
func (t *CopilotTransformer) Transform(servers map[string]interface{}) error {
	for name, serverRaw := range servers {
		server, ok := serverRaw.(map[string]interface{})
		if !ok {
			continue
		}

		if err := t.transformServer(name, server); err != nil {
			return err
		}
	}
	return nil
}

// transformServer applies transformations to a single server configuration.
func (t *CopilotTransformer) transformServer(name string, server map[string]interface{}) error {
	addToolsArrayIfMissing(server)

	if typ, ok := server["type"].(string); ok {
		switch strings.ToLower(strings.TrimSpace(typ)) {
		case "stdio":
			server["type"] = "local"
		case "streamable-http":
			server["type"] = "http"
		}
	}

	if isNetworkServer(server) {
		if err := validateNetworkServer(name, server); err != nil {
			return err
		}
	}

	return nil
}

// isNetworkServer returns true if the server appears to be a network-based server.
// A network-based server has either "type" or "url" field (or both).
func isNetworkServer(server map[string]interface{}) bool {
	_, hasType := server["type"]
	_, hasURL := server["url"]
	return hasType || hasURL
}

// addToolsArrayIfMissing adds an empty "tools" array to the server if not present.
func addToolsArrayIfMissing(server map[string]interface{}) {
	if _, hasTools := server["tools"]; !hasTools {
		server["tools"] = []interface{}{}
	}
}

// validateNetworkServer ensures that network-based servers have both "type" and "url" fields.
func validateNetworkServer(name string, server map[string]interface{}) error {
	rawType, hasType := server["type"]
	_, hasURL := server["url"]

	if !hasType && !hasURL {
		// Not a network server, nothing to validate
		return nil
	}

	if hasType {
		if t, ok := rawType.(string); ok {
			if strings.EqualFold(strings.TrimSpace(t), "local") {
				// Copilot local transports do not require a URL.
				return nil
			}
		}
	}

	var missing []string
	if !hasType {
		missing = append(missing, "type")
	}
	if !hasURL {
		missing = append(missing, "url")
	}

	if len(missing) > 0 {
		return fmt.Errorf("copilot validation error: network-based server %q is missing required field(s): %s. Network servers must have both 'type' and 'url' fields",
			name, strings.Join(missing, ", "))
	}

	return nil
}

// CodexTransformer applies Codex-specific conversions.
type CodexTransformer struct{}

// Transform converts GitHub Authorization headers into the env var token expected by Codex.
func (t *CodexTransformer) Transform(servers map[string]interface{}) error {
	githubRaw, ok := servers["github"]
	if !ok {
		return nil
	}

	server, ok := githubRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	headers, hasHeaders := server["headers"].(map[string]interface{})
	if !hasHeaders {
		return nil
	}

	if _, hasEnv := server["bearer_token_env_var"]; hasEnv {
		delete(headers, "Authorization")
		if len(headers) == 0 {
			delete(server, "headers")
		}
		return nil
	}

	if _, hasAuth := headers["Authorization"]; !hasAuth {
		return nil
	}

	server["bearer_token_env_var"] = "CODEX_GITHUB_PERSONAL_ACCESS_TOKEN"
	delete(headers, "Authorization")
	if len(headers) == 0 {
		delete(server, "headers")
	}
	return nil
}
