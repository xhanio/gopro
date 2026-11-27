# GoPro Configuration Reference

`gopro schema` prints the JSON Schema of the installed gopro; where it and
this reference disagree, the schema is right.

## project.yaml Complete Field Reference

### Top-Level Fields

| Field | Required | Description |
|-------|----------|-------------|
| `product` | Yes | Product name, used for env var prefixes and metadata |
| `model` | No | Product model identifier; sets `info.ProductModel` |
| `version` | No | Product version; falls back to the Git tag |
| `domain` | No | Domain name |
| `module` | No | Go module path; read from the `go.mod` beside project.yaml when unset. `gopro init` needs one of the two |

### Environment Settings (default / env.{name})

Only `image_tag` has a built-in default. The `build/…`, `bin/`, `env/…` and
`dist/…` paths used throughout are the layout `gopro example` writes, not values
gopro falls back to.

| Field | Default | Description |
|-------|---------|-------------|
| `binary_src` | `""` | Parent directory of binary sources; a binary builds from `{binary_src}/{name}` |
| `binary_tgt` | `""` | Output directory for compiled binaries (unset: the working directory) |
| `binary_build_env` | `[]` | Environment variables for go build (e.g., `CGO_ENABLED=0`) |
| `binary_build_args` | `[]` | Additional go build arguments |
| `binaries` | `[]` | List of binary names to build |
| `image_build_src` | `""` | Parent directory of Dockerfiles; an image builds from `{image_build_src}/{name}/Dockerfile` |
| `image_prefix` | `""` | Docker registry prefix |
| `image_tag` | `latest` | Default image tag |
| `image_build_env` | `[]` | Environment applied to the `docker build` / `docker tag` processes |
| `image_build_args` | `[]` | Extra `docker build` flags; `-t`, `-f` and the injected build args are reserved |
| `images` | `[]` | List of image names to build |
| `config_src` | `""` | Config template source directory |
| `config_tgt` | `""` | Config output directory (unset: render in place, beside the templates) |
| `configs` | `[]` | List of config names to generate |
| `kubernetes_src` | `""` | K8s template source directory |
| `kubernetes_tgt` | `""` | K8s output directory (unset: render in place) |
| `kubernetes_templates` | `[]` | List of K8s template names |
| `docker_compose_src` | `""` | Docker Compose template source |
| `docker_compose_tgt` | `""` | Docker Compose output directory (unset: the working directory); never cleared |

### Component Names

