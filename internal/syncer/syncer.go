package syncer

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

type Template struct {
    Name    string
    Payload string
}

func LoadTemplateFromFile(path string) (Template, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return Template{}, err
    }

    payload := strings.TrimSpace(string(data))
    return Template{
        Name:    filepath.Base(path),
        Payload: payload,
    }, nil
}

type Syncer struct {
    SourceAgent string
    Agents      []string
}

func New(sourceAgent string, agents []string) *Syncer {
    normalized := normalizeAgent(sourceAgent)
    cleanAgents := uniqueAgents(agents)
    return &Syncer{
        SourceAgent: normalized,
        Agents:      cleanAgents,
    }
}

func (s *Syncer) Sync(template Template) (map[string]string, error) {
    if strings.TrimSpace(template.Name) == "" {
        return nil, fmt.Errorf("template requires a name")
    }
    if strings.TrimSpace(template.Payload) == "" {
        return nil, fmt.Errorf("template payload cannot be empty")
    }
    if len(s.Agents) == 0 {
        return nil, fmt.Errorf("no agents configured to sync")
    }
    if !containsAgent(s.Agents, s.SourceAgent) {
        return nil, fmt.Errorf("source agent %q not found in agent list", s.SourceAgent)
    }

    result := make(map[string]string, len(s.Agents))
    for _, agent := range s.Agents {
        result[agent] = formatConfig(agent, s.SourceAgent, template)
    }

    return result, nil
}

func formatConfig(agent, source string, template Template) string {
    return fmt.Sprintf("agent=%s;source=%s;name=%s;payload=%s", agent, source, template.Name, template.Payload)
}

func normalizeAgent(agent string) string {
    return strings.ToLower(strings.TrimSpace(agent))
}

func uniqueAgents(agents []string) []string {
    seen := make(map[string]struct{}, len(agents))
    var out []string
    for _, agent := range agents {
        normalized := normalizeAgent(agent)
        if normalized == "" {
            continue
        }
        if _, exists := seen[normalized]; exists {
            continue
        }
        seen[normalized] = struct{}{}
        out = append(out, normalized)
    }
    return out
}

func containsAgent(agents []string, target string) bool {
    for _, agent := range agents {
        if agent == target {
            return true
        }
    }
    return false
}
