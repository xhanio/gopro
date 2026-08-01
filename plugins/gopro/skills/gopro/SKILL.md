---
name: gopro
description: Use when a project has a project.yaml or uses the gopro command; when building Go binaries or Docker images across environments; when cross-compiling for multiple platforms or setting per-target build env; when generating config, Kubernetes, or Docker Compose templates; when scaffolding a Go project structure; or when handling secret.env and config.yaml in env/ directories.
compatibility: Requires Go, git, and optionally Docker. Install with go install github.com/xhanio/gopro@latest
metadata:
  author: xhanio
  version: "1.1"
---

# GoPro - Go Project Generator and Build Tool

GoPro manages multi-environment Go projects through a single `project.yaml` configuration file. It builds binaries, Docker images, and generates configuration/Kubernetes/Docker Compose templates with environment-specific overlays.

## Quick Reference

| Task | Command |
|------|---------|
| Generate example config | `gopro example` |
| Initialize project | `gopro init` |
| Build binaries | `gopro build binary -e <env>` |
| Build Docker images | `gopro build image -e <env> --push` |
| Push versioned tag + `:latest` | `gopro build image -e <env> --push --latest` |
| Generate configs | `gopro generate config -e <env>` |
| Generate K8s manifests | `gopro generate kubernetes -e <env>` |
| Generate docker-compose | `gopro generate docker-compose -e <env>` |
| Show version info | `gopro version` |

### Global Flags

- `-c, --config <path>` - Config file (default: `project.yaml`)
- `-e, --environment <name>` - Target environment (local, prod, or custom). Omitted = use `default` as-is
- `-f, --filter <regex>` - Regex filter for selective component building (default: `.*`)
- `-v, --verbose` - Debug output

### Per-Command Flags

- `gopro build binary`: `-o/--output`, `--product-model`, `--product-version`, `--build-version`, `--build-type`, `--build-date`
- `gopro build image`: `-p/--push`, `-l/--latest` (also tag and push `:latest`; requires `--push`)
- `gopro generate`: `-x/--prefix` (template prefix, default `template.`) on all three subcommands
- `gopro generate config`: `-o/--output` — `gopro generate kubernetes`: `-t/--output`
- `gopro generate docker-compose`: no output flag; writes to `docker_compose_tgt`

## Configuration Structure

The `project.yaml` file has four main sections:

```yaml
# Top-level metadata
product: myapp              # Required
model: standard             # Optional; parsed but inert — see note below
version: v1.0.0             # Optional
domain: example.com         # Optional
module: github.com/user/app # Auto-detected from go.mod

# Base settings shared across environments
default:
  binary_src: build/binary
  binary_tgt: bin/
  binary_build_env: [CGO_ENABLED=0]
  binary_build_args: [-v, -ldflags, '-s -w']
  binaries: [api, worker]
  image_build_src: build/image
  image_prefix: registry.io/myapp
  image_tag: latest
  images: [api, worker]
  config_src: env/default/config
  config_tgt: dist/config
  configs: [api, worker]
  kubernetes_src: env/default/kubernetes
  kubernetes_tgt: dist/kubernetes
  kubernetes_templates: [api]
  docker_compose_src: env/default/docker-compose
  docker_compose_tgt: dist

# Environment-specific overrides.
# Scalars override individually; ARRAYS ARE REPLACED WHOLESALE (see below).
env:
  local:
    binary_build_env: [CGO_ENABLED=1]
    config_src: env/local/config
    config_tgt: dist/local/config
  prod:
    binary_build_env: [CGO_ENABLED=0, GOOS=linux, GOARCH=amd64]
    image_prefix: prod-registry.io/myapp
    image_tag: v1.0.0

# Build and generate specifications
build:
  binaries:
    - name: api
      src: cmd/api                  # Optional custom source path
      config_dir: /etc/api          # For template functions
      build_env: [CGO_ENABLED=0]    # Optional: MERGED over binary_build_env
      build_args: [-v]              # Optional: REPLACES binary_build_args
      platforms:                    # Optional cross-compile targets
        - name: linux/amd64
        - name: linux/arm64
          env: [CC=aarch64-linux-gnu-gcc]  # MERGED over build_env, this target only
          args: [-v, -tags=netgo]          # REPLACES build_args, this target only
  images:
    - name: db
      build_from: postgres:13       # Pull and tag existing image
    - name: api
      base: ubuntu:22.04            # Build from Dockerfile
      # base: $db                   # Or cross-reference another image

generate:
  configs:
    - name: api
      files: ["*.yaml", "*.json", "cert/*"]
      # NOTE: Do NOT include secret.env in files — see below
  kubernetes:
    - name: api
      files: ["deployment.yaml", "service.yaml"]
  docker_compose:
    files: ["docker-compose.yaml"]
```

