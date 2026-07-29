# GoPro Configuration Reference

## project.yaml Complete Field Reference

### Top-Level Fields

| Field | Required | Description |
|-------|----------|-------------|
| `product` | Yes | Product name, used for env var prefixes and metadata |
| `model` | No | Product model identifier |
| `version` | No | Product version string |
| `domain` | No | Domain name |
| `module` | No | Go module path (auto-detected from go.mod) |

### Environment Settings (default / env.{name})

| Field | Default | Description |
|-------|---------|-------------|
| `binary_src` | `build/binary` | Source directory for binary code |
| `binary_tgt` | `bin/` | Output directory for compiled binaries |
| `binary_build_env` | `[]` | Environment variables for go build (e.g., `CGO_ENABLED=0`) |
| `binary_build_args` | `[]` | Additional go build arguments |
| `binaries` | `[]` | List of binary names to build |
| `image_build_src` | `build/image` | Source directory for Dockerfiles |
| `image_prefix` | `""` | Docker registry prefix |
| `image_tag` | `latest` | Default image tag |
| `image_build_env` | `[]` | Environment applied to the `docker build` / `docker tag` processes |
| `images` | `[]` | List of image names to build |
| `config_src` | `""` | Config template source directory |
| `config_tgt` | `""` | Config output directory |
| `configs` | `[]` | List of config names to generate |
| `kubernetes_src` | `""` | K8s template source directory |
| `kubernetes_tgt` | `""` | K8s output directory |
| `kubernetes_templates` | `[]` | List of K8s template names |
| `docker_compose_src` | `""` | Docker Compose template source |
| `docker_compose_tgt` | `""` | Docker Compose output directory |

### Build Specification

#### Binary Spec

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Binary name (must be listed in `binaries`) |
| `src` | No | Custom source path (default: `{binary_src}/{name}`) |
| `config_dir` | No | Config directory path used in templates |
| `build_env` | No | Env vars for this binary, **merged** over `binary_build_env` |
| `build_args` | No | Go build args for this binary, **replacing** `binary_build_args` |
| `platforms` | No | Cross-compile targets with optional per-target env/args (see below) |
| `platform` | No | **Deprecated.** Flat target list `["linux/amd64", "darwin/arm64"]`; folded into `platforms` |

#### Platform Spec (entries of `platforms`)

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Target as `GOOS/GOARCH`, e.g. `linux/arm64` |
| `env` | No | Env vars for this target only, **merged** over `build_env` |
| `args` | No | Go build args for this target only, **replacing** `build_args` |

```yaml
build:
  binaries:
    - name: api
      build_env: [CGO_ENABLED=1]
      platforms:
        - name: linux/amd64
        - name: linux/arm64
          env: [CC=aarch64-linux-gnu-gcc]   # this target only
```

A `platform` name present in both lists keeps its first-seen position and takes
the `platforms` entry, so the two forms mix without producing a duplicate build.

**Output naming:** a host binary `{name}` is always built first with no
`GOOS`/`GOARCH` pinned, plus one `{name}_{GOOS}_{GOARCH}` per platform. The host
build cannot be suppressed. `GOOS`/`GOARCH` from the platform name outrank any
values in the build environment.

#### Image Spec

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Image name (must be listed in `images`) |
| `build_from` | No | Pull and tag existing image instead of building |
| `base` | No | Base image for Dockerfile (use `$name` to cross-reference) |
| `build_src` | No | Dockerfile directory (default: `{image_build_src}/{name}`) |
| `prefix` | No | Override `image_prefix` for this image |
| `repo` | No | Override repository name (default: `name`) |
| `tag` | No | Override `image_tag` for this image |
| `no_push` | No | Skip pushing this image when `--push` is used |

#### Generate Spec

**Config/Kubernetes entries:**

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Component name |
| `files` | No | Glob patterns for files to process |

**Docker Compose entry:**

| Field | Required | Description |
|-------|----------|-------------|
| `files` | No | Glob patterns for files to process |

## Docker Build Arguments

When building images from Dockerfiles, these four build args are automatically
provided. There is **no configuration hook for adding further `--build-arg`
values** — to influence a build in other ways, set `image_build_env`.

| Arg | Value |
|-----|-------|
| `NAME` | Image name |
| `BASE` | Base image, with `$name` cross-references already resolved |
| `CONFIG_TGT` | Config target directory |
| `CONFIG_DIR` | Component config directory, from the matching binary's `config_dir` |

The build runs as:

```bash
docker build -t <image> --no-cache \
  --build-arg NAME=... --build-arg BASE=... \
  --build-arg CONFIG_TGT=... --build-arg CONFIG_DIR=... \
  -f <project_root>/<build_src>/Dockerfile <project_root>
```

Builds always use `--no-cache`, and the build context is always the project root,
so a Dockerfile may `COPY` from anywhere in the repository.

## Image Tagging and Push

Image references resolve as `[prefix/]repo:tag`, where `prefix` falls back to
`image_prefix`, `repo` to the image `name`, and `tag` to `image_tag` then
`latest`.

| Flag | Effect |
|------|--------|
| `-p, --push` | Push after building; skipped for images with `no_push: true` |
| `-l, --latest` | Additionally tag and push `:latest`. Requires `--push` — alone it warns and does nothing |

