# Server Syncer

![Server Syncer](icon-resized.png)

server-syncer is a Go-based utility that keeps MCP configuration files aligned across coding agents such as Copilot, Codex, Claude Code, Gemini, and others. Give it a single template file, and it will convert that configuration into the formats required by each tool while treating one format as the source of truth.

## Repository layout

- `go.mod` pins the project to Go 1.25.4.
- `cmd/server-syncer` contains the CLI entrypoint that reads a template file, chooses a source-of-truth agent, and prints the converted configs for the supported agents.
- `internal/syncer` implements the conversion logic, template loader, and accompanying unit tests.

## Getting started

1. Install Go 1.22 or newer.
2. Run the CLI with a template file and source agent, for example:

   ```bash
   go run ./cmd/server-syncer -template ./configs/codex.json -source codex
   ```

3. The tool will echo the converted configurations for each agent so you can copy them into the appropriate files.

## Testing

Run:

```bash
go test ./...
```

The unit tests cover template loading and the syncer’s validation/conversion logic.

## CI and releases

- **Tests** – `go test ./...` runs on every push and pull request so the core package stays green.
- **Commit message format** – a workflow enforces Conventional Commit-style messages so releases can be calculated automatically.
- **Release** – a manual workflow dispatch runs Go tests and then semantic-release to bump the recorded semantic version and publish the tag/release; the job still infers the increment from Conventional Commits.