## Important: `model` Is Parsed but Inert

The top-level `model:` field loads into the config and is then never read.
It does **not** reach `info.ProductModel`; only the `--product-model` flag
sets that. A project relying on `model:` for build metadata gets an empty
value at runtime with no warning.

```bash
# model: rackmount in project.yaml  ->  info.ProductModel=
gopro build binary --product-model rackmount   # the only way to set it
```

## Important: Two Merge Layers That Behave Oppositely

Build settings pass through two independent layers. Mixing them up is the most
common source of silently wrong builds.

**Layer 1 — `default` → `env.<name>` (inside project.yaml): arrays are REPLACED.**

Scalars (`binary_src`, `image_tag`, …) override individually and uninvolved keys
are inherited. But any array an environment sets replaces its `default`
counterpart entirely:

```yaml
default:
  binary_build_env: [GOOS=linux, GOARCH=amd64, CGO_ENABLED=0]
env:
  local:
    binary_build_env: [CGO_ENABLED=1]   # GOOS and GOARCH are now GONE
```

`gopro build binary -e local` here builds with **only** `CGO_ENABLED=1`. Anything
still needed must be restated in full:

```yaml
env:
  local:
    binary_build_env: [GOOS=linux, GOARCH=amd64, CGO_ENABLED=1]
```

This is easy to miss because it is invisible on a host that already matches the
dropped `GOOS`/`GOARCH`.

**Layer 2 — `binary_build_env` → binary `build_env` → platform `env` (at build time): MERGED key-wise.**

Each level overrides only the variables it names and inherits the rest. The
parallel `binary_build_args` → `build_args` → `args` chain **replaces** instead,
because Go build flags are positional and repeatable, so a key-wise merge could
not tell an override from an accumulation. Only an unset level inherits — an
explicit `[]` therefore means "build with no arguments at all".

| Setting | `default` → `env` | env → binary → platform |
|---------|-------------------|--------------------------|
| `*_build_env` / `build_env` / `env` | replaced | **merged key-wise** |
| `*_build_args` / `build_args` / `args` | replaced | replaced (most specific set wins) |

### YAML anchors cannot extend an array

There is no sequence-splice in YAML. Aliasing a list as a list *item* nests it
and the config fails to load with `cannot unmarshal !!seq into string`:

```yaml
# BROKEN — do not do this
binary_build_args:
  - *default_args     # inserts a LIST as one element
  - -ldflags
```

To extend build args for one environment, restate the list. To layer them
without duplication, use Layer 2 instead — put invariant settings on the binary
spec and let the environment vary only what changes.

## Cross-Compiling With Per-Target Settings

Use `platforms:` when a target needs its own environment or flags. This builds
every target in a single `gopro build binary` run:

```yaml
default:
  binary_build_env: [CGO_ENABLED=1]     # reaches every target

build:
  binaries:
    - name: api
      platforms:
        - name: linux/amd64             # inherits CGO_ENABLED=1, nothing else
        - name: linux/arm64
          env: [CC=aarch64-linux-gnu-gcc]   # scoped to arm64 ONLY
```