Every command that reads project.yaml first rejects, reporting all at once: a
component name that is not a single path element (empty, `.`, `..`, or holding
`/` or `\`), since it names files and directories and generation clears the
component's target; a name defined twice in one section or listed twice in one
list; a `binaries`/`images`/`configs`/`kubernetes_templates` entry, in `default`
or any `env`, that names nothing defined; and a `$name` base naming no image.
Defining a component without listing it is fine.

### Build Specification

#### Binary Spec

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Binary name, a single path element; built only in environments whose `binaries` lists it |
| `version` | No | Application's own version, injected as `ApplicationVersion` (default: the product version) |
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
| `name` | Yes | Image name, a single path element; built only in environments whose `images` lists it |
| `build_from` | No | Pull and tag existing image instead of building |
| `base` | No | Base image for Dockerfile (use `$name` to cross-reference; the name must be defined in `build.images`) |
| `build_src` | No | Dockerfile directory (default: `{image_build_src}/{name}`) |
| `prefix` | No | Override `image_prefix` for this image |
| `repo` | No | Override repository name (default: `name`) |
| `tag` | No | Override `image_tag` for this image |
| `no_push` | No | Skip pushing this image when `--push` is used |

#### Generate Spec

**Config/Kubernetes entries:**

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Component name, a single path element; generated only in environments whose `configs`/`kubernetes_templates` lists it |
| `files` | No | Glob patterns for files to process (unset: every file). A `secret.env` is always skipped |

**Docker Compose entry:**

| Field | Required | Description |
|-------|----------|-------------|
| `files` | No | Glob patterns for files to process (unset: every file). A `secret.env` is always skipped |

## Docker Build Arguments

When building images from Dockerfiles, these four build args are always
provided:

| Arg | Value |
|-----|-------|
| `NAME` | Image name |
| `BASE` | Base image, with `$name` cross-references already resolved |
| `CONFIG_TGT` | Config target directory |
| `CONFIG_DIR` | Component config directory, from the matching binary's `config_dir` |

`image_build_args` passes extra flags to `docker build` (for example
`--platform`, `--target`, `--secret`, or more `--build-arg`s), placed after
gopro's own and before the Dockerfile and context. `-t`/`--tag`, `-f`/`--file`
and the four build args above are reserved for gopro and rejected.

The build runs as:

```bash
docker build -t <image> [--no-cache] \
  --build-arg NAME=... --build-arg BASE=... \
  --build-arg CONFIG_TGT=... --build-arg CONFIG_DIR=... \
  [image_build_args...] \
  -f <project_root>/<build_src>/Dockerfile <project_root>
```

`--no-cache` is passed unless `--cache` is given. The build context is always
the project root, so a Dockerfile may `COPY` from anywhere in the repository.

That includes every `secret.env` under `env/` unless an ignore file excludes it.
Before each Dockerfile build gopro warns about each one left in the context,
applying the patterns docker will: `<build_src>/Dockerfile.dockerignore` when
it exists (BuildKit reads it instead of the root's), else `.dockerignore`. The
build still runs. `gopro init` adds `**/secret.env` to `.dockerignore`; it does
not exclude `bin/` or `dist/`, which Dockerfiles copy from.

## Image Tagging and Push

Image references resolve as `[prefix/]repo:tag`, where `prefix` falls back to
`image_prefix`, `repo` to the image `name`, and `tag` to `image_tag` then
`latest`.

| Flag | Effect |
|------|--------|
| `-p, --push` | Push after building; skipped for images with `no_push: true` |
| `-l, --latest` | Additionally tag and push `:latest`. Requires `--push` — alone it warns and does nothing |
| `--cache` | Use the docker layer cache; without it builds pass `--no-cache` |
| `--dry-run` | Print each `docker` command, one per line, instead of running it; also on `build binary` and every `generate` subcommand |

`--latest` resolves the second reference through the same prefix/repo logic, and
skips the extra push when the image already builds as `:latest`. This replaces
any need for manual `docker tag`/`docker push` or a duplicate environment that
differs only in `image_tag`.

## Build Metadata Injection

`gopro build binary` injects fifteen fields into the binary via `-ldflags`,
setting package vars on `github.com/xhanio/framingo/pkg/types/info`. A raw
`go build` sets none of them, leaving every field empty.

| Field | Value |
|-------|-------|
| `ApplicationName` | The binary's `name` from `build.binaries` |
| `ApplicationVersion` | The binary's `version` from `build.binaries`, falling back to `ProductVersion` |
| `ProductName` | `product` from project.yaml |
| `ProductModel` | `model` from project.yaml; or `--product-model` |
| `ProductVersion` | `version` from project.yaml, falling back to `BuildVersion`; or `--product-version` |
| `BuildVersion` | The Git tag, or `--build-version` |
| `BuildType` | `--build-type` only |
| `BuildDate` | `--build-date` only |
| `BuildTime` | Time of the build, RFC3339 |
| `GitBranch` | `git rev-parse --abbrev-ref HEAD` |
| `GitTag` | `git describe --tags --always` |
| `GitCommit` | `git rev-parse HEAD` |
| `ProjectName` | The Go module path |
| `ProjectPath` | Project directory relative to `$GOPATH/src`; outside GOPATH, the module path |
| `ProjectRoot` | Absolute working directory of the build |

The three Git values are best-effort: outside a repository the build still
succeeds and they arrive empty. `BuildType` and `BuildDate` have no project.yaml
equivalent — a flag is the only way to set them.

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

Once a level is chosen, gopro assembles the `go build` line from it.
Tool-flag strings (`-ldflags`, `-gcflags`, `-asmflags`, `-gccgoflags`)
**accumulate**: occurrences sharing a package pattern join into one flag at the
position of the first, and the injected metadata `-X` fields join `-ldflags`
after yours, so `[-ldflags, '-s', -ldflags, '-w']` links with `-s -w`. This
differs from raw `go build`, where the last `-ldflags` wins. An empty value
still resets the ones before it. For `-ldflags`, `all=` counts as unpatterned
and any other pattern is an error. `-o` and
`-X github.com/xhanio/framingo/pkg/types/info.*` are reserved for gopro and
rejected, and so is any positional argument (a package path or `--`):
gopro supplies the package. Every other flag passes through unchanged.

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

1. Resolve the target from `-o`/`-t`, then `*_tgt`, then `*_src` — an unset target renders the component in place, beside its templates
2. Remove the component's target directory (config and Kubernetes only — not docker-compose), unless it overlaps a template source, which is the in-place case: clearing it would delete the templates about to be read, so existing files are left alone
3. Scan source directory for files matching `files` patterns; an empty or absent `files` list processes everything. A file named `secret.env` is skipped either way
4. For files with `template.` prefix: render as Go template, strip the prefix from the file name only — the subdirectory is part of the output path
5. For other files: copy as-is
6. Default layer renders first, then environment layer overlays; when the environment does not set its own source, the two are one directory and it renders once
7. Templates use `[[` `]]` delimiters, receive `{Name, Project, EnvName, Env}` context

A `files` pattern without a separator matches by base name at any depth, so
`*.yaml` selects `sub/config.yaml` too. A pattern containing one is matched
against the whole relative path, keeping `cert/*` scoped to `cert/`.

Because step 2 clears the target, config and Kubernetes output reflects only
current sources, except for an in-place render. `gopro generate docker-compose`
differs in steps 1 and 2: it has no output flag, writes to `docker_compose_tgt`
(or the working directory when unset), and never clears it, because that
directory is shared rather than named after a component. Output of a renamed or
removed template stays there until deleted by hand.

## Template Functions Returning Image Names

| Function | Signature | Notes |
|----------|-----------|-------|
| `GetImageName` | `(name)` | Tag resolved from the image's `tag`, then `image_tag`, then `latest` |
| `GetImageNameWithTag` | `(name, tag)` | Same prefix/repo resolution, explicit tag |

Both return an empty string when no image by that name is defined in
`build.images`.
