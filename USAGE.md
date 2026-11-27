# GoPro Usage Guide

This guide provides detailed instructions on using GoPro, a Go-based project generator and build tool for managing multi-environment Go projects.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Global Flags](#global-flags)
- [Commands](#commands)
  - [example](#example-command)
  - [schema](#schema-command)
  - [init](#init-command)
  - [version](#version-command)
  - [build binary](#build-binary-command)
  - [build image](#build-image-command)
  - [generate config](#generate-config-command)
  - [generate kubernetes](#generate-kubernetes-command)
  - [generate docker-compose](#generate-docker-compose-command)
- [Configuration File](#configuration-file)
- [Template System](#template-system)
- [Advanced Features](#advanced-features)
- [Common Workflows](#common-workflows)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Installation

### Install from Source

Install directly using Go:

```bash
go install github.com/xhanio/gopro@latest
```

### Build Locally

Clone and build the project:

```bash
git clone https://github.com/xhanio/gopro.git
cd gopro
go build -o gopro .
```

After building, you can move the binary to your PATH:

```bash
mv gopro /usr/local/bin/
```

### Verify Installation

```bash
gopro --help
```

## Quick Start

### 1. Generate Example Configuration

Navigate to your Go project directory and generate an example `project.yaml`:

```bash
gopro example
```

This saves an example configuration file to the current directory. Edit it to
match your project, and set `module` unless the directory already has a `go.mod`
— the next step needs it.

### 2. Initialize Project Structure

```bash
gopro init
```

This command will:
- Initialize Git repository (if not already initialized)
- Initialize Go module (if go.mod doesn't exist)
- Create directory structure based on configuration
- Create/update `.gitignore` and `.dockerignore`

### 3. Build Your Binary

```bash
gopro build binary
```

The compiled binary will be in the `bin/` directory.

## Global Flags

These flags are available for all commands:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | `-c` | `project.yaml` | Path to the configuration file |
| `--environment` | `-e` | (unset) | Target environment (local, prod, or custom). When unset (or `default`), the `default` section is used as-is; an undeclared name is an error |
| `--filter` | `-f` | `.*` | Regex filter for selecting components |
| `--verbose` | `-v` | `false` | Enable verbose output for debugging |
| `--dry-run` | | `false` | Print each command and file operation of `build` and `generate`, one per line, instead of performing it; `init` and `example` refuse it (see [Dry Run](#dry-run)) |
| `--help` | `-h` | `false` | Show help information |

The `example`, `schema` and `version` commands do not load `project.yaml`, so they work in
a directory that has no configuration yet.

### Examples

```bash
# Use a different config file
gopro build binary -c myconfig.yaml

# Build for production environment
gopro build binary -e prod

# Build only components matching "api"
gopro build binary -f "^api$"

# Enable verbose logging
gopro build binary -v

# Combine multiple flags
gopro build binary -e prod -f "api.*" -v
```

## Commands

### example Command

Generate an example `project.yaml` in the current directory.

```bash
gopro example
```

This saves a complete example configuration file that you can use as a starting point. The command will fail if a `project.yaml` already exists to prevent overwriting your configuration.

Its first line points editors at the JSON Schema of the gopro that wrote it;
see [schema](#schema-command).

### schema Command

Print the JSON Schema of `project.yaml` for the installed gopro.

```bash
gopro schema > project.schema.json
```

The schema is embedded in the binary, so it always describes the gopro that
prints it, needs no network, and needs no `project.yaml`. Editors use it for
completion and checking through a comment in `project.yaml`, which `gopro
example` writes as the first line:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/xhanio/gopro/<version>/project.schema.json
```

The schema line `gopro example` writes points at the installed version's
schema on GitHub: its own tag for a release, `main` for a local build. For an
editor without network access, or to pin the schema to the project, save it
beside `project.yaml` and point the line there instead:

```bash
gopro schema > project.schema.json
```

```yaml
# yaml-language-server: $schema=./project.schema.json
```

Either way the line does not follow a gopro upgrade: update the tag in the URL,
or rerun `gopro schema`, so that fields the new version added are not reported
as unknown.

VS Code reads the line with the Red Hat YAML extension; JetBrains IDEs read it
too, or map the schema to `project.yaml` under *Settings → Languages &
Frameworks → Schemas and DTDs → JSON Schema Mappings*.

The schema covers field names, types and formats: unknown fields, a component
name that is not a single path element, a platform that is not `GOOS/GOARCH`,
an environment variable without `=`. Whether each listed name is defined, and
whether a `$name` base exists, it cannot express; gopro checks those when it
loads the file (see [Component Names](#component-names)).

### init Command

Initialize project structure based on configuration.

```bash
gopro init
gopro init -e local        # Directories for default plus local only
gopro init -c custom.yaml  # Use custom config file
```

#### What it Does

1. **Git Repository**: Initializes Git if not already initialized
2. **Go Module**: Runs `go mod init` if `go.mod` doesn't exist, using the `module` field from config when it is set. Without it the command fails outside GOPATH — see [Module Path Not Found](#9-module-path-not-found)
3. **Directory Structure**: Creates all directories specified in:
   - Default environment configuration
   - All environment-specific configurations (or specific environment with `-e`)
4. **Gitignore**: Creates or updates `.gitignore` with:
   - `bin/` - Binary output directory
   - `dist/` - Generated files directory
   - `test/` - Test directory
   - `secret.env` - Secrets read by `FromSecretEnv` at render time
5. **Dockerignore**: Creates or updates `.dockerignore` so it excludes
   `**/secret.env`, appending the rule only when the existing ones do not
   already cover it. `bin/` and `dist/` are left in the build context, since
   Dockerfiles copy binaries and rendered config from them.

#### Example Output

```
initializing project directories
initializing git repository
initializing go module
create directories bin/ for environment default successfully
create directories build/binary for environment default successfully
create directories build/binary/api for environment default successfully
managing .gitignore file
.gitignore file created
added to .gitignore: bin/
added to .gitignore: dist/
added to .gitignore: test/
added to .gitignore: secret.env
added to .dockerignore: **/secret.env
```

The git and go-module lines appear only when those are missing. Entries are
appended in a fixed order, so repeated runs and diffs stay stable.

### version Command

Print gopro's own version.

```bash
gopro version
```

```
Version:    v0.1.11
Build Time: unknown
```

`Version` is the Git tag injected when gopro is itself built with
`gopro build binary`. Otherwise it is the module version the Go toolchain
recorded: the release tag for `go install github.com/xhanio/gopro@latest`, a
pseudo-version for `go build .` in a checkout, and `unknown` when none was
recorded. `Build Time` is the injected build time, or `unknown` for a binary not
built with gopro. Neither depends on the directory the command runs in.

### build binary Command

Build Go binaries with environment-specific configurations.

```bash
gopro build binary [flags]
```

#### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--output` | `-o` | Override output directory |
| `--product-model` | | Override product model metadata |
| `--product-version` | | Override product version metadata |
| `--build-version` | | Override build version (defaults to Git tag) |
| `--build-type` | | Override build type metadata |
| `--build-date` | | Override build date metadata |

#### Examples

```bash
# Build with default environment
gopro build binary

# Build for production (static binary)
gopro build binary -e prod

# Build only specific binary
gopro build binary -f "^api$"

# Override output directory
gopro build binary -o ./output

# Override version information
gopro build binary --build-version v2.0.0 --product-version v2.0.0
```

#### Cross-Platform Builds

Configure cross-platform compilation in `project.yaml`:

```yaml
build:
  binaries:
    - name: api
      src: cmd/api
      platforms:
        - name: linux/amd64
        - name: linux/arm64
        - name: darwin/amd64
        - name: darwin/arm64
        - name: windows/amd64
```

Running `gopro build binary` will produce:
- `api` (host build, no `GOOS`/`GOARCH` pinned — always built first)
- `api_linux_amd64`
- `api_linux_arm64`
- `api_darwin_amd64`
- `api_darwin_arm64`
- `api_windows_amd64`

`GOOS` and `GOARCH` derived from the platform name outrank any values set in the
build environment, so a `GOOS=linux` in `binary_build_env` will not leak into a
`darwin/arm64` build.

##### Deprecated `platform` shorthand

The older flat form is still honored and folded into `platforms`:

```yaml
build:
  binaries:
    - name: api
      platform: [linux/amd64, darwin/arm64]   # deprecated
```

A name appearing in both lists keeps its first-seen position and takes the
`platforms` entry, so the two forms can be mixed without building twice. Move a
platform to `platforms` when it needs its own environment or flags.

#### Per-Binary and Per-Platform Build Settings

Build environment and flags come from three levels, and the two resolve
differently:

```yaml
default:
  binary_build_env: [CGO_ENABLED=1, FOO=from_default]
  binary_build_args: [-v]

build:
  binaries:
    - name: api
      build_env: [FOO=from_binary]        # merged over binary_build_env
      build_args: []                      # replaces binary_build_args
      platforms:
        - name: linux/amd64
        - name: linux/arm64
          env: [CC=aarch64-linux-gnu-gcc] # merged over build_env
          args: [-v, -tags=netgo]         # replaces build_args
```

- **Environment** (`binary_build_env` → `build_env` → platform `env`) is
  **merged key-wise**: each level overrides only the variables it names and
  inherits the rest. Above, every build keeps `CGO_ENABLED=1` and resolves `FOO`
  to `from_binary`, and the `linux/arm64` build additionally gets `CC`.
- **Arguments** (`binary_build_args` → `build_args` → platform `args`)
  **replace** wholesale. Go build flags are positional and repeatable, so a
  key-wise merge cannot distinguish an override from an accumulation. Only an
  unset level inherits — the empty `build_args: []` above means the host and
  `linux/amd64` builds run with no arguments at all, while `linux/arm64`
  restores `-v -tags=netgo`.
- **Assembly**: Once a level is chosen, gopro assembles the `go build` line from it.
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

Note that this is a different mechanism from the `default` → `env` merge in
`project.yaml`, where arrays are replaced rather than merged. See
[Configuration Merging](#configuration-merging).

#### Build Metadata Injection

GoPro automatically injects build metadata using the [framingo](https://github.com/xhanio/framingo) package. The following information is embedded:

- **Git Information**: Branch, tag, commit hash
- **Build Information**: Build version, type, date, and time
- **Application Information**: Application name and version, per binary
- **Product Information**: Product name, model, version
- **Project Information**: Project name, path, root directory

This metadata is injected via `-ldflags` at compile time and can be accessed in your Go code:

```go
import "github.com/xhanio/framingo/pkg/types/info"

func main() {
    fmt.Printf("Version: %s\n", info.BuildVersion)
    fmt.Printf("Build Time: %s\n", info.BuildTime)
    fmt.Printf("Git Tag: %s\n", info.GitTag)
}
```

### build image Command

Build Docker images from Dockerfiles or third-party images.

```bash
gopro build image [flags]
```

#### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--push` | `-p` | Push images to registry after building |
| `--latest` | `-l` | Also tag and push the image as `:latest` (requires `--push`) |
| `--cache` | | Use the docker layer cache (default: build with `--no-cache`) |

#### Examples

```bash
# Build all images
gopro build image

# Build and push to registry
gopro build image --push
gopro build image -p

# Build specific images
gopro build image -f "^api$"

# Build for production environment with push
gopro build image -e prod -p

# Push the versioned tag and :latest alongside it
gopro build image -e prod -p -l
```

#### Image Building Methods

##### 1. Build from Dockerfile

```yaml
build:
  images:
    - name: api
      base: golang:1.21-alpine
      build_src: docker/api  # Optional: defaults to image_build_src/name
      prefix: myregistry.io
      repo: my-api          # Optional: defaults to name
      tag: v1.0.0          # Optional: defaults to environment's image_tag
```

Your Dockerfile can use these build arguments:

```dockerfile
ARG BASE
ARG NAME
ARG CONFIG_TGT
ARG CONFIG_DIR

FROM ${BASE}
WORKDIR /app
COPY ${CONFIG_TGT}/${NAME} ${CONFIG_DIR}
```

##### 2. Build from Third-Party Image

```yaml
build:
  images:
    - name: postgres
      build_from: postgres:13.21-alpine3.21
      prefix: myregistry.io
      tag: v1.0.0
```

This will:
1. Pull `postgres:13.21-alpine3.21`
2. Tag it as `myregistry.io/postgres:v1.0.0`

##### 3. Cross-Reference Images

Use `$image_name` to reference other images as base:

```yaml
build:
  images:
    - name: base
      build_from: golang:1.21-alpine
    - name: api
      base: $base  # References the "base" image defined above
      build_src: docker/api
```

Images build in the order an environment's `images` lists them, so when both
are listed, the base must come first: `images: [base, api]`. Listing `api`
before `base` is an error. A base not listed at all is used as it already
exists.

#### Image Naming Convention

Final image name format: `[prefix/]repo:tag`

- **prefix**: Registry URL (from image definition or environment's `image_prefix`)
- **repo**: Repository name (from image definition's `repo` or defaults to `name`)
- **tag**: Image tag (from image definition's `tag` or environment's `image_tag` or "latest")

Examples:
- `myregistry.io/api:v1.0.0`
- `api:latest` (no prefix)
- `registry.io/myapp/api-service:prod` (custom repo name)

#### Push Behavior

Images are pushed when:
- `--push` flag is provided
- Image does NOT have `no_push: true`

Skip pushing specific images:

```yaml
build:
  images:
    - name: local-only-image
      no_push: true
      build_src: docker/local
```

##### Also Pushing `:latest`

`--latest` (`-l`) tags the freshly built image as `:latest` and pushes that too,
so a release publishes both the versioned reference and the moving one:

```bash
gopro build image -e prod -p -l
# pushes prod-registry.io/myapp/api:v1.0.0
# then tags and pushes prod-registry.io/myapp/api:latest
```

The extra tag is skipped when the image already builds as `:latest`, so nothing
is pushed twice. `--latest` requires `--push`; on its own it prints a warning and
does nothing.

#### Build Execution

Dockerfile builds run as:

```bash
docker build -t <image> [--no-cache] \
  --build-arg NAME=... --build-arg BASE=... \
  --build-arg CONFIG_TGT=... --build-arg CONFIG_DIR=... \
  [image_build_args...] \
  -f <project_root>/<build_src>/Dockerfile <project_root>
```

`--no-cache` is passed unless `--cache` is given, so by default every layer is
rebuilt. The build context is always the project root, so a Dockerfile can `COPY` from
anywhere in the repository. `image_build_env` is applied as the environment for
the `docker build` and `docker tag` invocations.

`image_build_args` passes extra flags to `docker build` (for example
`--platform`, `--target`, `--secret`, or more `--build-arg`s), placed after
gopro's own and before the Dockerfile and context. `-t`/`--tag`, `-f`/`--file`
and the four injected build args are reserved for gopro and rejected.

The build context is the project root, so every `secret.env` under `env/` is sent to docker unless `.dockerignore` excludes it, and a Dockerfile's `COPY . .` puts it in the image. Before each Dockerfile build, gopro
warns about every `secret.env` the patterns docker will apply leave in the
context: those of `<build_src>/Dockerfile.dockerignore` when it exists (BuildKit
reads it in place of the root's), else of `.dockerignore`. The warning does not
stop the build. `gopro init` adds `**/secret.env` to `.dockerignore`.

### generate config Command

Generate configuration files from templates with environment-specific values.

```bash
gopro generate config [flags]
```

#### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | (from config) | Override output directory |
| `--prefix` | `-x` | `template.` | Template file prefix |

#### Examples

```bash
# Generate configs for default environment
gopro generate config

# Generate for local environment
gopro generate config -e local

# Override output directory
gopro generate config -o ./config-output

# Generate specific configs only
gopro generate config -f "^api$"

# Use custom template prefix
gopro generate config -x "tmpl."
```

#### How It Works

1. **Render, then write**: Both layers render first, into memory. Only when
   every template has rendered is the target directory cleared and the output
   written, so output is a clean reflection of the sources, and a template
   that fails leaves the previous output exactly as it was. An in-place render,
   where the target overlaps a template source, is never cleared, because
   clearing it would delete the templates. Before clearing a target, gopro refuses — with an error, deleting nothing — when the clear would take with it what it did not generate: a target directory that is the project root (so `config_tgt: .`, or `-o .`, is rejected), a directory gopro reads sources from in any environment, or another component's output in the same environment. A listed config with
   no template directory in either layer is an error rather than nothing

2. **Two-Layer Rendering**:
   - First: Renders default templates from `default.config_src/config_name/`
   - Second: Renders environment-specific templates from `env.config_src/config_name/`
   - Environment templates overlay/override default templates

3. **Template Processing**:
   - Files with prefix (default: `template.`) are processed as Go templates
   - Template delimiters: `[[` and `]]` (avoids conflicts with JSON/YAML `{{ }}`)
   - Prefix is removed in output: `template.config.yaml` → `config.yaml`
   - Non-template files are copied as-is

4. **File Filtering**:
   - Only files matching patterns in `files` array are processed
   - An empty or absent `files` list processes every file
   - A file named `secret.env` is never written to output, even when a pattern matches it

#### Configuration Example

```yaml
generate:
  configs:
    - name: api
      files:
        - "*.yaml"
        - "*.json"
        - "config/*"
        # secret.env needs no entry — FromSecretEnv reads it from the source, and it is never copied to output

default:
  config_src: env/default/config
  config_tgt: dist/config
  configs: [api]

env:
  local:
    config_src: env/local/config
    config_tgt: dist/local/config
```

Directory structure:
```
env/
├── default/
│   └── config/
│       └── api/
│           ├── template.config.yaml
│           └── static-file.txt
└── local/
    └── config/
        └── api/
            └── template.config.yaml  # Overrides default
```

### generate kubernetes Command

Generate Kubernetes manifests from templates.

```bash
gopro generate kubernetes [flags]
```

#### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-t` | (from config) | Override output directory |
| `--prefix` | `-x` | `template.` | Template file prefix |

#### Examples

```bash
# Generate Kubernetes manifests for default environment
gopro generate kubernetes

# Generate for production
gopro generate kubernetes -e prod

# Override output directory
gopro generate kubernetes -t ./k8s-manifests

# Generate specific templates only
gopro generate kubernetes -f "^api$"
```

As with configs, every template renders before the target is cleared and
written, under the same safety checks (except for an in-place render), and the
default layer renders first with the environment layer overlaid on top.

#### Configuration Example

```yaml
generate:
  kubernetes:
    - name: api
      files:
        - "deployment.yaml"
        - "service.yaml"

default:
  kubernetes_src: env/default/kubernetes
  kubernetes_tgt: dist/kubernetes
  kubernetes_templates: [api]

env:
  prod:
    kubernetes_src: env/prod/kubernetes
    kubernetes_tgt: dist/prod/kubernetes
    image_prefix: prod-registry.io/myapp
    image_tag: v1.0.0
```

#### Template Example

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
        - name: CONFIG_DIR
          value: [[ GetConfigDir .Name ]]
```

After generation (`gopro generate kubernetes -e prod`):

```yaml
# dist/prod/kubernetes/api/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  template:
    spec:
      containers:
      - name: api
        image: prod-registry.io/myapp/api:v1.0.0
        env:
        - name: CONFIG_DIR
          value: /etc/api
```

### generate docker-compose Command

Generate Docker Compose files from templates.

```bash
gopro generate docker-compose [flags]
```

#### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--prefix` | `-x` | `template.` | Template file prefix |

#### Examples

```bash
# Generate docker-compose for default environment
gopro generate docker-compose

# Generate for local development
gopro generate docker-compose -e local
```

Unlike the other two generate commands, this one has no output-directory flag:
output goes to `docker_compose_tgt`, or the current directory when that is unset.
It also does **not** clear the target directory first, since it renders into a
shared directory rather than one named after a component — below, `dist` also
holds the config and Kubernetes output. Output of a template that has since been
renamed or removed therefore stays in the target until you delete it.

#### Configuration Example

```yaml
generate:
  docker_compose:
    files:
      - "docker-compose.yaml"
      - "*.yml"

default:
  docker_compose_src: env/default/docker-compose
  docker_compose_tgt: dist

env:
  local:
    docker_compose_src: env/local/docker-compose
    docker_compose_tgt: dist/local
```

#### Template Example

```yaml
# env/default/docker-compose/template.docker-compose.yaml
version: '3.8'
services:
  api:
    image: [[ GetImageName "api" ]]
    ports:
      - "8080:8080"
    environment:
      - IMAGE_TAG=[[ .Env.ImageTag | default "latest" ]]
```

## Configuration File

The `project.yaml` file is the central configuration for GoPro.

### Top-Level Fields

```yaml
product: myapp                 # Product name (required); also the env var prefix
model: standard                # Product model (optional); sets info.ProductModel
version: v1.0.0                # Product version (optional; falls back to the Git tag)
domain: example.com            # Domain name (optional)
module: github.com/user/myapp  # Go module name (optional; read from go.mod when unset)
```

The field is `module`, not `project`. When it is omitted, GoPro reads the module
path from the `go.mod` sitting next to `project.yaml`.

### Component Names

`build.binaries`, `build.images`, `generate.configs` and `generate.kubernetes`
define components by `name`; the `binaries`, `images`, `configs` and
`kubernetes_templates` lists in `default` and `env` select which ones an
environment acts on. Every command that reads `project.yaml` checks them first
and reports every problem at once:

- A name must be a single path element — not empty, `.` or `..`, and without
  `/` or `\` — because it names files and directories, including the
  component's target directory, which generation clears.
- A name may be defined only once per section, and listed only once per list.
- A listed name must be defined. Defining one without listing it is fine: the
  component is just not acted on in that environment.
- An image `base` of the form `$name` must name an image in `build.images`,
  and when both are in one `images` list, the base must be listed first: images
  build in list order.

```
Error: project.yaml is invalid:
generate.configs[0].name "..": must be a single path element, since it names files and directories
default.binaries: "apii" is not defined in build.binaries
```

### Default Configuration

Base configuration shared across all environments:

```yaml
default:
  # Binary settings
  binary_src: build/binary           # Source directory for binaries
  binary_tgt: bin/                  # Output directory for binaries
  binary_build_env:                 # Environment variables for builds
    - CGO_ENABLED=0
    - GOOS=linux
  binary_build_args:                # Go build arguments
    - -v
    - -ldflags
    - '-s -w'
  binaries: [api, worker]           # Binaries to build

  # Image settings
  image_build_src: build/image      # Dockerfile source directory
  image_prefix: registry.io/myapp   # Registry prefix
  image_tag: latest                 # Default image tag
  image_build_env: []               # Environment for docker build/tag
  image_build_args: []              # Extra docker build flags (-t, -f reserved)
  images: [api, worker]             # Images to build

  # Config settings
  config_src: env/default/config    # Config template source
  config_tgt: dist/config           # Config output directory
  configs: [api, worker]            # Configs to generate

  # Kubernetes settings
  kubernetes_src: env/default/kubernetes
  kubernetes_tgt: dist/kubernetes
  kubernetes_templates: [api]

  # Docker Compose settings
  docker_compose_src: env/default/docker-compose
  docker_compose_tgt: dist
```

### Environment-Specific Configuration

Override default settings for specific environments:

```yaml
env:
  local:
    binary_build_env:
      - CGO_ENABLED=1                # Enable CGO for local development
    config_src: env/local/config
    config_tgt: dist/local/config

  prod:
    binary_build_env:
      - CGO_ENABLED=0
      - GOOS=linux
      - GOARCH=amd64
    binary_build_args:
      - -ldflags=-s -w -extldflags '-static'  # Static binary
    image_prefix: prod-registry.io/myapp
    image_tag: v1.0.0
    config_src: env/prod/config
    config_tgt: dist/prod/config

  staging:
    image_prefix: staging-registry.io/myapp
    image_tag: staging
```

### Build Configuration

Define binary and image build specifications:

```yaml
build:
  binaries:
    - name: api
      version: v1.2.3                  # Optional: app version, defaults to product version
      src: cmd/api                     # Optional: custom source path
      config_dir: /etc/api             # Config directory (for templates)
      build_env: [CGO_ENABLED=0]       # Optional: merged over binary_build_env
      build_args: [-v]                 # Optional: replaces binary_build_args
      platforms:                       # Optional: cross-compile platforms
        - name: linux/amd64
        - name: darwin/arm64
        - name: windows/amd64

    - name: worker
      src: cmd/worker
      config_dir: /etc/worker

  images:
    - name: base
      build_from: golang:1.21-alpine   # Pull and tag existing image

    - name: api
      base: $base                       # Reference to another image
      build_src: docker/api            # Optional: custom Dockerfile path
      prefix: custom-registry.io       # Optional: override prefix
      repo: my-api-service            # Optional: custom repo name
      tag: v2.0.0                     # Optional: override tag
      no_push: false                  # Optional: true skips pushing

    - name: worker
      base: golang:1.21-alpine
      build_src: docker/worker
```

### Generate Configuration

Define configuration and template generation:

```yaml
generate:
  configs:
    - name: api
      files:                          # File patterns to process
        - "*.yaml"
        - "*.json"
        - "config/*"
        # Do NOT include secret.env — it is read from source by FromSecretEnv

    - name: worker
      files: ["*.yaml"]

  kubernetes:
    - name: api
      files:
        - "deployment.yaml"
        - "service.yaml"
        - "configmap.yaml"

  docker_compose:
    files:
      - "docker-compose.yaml"
```

## Template System

GoPro uses Go's `text/template` with custom delimiters and functions.

### Template Delimiters

Use `[[` and `]]` instead of `{{ }}` to avoid conflicts with JSON/YAML:

```yaml
# template.config.yaml
app:
  name: [[ .Name ]]
  version: [[ .Project.Version ]]
  image: [[ GetImageName "api" ]]
```

### Template Context

Templates have access to these variables:

```go
.Name      // Component name being generated (string)
.Project   // Full project configuration (types.Project)
.EnvName   // Selected environment name, "" when -e was not given (string)
.Env       // Current environment configuration (types.EnvSpec)
```

`.Env` is the `default` section with the selected environment merged on top, so
it always reflects the effective values for this run. `.EnvName` is the raw name
as passed to `-e`, useful for stamping the environment into rendered output:

```yaml
metadata:
  labels:
    environment: [[ .EnvName | default "default" ]]
```

### Built-in Template Functions

#### GetEnvKey

Generate environment variable names with product prefix:

```yaml
env:
  - name: [[ GetEnvKey "DATABASE_URL" ]]
    value: "postgres://..."
```

If `product: myapp`, outputs: `MYAPP_DATABASE_URL`

#### GetConfigDir

Get config directory for a binary:

```yaml
volumeMounts:
  - name: config
    mountPath: [[ GetConfigDir "api" ]]
```

Returns the `config_dir` from binary definition (e.g., `/etc/api`)

#### GetImageName

Get fully qualified image name:

```yaml
containers:
  - name: api
    image: [[ GetImageName "api" ]]
```

Returns: `registry.io/myapp/api:v1.0.0`

The tag comes from the image's own `tag`, falling back to the environment's
`image_tag`, then to `latest`. Returns an empty string if no image by that name
is defined in `build.images`.

#### GetImageNameWithTag

Same prefix/repo resolution as `GetImageName`, but with the tag supplied
explicitly instead of resolved from configuration:

```yaml
containers:
  - name: api
    image: [[ GetImageName "api" ]]
    # pin a rollback reference to a known-good tag
    rollbackImage: [[ GetImageNameWithTag "api" "stable" ]]
```

Returns: `registry.io/myapp/api:stable`

Useful for referencing a moving tag such as `latest` (see the `--latest` flag on
[build image](#build-image-command)) alongside the versioned one.

#### FromFile

Read content from any file:

```yaml
data:
  certificate: [[ FromFile "/path/to/cert.pem" ]]
```

#### FromConfigFile

Read from generated config directory:

```yaml
# Reads from dist/config/api/database.conf
data: [[ FromConfigFile "api" "database.conf" ]]
```

#### FromConfigJSON

Extract a value from a generated JSON config file by
[gjson path](https://github.com/tidwall/gjson/blob/master/SYNTAX.md):

```yaml
# Reads dist/config/api/config.json and extracts database.host
host: [[ FromConfigJSON "api" "config.json" "database.host" ]]
```

#### FromSecretEnv

Read one key from a component's `secret.env`. The file is read from the source
config directory, `<config_src>/<name>/secret.env`, not from `dist/`:

```yaml
# Reads env/prod/config/api/secret.env when config_src is env/prod/config
password: [[ FromSecretEnv "api" "DB_PASSWORD" ]]
```

Format of `secret.env`:
```bash
# blank lines and lines starting with # are skipped
DB_PASSWORD=supersecret
API_KEY=abc123
```

Values are used verbatim apart from surrounding whitespace; quotes are not
stripped. Only the selected environment's `config_src` is consulted — there is
no fallback to the default layer's file — and a missing file or key fails the
render.

### Sprig Functions

All [Sprig v3](https://masterminds.github.io/sprig/) functions are available:

```yaml
# String functions
name: [[ .Name | upper ]]
slug: [[ .Name | kebabcase ]]

# Date functions
timestamp: [[ now | date "2006-01-02" ]]

# Encoding functions
encoded: [[ "hello" | b64enc ]]

# List functions
items: [[ list "a" "b" "c" | join "," ]]

# Default values
value: [[ .Project.Version | default "dev" ]]
```

### Template Examples

#### Kubernetes Deployment

```yaml
# env/default/kubernetes/api/template.deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: [[ .Name ]]
  labels:
    app: [[ .Name ]]
    version: [[ .Project.Version ]]
spec:
  replicas: 3
  selector:
    matchLabels:
      app: [[ .Name ]]
  template:
    metadata:
      labels:
        app: [[ .Name ]]
    spec:
      containers:
      - name: [[ .Name ]]
        image: [[ GetImageName .Name ]]
        ports:
        - containerPort: 8080
        env:
        - name: [[ GetEnvKey "CONFIG_DIR" ]]
          value: [[ GetConfigDir .Name ]]
        - name: IMAGE_TAG
          value: [[ .Env.ImageTag | default "latest" ]]
        volumeMounts:
        - name: config
          mountPath: [[ GetConfigDir .Name ]]
      volumes:
      - name: config
        configMap:
          name: [[ .Name ]]-config
```

#### ConfigMap with Secret Injection

```yaml
# env/prod/kubernetes/api/template.configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: [[ .Name ]]-config
data:
  config.yaml: |
    app:
      name: [[ .Name ]]
      version: [[ .Project.Version ]]
    database:
      host: [[ FromConfigJSON .Name "config.json" "database.host" ]]
      password: [[ FromSecretEnv .Name "DB_PASSWORD" ]]
```

#### Application Config

```yaml
# env/default/config/api/template.config.yaml
server:
  port: 8080
  host: 0.0.0.0

image:
  registry: [[ .Env.ImagePrefix ]]
  tag: [[ .Env.ImageTag | default "latest" ]]

product:
  name: [[ .Project.Product ]]
  version: [[ .Project.Version | default "dev" ]]
```

## Advanced Features

### Multi-Environment Builds

Build for multiple environments in sequence:

```bash
# Build binaries for all environments
for env in local staging prod; do
  gopro build binary -e $env
done

# Build and push images for production
gopro build image -e prod --push
```

### Selective Component Building

Use regex filters to build specific components:

```bash
# Build only API-related components
gopro build binary -f "^api$"
gopro build image -f "api.*"
gopro generate config -f "api.*"

# Build multiple components matching pattern
gopro build binary -f "^(api|worker)$"
```

Filters use Go's RE2 regex syntax, which has no lookahead, so an "everything
except" pattern such as `^(?!.*test).*$` is rejected as invalid. Match the
components to keep instead.

### Dry Run

The global `--dry-run` makes every `build` and `generate` subcommand print each
operation that would change something, one per line, and perform none:

| Operation | Printed as |
|---|---|
| A command (`go build`, `docker build`, `pull`, `tag`, `push`) | A shell line with the environment gopro adds: `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ...` |
| Clearing a target directory | `rm -rf <dir>` |
| Creating the docker-compose target | `mkdir -p <dir>` |
| Writing a rendered template | `render <template> <output>` |
| Copying a file as is | `cp <source> <output>` |

```bash
gopro build image -e prod --push --latest --dry-run
gopro generate config -e prod --dry-run
```

A printed command line runs the same when pasted into a shell, its arguments
quoted where needed. Everything without a side effect still runs, so a dry run
fails where the real run would: `project.yaml` is loaded and validated, Git is
queried for the injected metadata, `-ldflags` and `docker build` arguments are
assembled with their checks, and templates are rendered, only not written —
a broken template fails a dry run too. `FromConfigFile` and `FromConfigJSON`
read generated output from an earlier real run, since a dry run writes none.

stdout holds only the operations, so it can be saved or piped; progress
messages are left out, and warnings, such as a `secret.env` left in a Docker
build context, go to stderr. `init` and `example` have no dry run and refuse
the flag, before changing anything; `version` and `schema` change nothing anyway.

### Custom Output Directories

Override output directories for different purposes:

```bash
# Build binaries to custom location
gopro build binary -o ./release/v1.0.0

# Generate configs to custom location
gopro generate config -o ./configs/staging

# Generate k8s manifests to custom location
gopro generate kubernetes -t ./deploy/k8s
```

### Configuration Merging

Environment configurations are merged with defaults using a deep merge, but that
merge is **per key, not per element**. Maps are merged; arrays are replaced
wholesale:

```yaml
default:
  binary_build_env:
    - CGO_ENABLED=0
    - GOOS=linux
  binaries: [api, worker]

env:
  local:
    binary_build_env:
      - CGO_ENABLED=1        # Replaces the entire array
    binaries: [api]          # Replaces the entire array
```

Building with `-e local` here runs with `CGO_ENABLED=1` and **no** `GOOS` — the
`GOOS=linux` from `default` is gone, not merged in. Any array an environment
overrides must be restated in full:

```yaml
env:
  local:
    binary_build_env:
      - CGO_ENABLED=1
      - GOOS=linux           # Restated, or it would be dropped
```

YAML anchors do not help here. Splicing an anchored sequence into a list
produces a nested list, which fails to load:

```yaml
# BROKEN — do not do this
env:
  prod:
    binary_build_args:
      - *default_args        # yaml: cannot unmarshal !!seq into string
      - -ldflags
```

If you need genuinely layered build flags, use the per-binary and per-platform
levels instead of environment overrides — see [Per-Binary and Per-Platform Build
Settings](#per-binary-and-per-platform-build-settings). Note that those levels
merge environment variables key-wise, which the `default` → `env` layer described
here does not.

### Version Management

#### Automatic Git-Based Versioning

GoPro automatically uses Git tags/commits for versioning:

```bash
# Tag your release
git tag v1.0.0
git push --tags

# Build will automatically use v1.0.0
gopro build binary
```

#### Manual Version Override

```bash
# Override build version
gopro build binary --build-version v2.0.0-beta

# Override product version
gopro build binary --product-version v2.0.0
```

### Docker Build Arguments

Four build arguments are always provided to a Dockerfile build:
- `NAME`: Image name
- `BASE`: Base image (with `$image_name` cross-references already resolved)
- `CONFIG_TGT`: Config target directory
- `CONFIG_DIR`: Component config directory, from the matching binary's `config_dir`

Consume them with `ARG` declarations:

```dockerfile
ARG BASE
ARG NAME
ARG CONFIG_TGT
ARG CONFIG_DIR

FROM ${BASE}
COPY ${CONFIG_TGT}/${NAME} ${CONFIG_DIR}
```

Further `--build-arg`s and other `docker build` flags go in `image_build_args`;
`image_build_env` becomes the environment for the `docker build` and
`docker tag` processes:

```yaml
env:
  prod:
    image_build_env:
      - DOCKER_BUILDKIT=1
    image_build_args:
      - --platform=linux/amd64
      - --build-arg=VERSION=1.2.3
```

GoPro's own `-t`, `-f` and the four build args above are reserved, so setting
them in `image_build_args` is an error.

Builds run with `--no-cache` unless `--cache` is given.

### Template File Patterns

Control which files are processed:

```yaml
generate:
  configs:
    - name: api
      files:
        - "*.yaml"           # All YAML files
        - "*.json"           # All JSON files
        - "config/*"         # Everything in config/ subdirectory
        # Do NOT include secret.env — it is only used as a rendering source by FromSecretEnv
```

## Common Workflows

### Local Development Workflow

```bash
# 1. Initialize project
gopro init -e local

# 2. Build binaries for local development (with CGO)
gopro build binary -e local

# 3. Generate local configs
gopro generate config -e local

# 4. Generate docker-compose for local testing
gopro generate docker-compose -e local

# 5. Run with docker-compose
cd dist/local
docker-compose up
```

### Production Release Workflow

```bash
# 1. Update version in Git
git tag v1.0.0
git push --tags

# 2. Build static binaries for production
gopro build binary -e prod

# 3. Build and push Docker images
gopro build image -e prod --push

# 4. Generate production configs
gopro generate config -e prod

# 5. Generate Kubernetes manifests
gopro generate kubernetes -e prod

# 6. Apply to cluster (-R: manifests sit in one subdirectory per template)
kubectl apply -R -f dist/prod/kubernetes/
```

### Multi-Platform Release Workflow

```yaml
# project.yaml
build:
  binaries:
    - name: cli
      src: cmd/cli
      build_env: [CGO_ENABLED=0]     # static, so every target cross-compiles
      platforms:
        - name: linux/amd64
        - name: linux/arm64
        - name: darwin/amd64
        - name: darwin/arm64
        - name: windows/amd64
```

```bash
# Build for all platforms
gopro build binary -e prod

# Package releases
cd bin/
tar -czf cli-linux-amd64.tar.gz cli_linux_amd64
tar -czf cli-darwin-arm64.tar.gz cli_darwin_arm64
# Windows output is cli_windows_amd64, with no .exe suffix; add it when packaging
cp cli_windows_amd64 cli.exe && zip cli-windows-amd64.zip cli.exe
```

### Staging Environment Workflow

```yaml
# project.yaml
env:
  staging:
    image_prefix: staging-registry.io/myapp
    image_tag: staging-latest
    config_src: env/staging/config
    config_tgt: dist/staging/config
    kubernetes_src: env/staging/kubernetes
    kubernetes_tgt: dist/staging/kubernetes
```

```bash
# Deploy to staging
gopro build image -e staging --push
gopro generate config -e staging
gopro generate kubernetes -e staging
kubectl apply -R -f dist/staging/kubernetes/ --namespace=staging
```

### Configuration Template Testing

```bash
# Generate configs for review
gopro generate config -e local -v

# Check generated files
cat dist/local/config/api/config.yaml

# Test with specific component
gopro generate config -e local -f "^api$" -v
```

## Troubleshooting

### Common Issues

#### 1. Command Not Found

```
bash: gopro: command not found
```

**Solution**: Ensure GoPro is in your PATH:
```bash
export PATH=$PATH:/path/to/gopro
# Or move to system PATH
mv gopro /usr/local/bin/
```

#### 2. Configuration File Not Found

```
Error: error applying options: open project.yaml: no such file or directory
```

**Solution**: Specify config file path or generate an example:
```bash
gopro example                              # Generate example project.yaml
gopro build binary -c /path/to/project.yaml  # Or specify a path
```

#### 3. Empty Git Metadata in a Build

Git information is collected on a best-effort basis: outside a repository, or
with `git` unavailable, commands still succeed and simply inject empty
`GitTag`, `GitBranch`, and `GitCommit` values. There is no error — the symptom
is a binary whose `info.Git*` fields come back blank, along with the build and
product versions when those derive from the Git tag.

**Solution**: Initialize a Git repository and create at least one commit, so
`git describe --tags --always` has something to report:
```bash
git init
git add .
git commit -m "Initial commit"
```

Or run `gopro init`, which initializes the repository for you.

#### 4. Template Rendering Error

```
Error: template: template.config.yaml:1:6: executing "template.config.yaml" at
<FromSecretEnv "api" "DB_PASS">: error calling FromSecretEnv: failed to render
from api secret.env: open env/default/config/api/secret.env: no such file or directory
```

**Solution**: Ensure `secret.env` exists in the source config directory (e.g., `env/default/config/api/secret.env`). `FromSecretEnv` reads directly from the source directory, not from `dist/`. Also ensure configs are generated before kubernetes if k8s templates reference config files:
```bash
gopro generate config -e prod
gopro generate kubernetes -e prod
```

#### 5. Docker Build Failures

The message comes from Docker itself; GoPro reports only the exit status:

```
ERROR: failed to build: failed to solve: failed to read dockerfile: open Dockerfile: no such file or directory
Error: exit status 1
```

**Solution**: Verify Dockerfile location matches configuration:
```yaml
build:
  images:
    - name: api
      build_src: docker/api  # Must contain Dockerfile
```

Check directory structure:
```
docker/
└── api/
    └── Dockerfile
```

#### 6. Cross-Compilation Issues

Cross-compiling with `CGO_ENABLED=1` requires a C toolchain for the target.
When one is missing the failure comes from the Go toolchain or the C compiler,
not from GoPro — typically a `cc`/`gcc` "command not found", or an error about
building a cgo package for the target platform.

**Solution**: Either drop cgo for the cross-compiled targets:
```yaml
env:
  prod:
    binary_build_env:
      - CGO_ENABLED=0
```

Or keep cgo and point each target at its own cross compiler, which is what
per-platform `env` exists for:
```yaml
build:
  binaries:
    - name: api
      build_env: [CGO_ENABLED=1]
      platforms:
        - name: linux/arm64
          env: [CC=aarch64-linux-gnu-gcc]
```
The named toolchain must be installed on the build host.

#### 7. Product Name Missing

```
Error: product name is required in project.yaml configuration
```

**Solution**: `gopro init` requires a `product` at the top of `project.yaml`:
```yaml
product: myapp
```

#### 8. Stale Files Left in the Output Directory

Generation normally clears a component's target directory first, so output
reflects only current sources. It deliberately does **not** do so when the
target overlaps a template source — the in-place case, where clearing would
delete the templates about to be rendered.

**Solution**: If you want every render to start clean, set `config_tgt` (or
`kubernetes_tgt`) to a directory separate from `config_src`, such as `dist/`.
Leaving the target unset is a supported layout, not an error; it just means old
output persists alongside the templates.

`gopro generate docker-compose` never clears its target, whatever the layout: it
renders into a shared directory, so output of a renamed or removed template
stays until you delete it.

#### 8a. Generation Refuses to Clear a Target

```
Error: config build: clearing build would clear a directory of the project root; give its output a directory of its own, such as dist/
Error: config api: clearing env/default/kubernetes/api would delete env/default/kubernetes/api, which gopro reads sources from
Error: kubernetes template api: clearing dist/api would delete the output of config api in dist/api
```

Clearing that target would delete files gopro did not generate there, so
nothing is cleared or written. Give each kind of output a directory of its own,
such as `dist/config` and `dist/kubernetes`, outside every `*_src`.

A `has no templates` error means a listed component has no template directory
in either layer of the selected environment; create one, or drop the name from
that environment's list.

#### 9. Module Path Not Found

With `module` unset and no `go.mod` beside `project.yaml`, the module path is
empty. Builds still run, but inject an empty `ProjectName`, and `gopro init`
stops at `initializing go module`: it runs `go mod init` without a path, which
Go refuses outside GOPATH.

```
Error: go mod init: exit status 1; set module in project.yaml, go could not infer a module path here
```

**Solution**: Set it explicitly:
```yaml
module: github.com/user/myapp
```

Or initialize the Go module so it can be read from `go.mod`:
```bash
go mod init github.com/user/myapp
```

The key is `module`, not `project`.

### Non-interactive Runs

Every command gopro runs — `go`, `docker`, `git` — gets no standard input, so
one that would prompt fails at once instead of blocking a CI job or an agent.
Log in to registries beforehand (`docker login`), since `--push` cannot ask.

### Debug Mode

Enable verbose output for debugging:

```bash
gopro build binary -v
gopro generate config -v -e prod
```

This adds:
- Each executed command with its arguments
- The environment variables gopro adds to it (never the inherited environment)
- The assembled `docker build` arguments, and a notice when a render is in place

### Validation

Verify your configuration:

```bash
# Check if binary source exists
ls -la build/binary/api

# Check if Dockerfile exists
ls -la build/image/api/Dockerfile

# Check template files
ls -la env/default/config/api/

# Verify git information
git describe --tags --always
git rev-parse --abbrev-ref HEAD
```

### Getting Help

```bash
# General help
gopro --help

# Command-specific help
gopro build --help
gopro build binary --help
gopro generate config --help

# Version information
gopro version
```

## Best Practices

### 1. Version Control

```bash
# Commit configuration
git add project.yaml

# Ignore generated files
echo "bin/" >> .gitignore
echo "dist/" >> .gitignore

# Version your environment templates
git add env/
```

### 2. Secret Management

`secret.env` files are **only a rendering source** for `FromSecretEnv`, which reads them directly from the source config directory. A file named `secret.env` is **never written to the output directory**, even when a `files` pattern matches it or `files` is absent. Releases up to v0.1.11 did copy it in those two cases, so keep it out of `files` regardless.

```yaml
# secret.env is NOT listed in files — it is read from source by FromSecretEnv
generate:
  configs:
    - name: api
      files:
        - "*.yaml"
```

```gitignore
# .gitignore — gopro init adds this entry
secret.env
```

```
# .dockerignore — gopro init adds this entry; the build context is the project root
**/secret.env
```

Place `secret.env` alongside the config templates it serves — GoPro reads it from
`<config_src>/<name>/secret.env`. To keep the expected keys discoverable without
committing values, check in a placeholder file next to it:

```bash
# env/default/config/api/secret.env.example  (committed)
DB_PASSWORD=YOUR_DB_PASSWORD
API_KEY=YOUR_API_KEY
```

### 3. Environment Organization

```
env/
├── default/          # Base templates
│   ├── config/
│   ├── kubernetes/
│   └── docker-compose/
├── local/           # Local overrides
│   └── config/
├── staging/         # Staging overrides
│   └── config/
└── prod/            # Production overrides
    ├── config/
    └── kubernetes/
```

### 4. Consistent Naming

Using one name per component across sections keeps the lists easy to check;
gopro itself rejects a listed name that nothing defines (see
[Component Names](#component-names)).

```yaml
# Use consistent names across sections
build:
  binaries:
    - name: api      # Same name everywhere
  images:
    - name: api

generate:
  configs:
    - name: api
  kubernetes:
    - name: api

default:
  binaries: [api]
  images: [api]
  configs: [api]
  kubernetes_templates: [api]
```

### 5. Documentation

Document your configuration:

```yaml
# project.yaml

# Product Configuration
product: myapp
version: v1.0.0

# Default environment for local development
default:
  # Build Go binaries with debug symbols
  binary_build_args:
    - -v
    - -gcflags
    - '-N -l'  # Disable optimizations for debugging
```

## Additional Resources

- [Project Repository](https://github.com/xhanio/gopro)
- [README](README.md) — feature overview and condensed reference
- [Example Configuration](example.project.yaml) (run `gopro example` to generate)
- [Claude Code plugin](plugins/gopro/README.md) — packages these conventions as a skill
- [Sprig Template Functions](https://masterminds.github.io/sprig/)
- [Go Templates Documentation](https://pkg.go.dev/text/template)
- [Framingo Package](https://github.com/xhanio/framingo)

## Support

For issues, questions, or contributions:
- [GitHub Issues](https://github.com/xhanio/gopro/issues)
- [Pull Requests](https://github.com/xhanio/gopro/pulls)

## License

MIT License - see [LICENSE](LICENSE) file for details.