Do **not** model per-target settings as separate environments — that requires one
`gopro build binary` invocation per target and duplicates every unrelated
setting. `platforms:` exists for exactly this.

Key behaviors:
- A **host binary is always built first**, named `{name}` with no `GOOS`/`GOARCH`
  pinned, in addition to one `{name}_{GOOS}_{GOARCH}` per platform. There is no
  flag to suppress it.
- `GOOS`/`GOARCH` derived from the platform name outrank any set in `build_env`.
- The flat `platform: [linux/amd64, darwin/arm64]` form is **deprecated** but
  still honored and folded into `platforms`. A name in both keeps its first-seen
  position and takes the `platforms` entry, so the two can be mixed without
  building twice. Move a target to `platforms` as soon as it needs `env` or `args`.

## Important: Use `config.yaml` and `secret.env`, Not `.env`

GoPro projects store application configuration in `config.yaml` (or other structured config files like `*.json`) and secrets in `secret.env` — **not** `.env` files. Do NOT create `.env` files for GoPro projects.

- **`config.yaml`** — structured application config (database hosts, ports, feature flags, etc.), placed in environment config directories (e.g., `env/default/config/api/config.yaml`)
- **`secret.env`** — secret key-value pairs (passwords, API keys, tokens), placed alongside config files (e.g., `env/default/config/api/secret.env`)

These files live under `env/<environment>/config/<component>/` and are rendered by `gopro generate config` into the output directory. Templates can read from them at generation time using `FromConfigFile`, `FromConfigJSON`, and `FromSecretEnv` functions.

```
env/default/config/api/
├── config.yaml          # application config
├── secret.env           # secrets (KEY=VALUE format)
└── cert/                # other config files (certs, etc.)
```

## Important: `secret.env` Is for Rendering Only — Never in Output or Git

The `secret.env` file exists **solely as a source for template rendering** — it is read by `FromSecretEnv` directly from the **source** config directory (e.g., `env/default/config/api/secret.env`) during `gopro generate`. It is **never copied to the output directory** (`dist/`).

**Do NOT list `secret.env` in the `files` patterns** of `generate.configs` — it should not appear in generated output. `FromSecretEnv` reads it from the source directory, so it does not need to be in `dist/`.

**`secret.env` must be added to the project's `.gitignore`** to prevent secrets from being committed to version control. When scaffolding or initializing a GoPro project, always ensure `.gitignore` includes:

```gitignore
# GoPro secrets — rendering source only, never commit
secret.env
```

In CI/CD or production pipelines, `secret.env` files should be provisioned from a secrets manager (Vault, AWS Secrets Manager, etc.) before running `gopro generate`, and discarded afterward.

## Template System

Templates use `[[` and `]]` delimiters (not `{{ }}`). Files prefixed with `template.` are rendered as Go templates; the prefix is stripped in output.

### Template Context

```
.Name      - Component name (string)
.Project   - Full project config (types.Project)
.EnvName   - Selected environment name, "" when -e was not given (string)
.Env       - Current environment config (types.EnvSpec), default merged with the selected env
```

### Built-in Functions

| Function | Example | Description |
|----------|---------|-------------|
| `GetEnvKey` | `[[ GetEnvKey "DB_URL" ]]` | Env var with product prefix (e.g. `MYAPP_DB_URL`) |
| `GetConfigDir` | `[[ GetConfigDir "api" ]]` | Config directory from binary spec |
| `GetImageName` | `[[ GetImageName "api" ]]` | Full image name: `prefix/repo:tag` |
| `GetImageNameWithTag` | `[[ GetImageNameWithTag "api" "stable" ]]` | Same, with an explicit tag instead of the configured one |
| `FromFile` | `[[ FromFile "/path/to/file" ]]` | Read file contents |
| `FromConfigFile` | `[[ FromConfigFile "api" "db.conf" ]]` | Read from generated config dir |
| `FromConfigJSON` | `[[ FromConfigJSON "api" "config.json" "db.host" ]]` | Extract JSON value by path |
| `FromSecretEnv` | `[[ FromSecretEnv "api" "DB_PASS" ]]` | Read key from secret.env |

