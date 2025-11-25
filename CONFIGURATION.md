# Configuration format

`server-syncer` reads a single YAML configuration that tells it which agent
format is the source of truth and which agents should be generated from it.

```yaml
source: codex
targets:
  - gemini
  - copilot
  - claudecode
template: configs/codex.json
```

## Fields

- `source` (string, required) – the agent whose configuration acts as the
  authoritative template. Allowed values are `codex`, `gemini`, `copilot`, and
  `claudecode`.
- `targets` (sequence, required, non-empty) – a list of one or more agents to
  update from the source. Each target value must also be one of `codex`,
  `gemini`, `copilot`, or `claudecode`. Targets **cannot** include the same
  value as `source`.
- `template` (string, required) – the path to the file representing your source
  agent’s configuration. Any readable path is acceptable (absolute or relative),
  and the CLI reads this file each time it runs.

Only these four agent names are supported by the current implementation.
