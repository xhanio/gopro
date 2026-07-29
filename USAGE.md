# GoPro Usage Guide

This guide provides detailed instructions on using GoPro, a Go-based project generator and build tool for managing multi-environment Go projects.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Global Flags](#global-flags)
- [Commands](#commands)
  - [example](#example-command)
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
go build -o gopro main.go
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

This saves an example configuration file to the current directory. Edit it to match your project.

### 2. Initialize Project Structure

```bash
gopro init
```

This command will:
- Initialize Git repository (if not already initialized)
- Initialize Go module (if go.mod doesn't exist)
- Create directory structure based on configuration
- Create/update `.gitignore` file

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
| `--environment` | `-e` | (unset) | Target environment (local, prod, or custom). When unset, the `default` section is used as-is |
| `--filter` | `-f` | `.*` | Regex filter for selecting components |
| `--verbose` | `-v` | `false` | Enable verbose output for debugging |
| `--help` | | `false` | Show help information |

The `example` and `version` commands do not load `project.yaml`, so they work in
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

### init Command

Initialize project structure based on configuration.

```bash
gopro init
gopro init -e local        # Initialize only for local environment
gopro init -c custom.yaml  # Use custom config file
```

#### What it Does

1. **Git Repository**: Initializes Git if not already initialized
2. **Go Module**: Runs `go mod init` if `go.mod` doesn't exist, using the `module` field from config when it is set
3. **Directory Structure**: Creates all directories specified in:
   - Default environment configuration
   - All environment-specific configurations (or specific environment with `-e`)
4. **Gitignore**: Creates or updates `.gitignore` with:
   - `bin/` - Binary output directory
   - `dist/` - Generated files directory
   - `test/` - Test directory
   - `secret.env` - Secret environment files (rendering source only, never copied to output)

#### Example Output

```
initializing project directories
- initializing git repository
- initializing go module
- create directories build/binary/api for environment default successfully
- create directories env/local/config/api for environment local successfully
- managing .gitignore file
- added to .gitignore: bin/
- added to .gitignore: dist/
```

### version Command

Print version information.

```bash
gopro version
```

Outputs the Git tag (from `-ldflags` injection at build time, falling back to
`git describe --tags --always` in the current directory) and the current time:

```
Version:  v0.1.9
Build Time: 2026-07-28 21:53:27.884189693 -0700 PDT m=+0.004790545
```

Note that "Build Time" is the time the command ran, not the time the binary was
compiled; the compile-time metadata lives in the `-ldflags`-injected fields
described under [Build Metadata Injection](#build-metadata-injection).

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
  inherits the rest. Above, every build keeps `CGO_ENABLED=1`, while `FOO`
  resolves to `from_binary` for most targets and the `linux/arm64` build
  additionally gets `CC`.
- **Arguments** (`binary_build_args` → `build_args` → platform `args`)
  **replace** wholesale. Go build flags are positional and repeatable, so a
  key-wise merge cannot distinguish an override from an accumulation. Only an
  unset level inherits — the empty `build_args: []` above means the host and
  `linux/amd64` builds run with no arguments at all, while `linux/arm64`
  restores `-v -tags=netgo`.

Note that this is a different mechanism from the `default` → `env` merge in
`project.yaml`, where arrays are replaced rather than merged. See
[Configuration Merging](#configuration-merging).

#### Build Metadata Injection

GoPro automatically injects build metadata using the [framingo](https://github.com/xhanio/framingo) package. The following information is embedded:

- **Git Information**: Branch, tag, commit hash
- **Build Information**: Build time, version, type
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
docker build -t <image> --no-cache \
  --build-arg NAME=... --build-arg BASE=... \
  --build-arg CONFIG_TGT=... --build-arg CONFIG_DIR=... \
  -f <project_root>/<build_src>/Dockerfile <project_root>
```

The build context is always the project root, so a Dockerfile can `COPY` from
anywhere in the repository. `image_build_env` is applied as the environment for
the `docker build` and `docker tag` invocations.

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

1. **Clean Target**: The config's target directory is removed before rendering,
   so output is always a clean reflection of the sources — never a mix with
   files left over from a previous run

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
   - An empty or absent `files` list processes everything
   - Useful for excluding sensitive or irrelevant files

#### Configuration Example

```yaml
generate:
  configs:
    - name: api
      files:
        - "*.yaml"
        - "*.json"
        - "config/*"
        # Do NOT include secret.env here — it is read from source by FromSecretEnv, never copied to output

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

As with configs, each template's target directory is removed before rendering,
and the default layer renders first with the environment layer overlaid on top.

#### Configuration Example

```yaml
generate:
  kubernetes:
    - name: api
      files:
        - "deployment.yaml"
        - "service.yaml"
        - "*.yaml"

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
It also does not clear the target directory first, since it renders into a shared
directory rather than one named after a component.

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
model: standard                # Product model (optional)
version: v1.0.0                # Product version (optional; falls back to the Git tag)
domain: example.com            # Domain name (optional)
module: github.com/user/myapp  # Go module name (optional; read from go.mod when unset)
```

The field is `module`, not `project`. When it is omitted, GoPro reads the module
path from the `go.mod` sitting next to `project.yaml`.

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
      no_push: false                  # Optional: skip pushing

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

Extract JSON values using JSONPath:

```yaml
# Reads from dist/config/api/config.json and extracts .database.host
host: [[ FromConfigJSON "api" "config.json" "database.host" ]]
```

#### FromSecretEnv

Read key-value pairs from secret.env (reads directly from the source config directory, not from dist/):

```yaml
# Reads from env/<environment>/config/api/secret.env (source directory)
password: [[ FromSecretEnv "api" "DB_PASSWORD" ]]
```

Format of `secret.env`:
```bash
DB_PASSWORD=supersecret
API_KEY=abc123
```

### Sprig Functions

All [Sprig v3](http://masterminds.github.io/sprig/) functions are available:

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
gopro build binary -f "(api|worker)"

# Build everything except test components
gopro build binary -f "^(?!.*test).*$"
```

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

There is no configuration hook for adding further `--build-arg` values. To
influence the build in other ways, set `image_build_env`, which becomes the
environment for the `docker build` and `docker tag` processes:

```yaml
env:
  prod:
    image_build_env:
      - DOCKER_BUILDKIT=1
```

Builds always run with `--no-cache`.

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

# 6. Apply to cluster
kubectl apply -f dist/prod/kubernetes/
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
zip cli-windows-amd64.zip cli_windows_amd64.exe
```

### Staging Environment Workflow

```yaml
# project.yaml
env:
  staging:
    image_prefix: staging-registry.io/myapp
    image_tag: staging-latest
    config_src: env/staging/config
    kubernetes_src: env/staging/kubernetes
```

```bash
# Deploy to staging
gopro build image -e staging --push
gopro generate config -e staging
gopro generate kubernetes -e staging
kubectl apply -f dist/staging/kubernetes/ --namespace=staging
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

#### 3. Git Repository Required

```
Error: failed to get git information
```

**Solution**: Initialize Git repository:
```bash
git init
git add .
git commit -m "Initial commit"
```

Or run `gopro init` which does this automatically.

#### 4. Template Rendering Error

```
Error: failed to render from secret.env: no such file or directory
```

**Solution**: Ensure `secret.env` exists in the source config directory (e.g., `env/default/config/api/secret.env`). `FromSecretEnv` reads directly from the source directory, not from `dist/`. Also ensure configs are generated before kubernetes if k8s templates reference config files:
```bash
gopro generate config -e prod
gopro generate kubernetes -e prod
```

#### 5. Docker Build Failures

```
Error: failed to build image: Dockerfile not found
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

```
Error: C compiler not found (CGO)
```

**Solution**: Disable CGO for cross-compilation:
```yaml
env:
  prod:
    binary_build_env:
      - CGO_ENABLED=0
```

#### 7. Product Name Missing

```
Error: product name is required in project.yaml configuration
```

**Solution**: `gopro init` requires a `product` at the top of `project.yaml`:
```yaml
product: myapp
```

#### 8. Module Path Not Found

If the module path cannot be resolved, set it explicitly:
```yaml
module: github.com/user/myapp
```

Or initialize the Go module so it can be read from `go.mod`:
```bash
go mod init github.com/user/myapp
```

The key is `module`, not `project`.

### Debug Mode

Enable verbose output for debugging:

```bash
gopro build binary -v
gopro generate config -v -e prod
```

This shows:
- Executed commands with arguments
- Environment variables
- File operations
- Template rendering details

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

`secret.env` files are used **only as a rendering source** for `FromSecretEnv` — they are read directly from the source config directory and are **never copied to the output directory**. Do NOT list `secret.env` in the `files` patterns.

```yaml
# secret.env is NOT listed in files — it is read from source by FromSecretEnv
generate:
  configs:
    - name: api
      files:
        - "*.yaml"
```

```bash
# .gitignore (gopro init adds this automatically)
secret.env  # Don't commit actual secrets
```

Place `secret.env` alongside the config templates it serves — GoPro reads it from
`<config_src>/<name>/secret.env`. To keep the expected keys discoverable without
committing values, check in a placeholder file next to it:

```bash
# env/default/config/api/secret.env.example  (committed)
DB_PASSWORD="YOUR_DB_PASSWORD"
API_KEY="YOUR_API_KEY"
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
- [Sprig Template Functions](http://masterminds.github.io/sprig/)
- [Go Templates Documentation](https://pkg.go.dev/text/template)
- [Framingo Package](https://github.com/xhanio/framingo)

## Support

For issues, questions, or contributions:
- [GitHub Issues](https://github.com/xhanio/gopro/issues)
- [Pull Requests](https://github.com/xhanio/gopro/pulls)

## License

MIT License - see [LICENSE](LICENSE) file for details.
