# Server Syncer

![Server Syncer](icon-resized.png)

server-syncer is a Go-based utility that keeps MCP configuration files aligned
across coding agents such as Copilot, Codex, Claude Code, Gemini, and others.
Give it a single template file, and it will convert that configuration into the
formats required by each tool while treating one format as the source of truth.

## Repository layout

- `go.mod` pins the project to Go 1.25.4.
- `cmd/server-syncer` contains the CLI entrypoint that reads a template file, chooses
  a source-of-truth agent, and prints the converted configs for the supported
  agents.
- `internal/syncer` implements the conversion logic, template loader, and
  accompanying unit tests.

## Getting started

1. Download the latest binary from the [releases page](https://github.com/timbuchinger/server-syncer/releases/latest).
2. Create a config; for example, save this to `server-syncer.yml` next to the binary:

   ```yaml
   source: codex
   targets:
     - copilot
     - claudeCode
     - gemini
   template: configs/codex.json
   ```

3. Run the app with your config file:

   ```bash
   ./server-syncer -config server-syncer.yml
   ```

## Development commands

### Build

```bash
go build ./cmd/server-syncer
```

This compiles the CLI into the current directory so you can run it repeatedly
without `go run`.

### Run

```bash
go run ./cmd/server-syncer -config server-syncer.yml
```

Use `-source` or `-agents` if you need to override values in the config for a
single run; the template path is always read from the config file.

### Test

```bash
go test ./...
```

The tests cover template loading along with the syncer's validation and
conversion logic.

## Documentation linting

When editing markdown, run the lint fixer to download the tool and apply all
reported fixes:

```bash
npx markdownlint-cli2 --fix '**/*.md'
```

## Configuration file

`server-syncer` looks for a YAML configuration at one of the platform-specific locations:

- Linux: `/etc/server-syncer.yml`
- macOS: `/usr/local/etc/server-syncer.yml`
- Windows: `C:\ProgramData\server-syncer\config.yml`

You can override this path with `-config <path>`. The file should describe the
`source` agent, the list of `targets`, and the `template` path; see
`CONFIGURATION.md` for the schema and a sample layout. Config values are used
unless you explicitly set `-source` or `-agents`, and the template path always
comes from the config.

The tool will echo the converted configurations for each agent so you can copy
them into the appropriate files.

## CI and releases

- **Tests** – `go test ./...` runs on every push and pull request so the core
  package stays green.
- **Commit message format** – a workflow enforces Conventional Commit-style
  messages so releases can be calculated automatically.
- **Release** – a manual workflow dispatch runs Go tests and then semantic-release
  to bump the recorded semantic version and publish the tag/release; the job
  still infers the increment from Conventional Commits.