All [Sprig v3](http://masterminds.github.io/sprig/) functions are also available (upper, lower, default, b64enc, list, join, etc.).

### Template Example (Kubernetes Deployment)

```yaml
# env/default/kubernetes/api/template.deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: [[ .Name ]]
spec:
  template:
    spec:
      containers:
      - name: [[ .Name ]]
        image: [[ GetImageName .Name ]]
        env:
        - name: [[ GetEnvKey "CONFIG_DIR" ]]
          value: [[ GetConfigDir .Name ]]
```

## Workflows

### Setting Up a New Project

The first step is always to design a proper `project.yaml` — this is the blueprint for the entire project. Define the project metadata (`product`, `module`, `version`), directory layout (`binary_src`, `config_src`, etc.), environments, build targets, and generate specs before running anything. Use `gopro example` to generate a reference template, then customize it to match the project's needs.

Once `project.yaml` is ready, run `gopro init` to scaffold the project. This creates all the directories defined in the config (binary source, image build, environment config/kubernetes/docker-compose dirs) and initializes `go.mod` and git.

```bash
# Step 1: Design project.yaml first (use example as a starting point)
gopro example                    # generates example project.yaml in current dir
# Edit project.yaml to define your project structure, environments, and targets

# Step 2: Initialize the project from project.yaml
gopro init                       # creates dirs, go.mod, git init

# Step 3: Add source code, templates, and Dockerfiles
```

Do NOT create directories manually or run `go mod init` yourself — `gopro init` handles all of that based on `project.yaml`.

**`gopro init` must be completed before writing any code.** The scaffolded directory structure and `go.mod` are prerequisites for all implementation work. Writing code before `gopro init` risks placing files in wrong locations, missing required directories, or having an inconsistent module setup.

### Local Development

```bash
gopro build binary -e local
gopro generate config -e local
gopro generate docker-compose -e local
cd dist/local && docker-compose up
```

### Production Release

```bash
git tag v1.0.0 && git push --tags
gopro build binary -e prod
gopro build image -e prod --push --latest    # pushes :v1.0.0 and :latest
gopro generate config -e prod
gopro generate kubernetes -e prod
kubectl apply -f dist/prod/kubernetes/
```

### Publishing a Moving `:latest` Alongside the Version

`--latest` (`-l`) tags the freshly built image as `:latest` and pushes that too,
so one run publishes both references. Never reach for manual `docker tag` /
`docker push`, and never create a duplicate environment that differs only in
`image_tag` — both are unnecessary:

```bash
gopro build image -e prod --push --latest
# pushes registry/myapp/api:v1.0.0, then tags and pushes registry/myapp/api:latest
```

The extra tag is skipped when the image already builds as `:latest`, so nothing
is pushed twice. `--latest` requires `--push`; alone it warns and does nothing.
Images with `no_push: true` are skipped entirely.

### Generate Dependencies Order

When Kubernetes templates reference config values (via `FromConfigFile`, `FromConfigJSON`, `FromSecretEnv`), generate configs first:

```bash
gopro generate config -e prod
gopro generate kubernetes -e prod   # Can now read from dist/prod/config/
```

## Important: Every Project Needs at Least One Environment Directory

GoPro projects **must** have at least one environment directory under `env/` (e.g., `env/default/`) containing config, Kubernetes, or Docker Compose templates. The `env/` directory is where all template source files live — without it, `gopro generate` commands have nothing to render.

- `env/default/` holds base templates shared across all environments.
- Environment-specific directories (e.g., `env/local/`, `env/prod/`) provide overlay files that are merged on top of the defaults.
- The `config_src`, `kubernetes_src`, and `docker_compose_src` fields in `project.yaml` point into these directories.

```yaml
# project.yaml — these paths must exist under env/
default:
  config_src: env/default/config
  kubernetes_src: env/default/kubernetes
  docker_compose_src: env/default/docker-compose

env:
  local:
    config_src: env/local/config       # overrides for local
  prod:
    config_src: env/prod/config        # overrides for prod
```

If you skip creating the environment directory, builds will succeed but `gopro generate` will produce no output or fail with missing path errors.

## Directory Layout

```
project-root/
├── project.yaml
├── build/
│   ├── binary/{name}/          # Binary source (or cmd/{name})
│   └── image/{name}/Dockerfile # Docker build contexts
├── env/                        # REQUIRED: at least env/default/
│   ├── default/
│   │   ├── config/{name}/      # Base config templates
│   │   ├── kubernetes/{name}/  # Base K8s templates
│   │   └── docker-compose/     # Base docker-compose templates
│   ├── local/                  # Local overrides
│   └── prod/                   # Production overrides
├── bin/                        # Built binaries (gitignored)
└── dist/                       # Generated output (gitignored)
```

## Important: Always Use `gopro build binary` Instead of `go build`

Projects using GoPro **must** be built with `gopro build binary`, not raw `go build`. The `gopro build binary` command automatically injects version, git, and product metadata into the binary via `-ldflags` (using `framingo/pkg/types/info`). A raw `go build` will produce a binary with empty version/build information.

```bash
# CORRECT - injects version, git tag, branch, build time, product info
gopro build binary -e local

# WRONG - no build info injected, `gopro version` will show empty fields
go build -o myapp ./cmd/myapp
```

Thirteen fields are injected: `ProductName`, `ProductModel`, `ProductVersion`,
`BuildVersion`, `BuildType`, `BuildDate`, `BuildTime`, `GitBranch`, `GitTag`,
`GitCommit`, `ProjectName`, `ProjectPath`, and `ProjectRoot`. They are set on
`github.com/xhanio/framingo/pkg/types/info` package vars and readable at runtime
(e.g. via `gopro version`). The `--product-model`, `--product-version`,
`--build-version`, `--build-type`, and `--build-date` flags each override the
matching field. See `references/REFERENCE.md` for where every value comes from.

## Troubleshooting

- **Config not found**: Run `gopro example` or use `-c path/to/config.yaml`
- **Blank git metadata**: Not an error — git info is best-effort, so outside a repository commands still succeed and inject empty `GitTag`/`GitBranch`/`GitCommit`. Initialize git and make a commit (`git init && git add . && git commit -m "init"`) so `git describe --tags --always` has something to report
- **Template render error**: Check dependency order (configs before K8s), verify file paths
- **Cross-compile CGO error**: Either set `CGO_ENABLED=0` to drop cgo entirely, or keep `CGO_ENABLED=1` and supply a cross compiler per target via `platforms[].env` (e.g. `CC=aarch64-linux-gnu-gcc`) — the toolchain must exist on the build host
- **`cannot unmarshal !!seq into string`**: A YAML alias was spliced into an array. Restate the list; anchors cannot concatenate sequences
- **Env var set in `default` went missing**: An `env.<name>` override replaced the whole array. Restate every value it still needs
- **Dockerfile not found**: Verify `build_src` or `image_build_src` paths contain a Dockerfile
- **Empty version info**: Binary was built with `go build` instead of `gopro build binary`
- **Generated output missing old files**: Expected when `config_tgt`/`kubernetes_tgt` names a directory of its own — it is wiped before each render so output reflects only current sources
- **Stale files persist instead**: The opposite case, an in-place render. With `config_tgt`/`kubernetes_tgt` unset the output goes beside the templates, so the target is *not* cleared — clearing it would delete the templates. Set a distinct target (e.g. `dist/`) for a clean render every time
- Use `-v` (verbose) flag on any command for detailed debug output; it prints the resolved build env and command line
