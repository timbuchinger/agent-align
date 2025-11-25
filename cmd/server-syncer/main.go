package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "strings"

    "server-syncer/internal/syncer"
)

func main() {
    templatePath := flag.String("template", "", "path to the template file")
    sourceAgent := flag.String("source", "", "source-of-truth agent name")
    agents := flag.String("agents", "Copilot,Codex,ClaudeCode,Gemini", "comma-separated list of agents to keep in sync")
    flag.Parse()

    if *templatePath == "" || *sourceAgent == "" {
        flag.Usage()
        os.Exit(1)
    }

    tpl, err := syncer.LoadTemplateFromFile(*templatePath)
    if err != nil {
        log.Fatalf("failed to load template: %v", err)
    }

    candidateAgents := parseAgents(*agents)
    s := syncer.New(*sourceAgent, candidateAgents)

    converted, err := s.Sync(tpl)
    if err != nil {
        log.Fatalf("sync failed: %v", err)
    }

    fmt.Println("Converted configurations:")
    for agent, cfg := range converted {
        fmt.Printf("  %s -> %s\n", agent, cfg)
    }
}

func parseAgents(agents string) []string {
    segments := strings.Split(agents, ",")
    var out []string
    for _, segment := range segments {
        trimmed := strings.TrimSpace(segment)
        if trimmed == "" {
            continue
        }
        out = append(out, trimmed)
    }
    return out
}
