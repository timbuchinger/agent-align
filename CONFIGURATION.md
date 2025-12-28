# Configuration format

`agent-align` reads two YAML files:

1. **MCP definitions** – the source of truth for your servers (default
   `agent-align-mcp.yml` next to the target config). This file is required.
2. **Target config** – describes which agents to update, optional path overrides,
   and any extra copy tasks (default `/etc/agent-align.yml`).

## MCP definitions file (agent-align-mcp.yml)

The MCP file lists every server in a neutral JSON-style shape:

```yaml
servers:
  github:
    type: streamable-http
    url: https://api.example.com/mcp/
    headers:
      Authorization: "Bearer ${GITHUB_TOKEN}"
    tools: []
  claude-cli:
    command: npx
    args:
      - '@example/mcp-server@latest'
    env:
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
  prompts:
    command: ./scripts/run-prompts.sh
    args:
      - --watch
```

You can also use the legacy `mcpServers` key instead of `servers`. Each server
entry is a mapping; the keys match the fields you would normally place in the
agent-specific files (for example, `command`, `args`, `env`, `headers`,
`alwaysAllow`, `autoApprove`, `disabled`, `tools`, `type`, and `url`).

### Environment variable expansion

All string values in the MCP definitions file support environment variable
expansion using `${VAR}` or `$VAR` syntax. This allows you to securely
reference secrets and configuration from your environment instead of
hardcoding them in the YAML file.

Default values are supported with the `${VAR:-default}` syntax. If the
environment variable is not set or is empty, the default value will be used.

**Examples:**

```yaml
servers:
  secure-api:
    type: streamable-http
    url: https://api.example.com
    headers:
      # Reference environment variable directly
      Authorization: "Bearer ${API_TOKEN}"
  
  database-server:
    command: python
    args:
      - -m
      - db_server
      # Use default value if environment variable is not set
      - --host=${DB_HOST:-localhost}
      - --port=${DB_PORT:-5432}
    env:
      # Works in env blocks too
      DB_PASSWORD: ${DB_PASSWORD}
      LOG_LEVEL: ${LOG_LEVEL:-info}
```

Environment variables are expanded recursively in all string values
throughout the configuration, including headers, URLs, command arguments,
and environment variable definitions.

## Target config file (agent-align.yml)

The target config points to the MCP file (optional if you accept the default
path) and lists the destinations to update:

```yaml
mcpServers:
  configPath: agent-align-mcp.yml
  targets:
    agents:
      - name: copilot
      - name: vscode
      - name: codex
        path: /custom/.codex/config.toml  # optional override
      - claudecode
      - gemini
      - kilocode
    additionalTargets:
      json:
        - filePath: /path/to/additional_targets.json
          jsonPath: .mcpServers
      jsonc:
        - filePath: /path/to/additional_targets.jsonc
          jsonPath: .mcpServers
extraTargets:
  files:
    - source: /path/to/AGENTS.md
      destinations:
        - /path/to/other/AGENTS.md
        - path: /path/to/enhanced/AGENTS.md
          appendSkills:
            - path: /path/to/skills/
              ignoredSkills:
                - test1
                - test2
  directories:
    - source: /path/to/prompts
      destinations:
        - path: /path/to/another/prompts
          excludeGlobs:
            - 'troubleshoot/**'
          flatten: true
```

### Fields

- `mcpServers` (mapping, required) – nests MCP sync settings.
  - `configPath` (string, optional) – path to the MCP definitions file. Defaults
    to `agent-align-mcp.yml` next to the target config when omitted.
  - `targets` (mapping, required) – agents to write plus optional extras.
    - `agents` (sequence, required) – list of agent names or objects with `name`
      and optional `path` override for the destination file. Repeat an agent
      with different `path` values to write the same format to multiple
      destinations. Exact duplicate `name + path` combinations and blank entries
      are ignored.
    - `additionalTargets.json` (sequence, optional) – mirror the MCP payload
      into other JSON files. Each entry must specify `filePath` and may set
      `jsonPath` (dot-separated) where the servers should be placed; omit
      `jsonPath` to replace the entire file.
    - `additionalTargets.jsonc` (sequence, optional) – mirror the MCP payload
      into other JSONC (JSON with Comments) files. Each entry must specify
      `filePath` and may set `jsonPath` (dot-separated) where the servers
      should be placed; omit `jsonPath` to replace the entire file. Comments
      in the original file will be stripped when writing the updated content.
- `extraTargets` (mapping, optional) – copies additional content alongside the
  MCP sync.
  - `files` (sequence) – mirror a single source file to multiple destinations.
    Each entry must specify `source` and at least one `destinations` value.
    Each destination may be provided as a plain string (the destination path)
    or as a mapping with additional options:
    - `path` (string, required) – destination file path.
    - `frontmatterPath` (string, optional) – path to a file whose contents
      will be written as frontmatter/template block to the destination before
      any skills content is appended. Useful for copying prompt files that
      need YAML frontmatter or a fixed header.
    - `appendSkills` (sequence, optional) – append skills from one or more
      directories. Each entry specifies:
      - `path` (string, required) – directory containing SKILL.md files.
      - `ignoredSkills` (sequence, optional) – skill names to exclude.
    - `pathToSkills` (string, optional) – deprecated, use `appendSkills`
      instead. Single directory path for appending skills.
    - `appendToFilename` (string, optional) – string to insert before the
      destination filename's extension. Example: with `.prompt`, `plan.md` ->
      `plan.prompt.md`. If the source file has no extension the value is
      appended to the end of the filename.
  - `directories` (sequence) – copy every file within `source` to each entry in
    `destinations`. Every destination entry must specify a `path` and may set
    `excludeGlobs` (sequence of glob patterns) to skip matching files, and/or
    `flatten: true` to drop the source directory structure while copying.
    Glob patterns support `**` for recursive matching (e.g., `dir/**` excludes
    all files under `dir/`, `*.log` excludes all log files).
    - `appendToFilename` (string, optional) – insert the given string before
      each copied file's extension. Useful for adding agent-specific suffixes
      like `.prompt` so `plan.md` becomes `plan.prompt.md`.

## Supported Agents and defaults

Agent | Config File | Format | Root
----- | ----------- | ------ | ----
copilot | `~/.copilot/mcp-config.json` | JSON | `mcpServers`
vscode | `~/.config/Code/User/mcp.json` | JSON | `servers`
codex | `~/.codex/config.toml` | TOML | `mcp_servers`
claudecode | `~/.claude.json` | JSON | `mcpServers`
gemini | `~/.gemini/settings.json` | JSON | `mcpServers`
kilocode | Platform-dependent (see note below) | JSON | `mcpServers`

Every agent accepts a `path` override in `targets.agents` if your installation
lives elsewhere.

Note: Kilocode config paths

- Windows: `~/AppData/Roaming/Code/user/mcp.json`
- Linux: `~/.config/Code/User/globalStorage/kilocode.kilo-code/settings/mcp_settings.json`

## CLI flags and init command

- `-config` – Path to the target config. Defaults to the platform-specific
  location listed above.
- `-mcp-config` – Path to the MCP definitions file. Defaults to
  `agent-align-mcp.yml` next to the selected config.
- `-agents` – Override the target agents defined in the config. Overrides still
  honor per-agent `path` entries if they exist in the file.
- `-dry-run` – Preview changes without writing.
- `-confirm` – Skip the confirmation prompt when applying writes.

Run `agent-align init -config ./agent-align.yml` to generate a starter config via
prompts if you prefer not to edit YAML manually. The wizard collects the agent
list plus optional additional JSON destinations and writes the final file for you.