`--latest` resolves the second reference through the same prefix/repo logic, and
skips the extra push when the image already builds as `:latest`. This replaces
any need for manual `docker tag`/`docker push` or a duplicate environment that
differs only in `image_tag`.

## Build Metadata Injection

GoPro injects metadata into binaries via `-ldflags` using the framingo package:

- Git: branch, tag, commit hash
- Build: time, version, type
- Product: name, model, version

Access in Go code:

```go
import "github.com/xhanio/framingo/pkg/types/info"

fmt.Println(info.BuildVersion)  // Git tag or override
fmt.Println(info.BuildTime)     // Build timestamp
fmt.Println(info.GitTag)        // Current git tag
fmt.Println(info.GitBranch)     // Current branch
fmt.Println(info.GitCommit)     // Commit hash
```

## Environment Merging Behavior

There are two separate layers, and they resolve differently.

### Layer 1: `default` → `env.{name}` — arrays REPLACED

Environment configs use `go.uber.org/config`. Maps merge key-by-key, but
sequences are replaced wholesale:

```yaml
default:
  binary_build_env: [CGO_ENABLED=0, GOOS=linux]  # Both values

env:
  local:
    binary_build_env: [CGO_ENABLED=1]  # Replaces entire array; GOOS is gone
```

Building with `-e local` applies **only** `CGO_ENABLED=1`. Restate every value
the environment still needs:

```yaml
env:
  local:
    binary_build_env: [CGO_ENABLED=1, GOOS=linux]
```

The asymmetry makes this easy to miss: scalars and untouched arrays inherit
correctly, so only the specific array you overrode is affected — and if the
dropped values match your host, the build still succeeds locally and breaks only
on another platform or in CI.

### YAML anchors cannot extend a sequence

YAML aliases substitute a *node*; they do not splice a sequence's items into an
enclosing sequence. There is no sequence equivalent of the `<<:` merge key, which
works on mappings only. Aliasing a list as a list item nests it:

```yaml
# BROKEN
default:
  binary_build_args: &default_args [-v, -ldflags, '-s']
env:
  prod:
    binary_build_args:
      - *default_args              # yields [[-v, -ldflags, -s], -ldflags, ...]
      - -ldflags
```

Loading this fails with:

```
yaml: unmarshal errors:
  line N: cannot unmarshal !!seq into string
```

To extend build args for one environment, restate the list literally. To layer
settings without duplication, use Layer 2 instead.

### Layer 2: env → binary → platform — env MERGED, args REPLACED

At build time each binary resolves its settings through three levels:

| Chain | Behavior |
|-------|----------|
| `binary_build_env` → `build_env` → platform `env` | **Merged key-wise** — each level overrides only the variables it names |
| `binary_build_args` → `build_args` → platform `args` | **Replaced** — the most specific level that sets a list wins |

Args replace because Go build flags are positional and repeatable, so a key-wise
merge cannot distinguish an override from an accumulation. Only an unset level
inherits, which means an explicit `[]` is how you build with no arguments.

```yaml
default:
  binary_build_env: [CGO_ENABLED=1, FOO=from_default]
  binary_build_args: [-v]

build:
  binaries:
    - name: api
      build_env: [FOO=from_binary]    # CGO_ENABLED=1 survives; FOO overridden
      build_args: []                  # host + amd64 build with NO args
      platforms:
        - name: linux/amd64
        - name: linux/arm64
          env: [CC=aarch64-linux-gnu-gcc]
          args: [-v, -tags=netgo]     # arm64 only
```

Resolved environments:

```
host          CGO_ENABLED=1 FOO=from_binary
linux/amd64   CGO_ENABLED=1 FOO=from_binary GOOS=linux GOARCH=amd64
linux/arm64   CGO_ENABLED=1 FOO=from_binary CC=aarch64-linux-gnu-gcc GOOS=linux GOARCH=arm64
```

Because Layer 2 genuinely merges environment variables, it is the right place for
settings that would otherwise be duplicated across environments: put invariant
vars on the binary spec and let `env.{name}` vary only what changes.

## Template Rendering Pipeline

1. Remove the component's target directory (config and Kubernetes only — not docker-compose)
2. Scan source directory for files matching `files` patterns; an empty or absent `files` list processes everything
3. For files with `template.` prefix: render as Go template, strip prefix
4. For other files: copy as-is
5. Default layer renders first, then environment layer overlays
6. Templates use `[[` `]]` delimiters, receive `{Name, Project, EnvName, Env}` context

Because step 1 wipes the target, generated output is always a clean reflection of
the sources; stale files from a previous run never survive. `gopro generate
docker-compose` is the exception — it renders into a shared directory rather than
one named after a component, so it neither clears the target nor accepts an
output flag.

## Template Functions Returning Image Names

| Function | Signature | Notes |
|----------|-----------|-------|
| `GetImageName` | `(name)` | Tag resolved from the image's `tag`, then `image_tag`, then `latest` |
| `GetImageNameWithTag` | `(name, tag)` | Same prefix/repo resolution, explicit tag |

Both return an empty string when no image by that name is defined in
`build.images`.
