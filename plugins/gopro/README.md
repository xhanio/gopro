# gopro — Claude Code plugin

Packages the `gopro` skill so Claude Code can drive
[GoPro](https://github.com/xhanio/gopro): writing and editing `project.yaml`,
building binaries and Docker images across environments, and generating config,
Kubernetes, and Docker Compose templates with environment overlays.

Documentation only — no MCP server, no binary. The `gopro` command itself is a
separate install:

```bash
go install github.com/xhanio/gopro@latest
```

## Install

```
/plugin marketplace add https://github.com/xhanio/plugins
/plugin install gopro@xhanio
```

The skill activates on its own whenever a session touches `project.yaml`,
multi-environment builds, or Go project generation. You can also invoke it by
name with `/gopro`.

## What's inside

One skill, `gopro`, with a reference document it pulls in on demand:

- **SKILL.md** — the entry point: command quick reference, `project.yaml`
  structure, the template system, the standard workflows, directory layout,
  and troubleshooting. Also carries the rules that are easy to get wrong —
  the two merge layers that behave oppositely (environment overrides replace
  arrays; per-binary and per-platform settings merge key-wise), cross-compiling
  with per-target env via `platforms`, publishing `:latest` alongside a version,
  `config.yaml` and `secret.env` rather than `.env`, `secret.env` never
  reaching output or git, every project needing at least one environment
  directory, and `gopro build binary` rather than a bare `go build`.
- **references/REFERENCE.md** — the complete `project.yaml` field reference
  including the binary and platform specs, Docker build arguments, image
  tagging and push, build metadata injection, both environment merging layers,
  and the template rendering pipeline.

Only SKILL.md is read up front; the reference loads when the task calls for it.

## Source

This plugin is developed in the tool repo at
[`plugins/gopro`](https://github.com/xhanio/gopro/tree/main/plugins/gopro) and
mirrored into the [xhanio/plugins](https://github.com/xhanio/plugins)
marketplace. File issues against the gopro repo.
