# AGENTS

This repository powers **server-syncer**, a Go utility for synchronizing MCP (model configuration profile) configs across coding agents such as Copilot, Codex, Claude Code, Gemini, and others. It is designed to keep every agent's configuration in lockstep so you can iterate on a single template and automatically propagate changes to the rest of the toolchain.

You provide one file as the template, and server-syncer converts it into the formats required by the other agents. One of the agent-specific outputs is chosen as the source of truth, and the tool uses that to update the remaining files so all the agents stay in sync.

All commits must follow Conventional Commits so the release workflow can determine the next semantic version automatically.
