# GoPro

A Go-based project generator and build tool for managing Go projects with multi-environment configurations. GoPro streamlines building binaries, Docker images, and generating configuration/Kubernetes templates with environment-specific overrides.

## Features

- **Multi-environment builds**: Separate configurations for local, production, and custom environments
- **Cross-platform compilation**: Build Go binaries for multiple OS/architecture combinations, each with its own build environment and flags
- **Docker image management**: Build from Dockerfiles or third-party images with automatic tagging
- **Template-based generation**: Generate configs, Kubernetes manifests, and Docker Compose files using Go templates with Sprig functions
- **Build metadata injection**: Automatically inject Git version info and build metadata into binaries
- **Flexible filtering**: Use regex patterns to selectively build/generate specific components

## Installation

Install directly from source:

```bash
go install github.com/xhanio/gopro@latest
```

Or build locally:

```bash
git clone https://github.com/xhanio/gopro.git
cd gopro
go build -o gopro main.go
```

### Claude Code plugin

GoPro's conventions are packaged as a Claude Code skill, so an agent writes
`project.yaml` and drives the build correctly without being walked through
them:

```
/plugin marketplace add https://github.com/xhanio/plugins
/plugin install gopro@xhanio
```

It activates on its own whenever a session touches `project.yaml` or a GoPro
build. See [`plugins/gopro/README.md`](plugins/gopro/README.md).

## Quick Start

Generate an example configuration file:

```bash
gopro example
```

This saves an example `project.yaml` to the current directory. It refuses to
overwrite an existing one.

Then initialize the project structure:

```bash
gopro init
```

This creates the source and target directories declared in `project.yaml`,
initializes a Git repository and Go module if they are missing, and adds
`bin/`, `dist/`, `test/`, and `secret.env` to `.gitignore`.

## Configuration

GoPro uses a YAML-based configuration file (`project.yaml`) with the following structure:

- **product**: Product name, required (also used as the environment variable prefix)
- **model**, **version**, **domain**, **module**: Optional project metadata; `module` is read from `go.mod` when unset, and `version` falls back to the current Git tag. `model` is currently parsed but never read — only `gopro build binary --product-model` sets the injected `ProductModel`
- **default**: Base configuration shared across all environments
- **env**: Environment-specific overrides (local, prod, custom)
- **build**: Binary and Docker image build specifications
- **generate**: Template generation specifications for configs, Kubernetes manifests, and Docker Compose files

The `default`/`env` sections say *where* things live and *which* components are
active (`binaries`, `images`, `configs`, `kubernetes_templates`); the
`build`/`generate` sections describe *how* each named component is built or
rendered. A component is only acted on when its name appears in both.

When an environment overrides a `default` value, the merge is per key, not per
element: **arrays are replaced wholesale**. An `env.local.binary_build_env` of
`[CGO_ENABLED=1]` drops every other variable `default` had set, so anything still
needed must be restated. This is deliberately unlike the per-binary and
per-platform build layers described under
[Build Binaries](#build-binaries), which merge environment variables key-wise.

### Configuration Structure

Run `gopro example` to generate a complete example, or see the embedded
[example.project.yaml](example.project.yaml). For the full field-by-field
reference, see the [Usage Guide](USAGE.md). Key sections:

```yaml
product: myapp                   # Required; also the environment variable prefix
module: github.com/me/myapp      # Optional; read from go.mod when unset

default:
  binary_src: build/binary       # Source directory for binaries
  binary_tgt: bin/               # Output directory for binaries
  binary_build_env:              # Environment variables for all builds
    - CGO_ENABLED=0
  binary_build_args:             # Go build arguments
    - -v
    - -ldflags
    - '-s -w'

  image_build_src: build/image   # Dockerfile source directory
  image_prefix: myregistry.io    # Default image registry
  image_tag: v1.0.0              # Default image tag (defaults to "latest")
  image_build_env:               # Environment for docker build/tag
    - DOCKER_BUILDKIT=1

  config_src: env/default/config
  config_tgt: dist/config
  kubernetes_src: env/default/kubernetes
  kubernetes_tgt: dist/kubernetes
  docker_compose_src: env/default/docker-compose
  docker_compose_tgt: dist

  binaries: [api, worker]        # Binaries to build by default
  images: [api, worker]          # Images to build by default
  configs: [api, worker]         # Configs to generate
  kubernetes_templates: [api]    # K8s templates to generate

env:
  local:
    binary_build_env: [CGO_ENABLED=1]
    config_src: env/local/config
    config_tgt: dist/local/config

  prod:
    binary_build_env: [CGO_ENABLED=0, GOOS=linux, GOARCH=amd64]
    binary_build_args: ["-ldflags=-s -w -extldflags '-static'"]
    image_prefix: prod-registry.io/myapp
    image_tag: v1.0.0

build:
  binaries:
    - name: api
      src: cmd/api                    # Optional: override source path
      config_dir: /etc/myapp          # Config directory in container
      build_env: [CGO_ENABLED=0]      # Optional: merged over binary_build_env
      build_args: [-v]                # Optional: replaces binary_build_args
      platforms:
        - name: linux/amd64
        - name: linux/arm64
          env: [CC=aarch64-linux-gnu-gcc]  # Merged over build_env
          args: [-v, -tags=netgo]          # Replaces build_args

  images:
    - name: base
      build_from: golang:1.21-alpine  # Pull and re-tag existing image
    - name: api
      base: $base                     # Reference to another image
      build_src: docker/api           # Optional: override Dockerfile path
      prefix: custom-registry.io      # Optional: override image prefix
      repo: my-api                    # Optional: override repo name
      tag: latest                     # Optional: override tag
      no_push: true                   # Skip pushing this image

generate:
  configs:
    - name: api
      files: ["*.yaml", "*.json"]  # File patterns to process (do NOT include secret.env — it is read from source by FromSecretEnv)

  kubernetes:
    - name: api
      files: ["deployment.yaml", "service.yaml"]

  docker_compose:
    files: ["docker-compose.yaml"]
```

## Usage

### Global Options

All commands support these global flags:

- `-c, --config <path>`: Specify configuration file path (default: `project.yaml`)
- `-e, --environment <name>`: Target environment (`local`, `prod`, or custom). When omitted, the `default` section is used as-is; a named environment is merged on top of `default`
- `-f, --filter <regex>`: Filter components using regex pattern (default: `.*`)
- `-v, --verbose`: Enable verbose output for debugging

### Project Commands

```bash
gopro example                          # Write an example project.yaml to the current directory
gopro init                             # Create project directories, git repo, go module, .gitignore
gopro init -e prod                     # Only create directories for the prod environment
gopro version                          # Print version and build time
```

`gopro init` creates directories for `default` plus every environment in
`project.yaml`; passing `-e` limits it to `default` plus that one environment.

### Build Commands

#### Build Binaries

```bash
gopro build binary                     # Build for default environment
gopro build binary -e local            # Build for local environment
gopro build binary -e prod             # Build for production environment
gopro build binary -f "api.*"          # Build only binaries matching the filter
gopro build binary -o ./dist           # Specify custom output directory
```

Additional flags:
- `--product-model <value>`: Override product model metadata
- `--product-version <value>`: Override product version metadata
- `--build-version <value>`: Override build version metadata
- `--build-type <value>`: Override build type metadata
- `--build-date <value>`: Override build date metadata

**Features:**

- **Cross-platform compilation**: Build for multiple OS/arch combinations
  - A host binary named `{name}` is always built first, with no `GOOS`/`GOARCH` pinned
  - Each entry in `platforms` then produces `{name}_{GOOS}_{GOARCH}` (e.g., `api_linux_amd64`)
  - `GOOS`/`GOARCH` derived from the platform name outrank anything set in the build environment
  - The older `platform: [linux/amd64, darwin/arm64]` shorthand is deprecated but still honored, and is folded into `platforms`. A name given in both lists keeps its first-seen position and takes the `platforms` entry, so the two can be mixed without building twice

- **Layered build environment and flags**: Three levels, resolved differently
  - `binary_build_env` → binary `build_env` → platform `env` are **merged** key-wise: each level overrides only the variables it names and inherits the rest
  - `binary_build_args` → binary `build_args` → platform `args` **replace** wholesale: the most specific level that sets a list wins, because Go build flags are positional and repeatable, so a key-wise merge cannot tell an override from an accumulation. Only an unset level inherits — an explicitly empty list means "build with no arguments"

- **Build metadata injection**: Automatic version info embedding via `-ldflags`
  - Product model, version, and build type
  - Git-based build version (tags/commits)
  - Build date and custom metadata
  - Override with command-line flags if needed

- **Environment-specific builds**: Customize per environment
  - Apply environment variables (e.g., `CGO_ENABLED=0` for static builds)
  - Add custom build arguments (e.g., `-ldflags=-s -w` for smaller binaries)
  - Configure source and output paths

#### Build Docker Images

```bash
gopro build image                      # Build images
gopro build image --push               # Build and push images
gopro build image -p -f "api.*"        # Build and push specific images
gopro build image -p -l                # Build, push, and also tag/push :latest
```

Additional flags:
- `-p, --push`: Push images to registry after building
- `-l, --latest`: Additionally tag and push the image as `:latest` (requires `--push`; warns and does nothing without it)

**Features:**

- **Two build methods**:
  1. **From Dockerfile**: Build from source with automatic build args injection
     - `NAME`: Image name from configuration
     - `BASE`: Base image (supports `$image_name` for cross-references)
     - `CONFIG_TGT`: Configuration target directory
     - `CONFIG_DIR`: Component-specific config directory
  2. **From third-party images**: Pull and re-tag existing images
     - Use `build_from` field to specify source image
     - Ideal for using pre-built base images with custom tags

- **Flexible image naming**: `[prefix/]repo:tag`
  - `prefix`: Registry prefix (from config or environment)
  - `repo`: Repository name (defaults to image name)
  - `tag`: Image tag (defaults to "latest")
  - Example: `myregistry.io/myapp:v1.0.0`

- **Smart push behavior**:
  - Use `--push` flag to push after building
  - Add `--latest` to also tag and push `:latest` (skipped when the image already builds as `:latest`)
  - Skip specific images with `no_push: true`
  - Only pushes successfully built images

- **Build execution**: Images build with `--no-cache`, using `{build_src}/Dockerfile` with the project root as build context, and inherit `image_build_env` as the Docker environment

### Generate Commands

#### Generate Configurations

```bash
gopro generate config -e local         # Generate config for local environment
gopro generate config -o ./output      # Custom output directory
gopro generate config -f "api.*"       # Generate specific configs
```

Additional flags:
- `-o, --output <path>`: Specify custom output directory (defaults to `config_tgt`, then to `config_src` for an in-place render)
- `-x, --prefix <prefix>`: Template file prefix (default: `template.`) — available on all `generate` subcommands

Each config's target directory is removed before rendering, so generated output
is a clean reflection of the sources — except for an in-place render, where the
target is a template source and is left alone.

#### Generate Kubernetes Manifests

```bash
gopro generate kubernetes -e prod      # Generate k8s manifests for prod
gopro generate kubernetes -t ./k8s     # Custom output directory
gopro generate kubernetes -f "api.*"   # Generate specific manifests
```

Additional flags:
- `-t, --output <path>`: Specify custom output directory (defaults to `kubernetes_tgt`, then to `kubernetes_src` for an in-place render)

As with configs, each template's target directory is removed before rendering,
unless the render is in place.

#### Generate Docker Compose

```bash
gopro generate docker-compose -e local # Generate docker-compose.yaml
```

Output goes to `docker_compose_tgt` (or the current directory when unset). Unlike
config and Kubernetes generation, this command has no output flag and does not
clear the target directory.

**Template Rendering Features:**

- **Two-layer template system**:
  - Renders default templates first as base layer
  - Applies environment-specific overlays on top
  - Enables incremental overrides without duplicating entire configs

- **Template processing**:
  - Files with `template.` prefix (customizable) are processed as Go templates
  - Template delimiters: `[[` and `]]` (avoids conflicts with `{{ }}`)
  - Template prefix removed in output: `template.config.yaml` → `config.yaml`
  - Non-template files copied directly without processing

- **Rich template functions**:
  - **Sprig v3**: Complete Sprig library (strings, dates, encoding, etc.)
  - **`GetEnvKey`**: Generate environment variable names with product prefix
  - **`GetConfigDir`**: Get config directory path for a binary
  - **`GetImageName`**: Get fully qualified image name for a configured image
  - **`GetImageNameWithTag`**: Same resolution, with an explicit tag — `GetImageNameWithTag "api" "latest"`
  - **`FromFile`**: Read file content from any path
  - **`FromConfigFile`**: Read from generated config files
  - **`FromConfigJSON`**: Extract JSON values via JSONPath
  - **`FromSecretEnv`**: Read key-value pairs from `secret.env`

- **Template context** (available in templates):
  ```go
  .Name    // Component name being generated
  .Project // Full project configuration
  .EnvName // Selected environment name ("" when -e was not given)
  .Env     // Current environment configuration (default merged with the selected env)
  ```

- **File filtering**:
  - Use `files` array with glob patterns in configuration
  - Example: `["*.yaml", "*.json"]` to process only specific file types
  - A pattern without a `/` matches by file name at any depth, so `*.yaml` also selects `sub/config.yaml`; a pattern containing one is matched against the whole relative path, so `cert/*` stays scoped to `cert/`
  - Useful for excluding sensitive files or controlling what gets generated

- **Directory structure preserved**: A template keeps the subdirectory it was
  authored in — `sub/template.config.yaml` renders to `sub/config.yaml`, not to
  the output root

- **In-place rendering**: With `config_tgt` (or `kubernetes_tgt`) unset, output
  lands beside the templates it came from — `template.config.yaml` renders to
  `config.yaml` in the same directory. A distinct target is cleared before each
  render; an in-place one is not, since its inputs and outputs share a
  directory and clearing it would delete the templates

## Examples

### Multi-Environment Binary Build

```yaml
# project.yaml
build:
  binaries:
    - name: api
      src: cmd/api
      platforms:
        - name: linux/amd64
        - name: darwin/arm64
        - name: windows/amd64

env:
  local:
    binaries: [api]
    binary_build_env: [CGO_ENABLED=1]
  prod:
    binaries: [api]
    binary_build_env: [CGO_ENABLED=0]
    binary_build_args: ["-ldflags=-s -w -extldflags '-static'"]
```

```bash
# Build with CGO for local development
gopro build binary -e local

# Build static binaries for production
gopro build binary -e prod
```

### Template-Based Config Generation

```yaml
# project.yaml
env:
  prod:
    image_prefix: myregistry.io/myapp
    image_tag: v1.0.0
    config_src: env/prod/config
    config_tgt: dist/prod/config
```

```yaml
# env/prod/config/api/template.config.yaml
deployment:
  image: [[ GetImageName "api" ]]
  rollback_image: [[ GetImageNameWithTag "api" "stable" ]]
  config_path: [[ .Env.ConfigTgt ]]

service:
  name: [[ .Name ]]
  environment: [[ .EnvName ]]
  version: [[ .Project.Version ]]
```

```bash
gopro generate config -e prod
# Outputs: dist/prod/config/api/config.yaml with evaluated template values
```

### Docker Image with Cross-References

```yaml
# project.yaml
build:
  images:
    - name: base
      build_from: golang:1.21-alpine
    - name: api
      base: $base  # References the "base" image
      build_src: docker/api
```

```bash
gopro build image -e prod --push
```

## Architecture

The project follows a modular CLI architecture using Cobra:

- **[main.go](main.go)**: Entry point that embeds `example.project.yaml`, then initializes and executes the root command
- **[pkg/components/cmd/](pkg/components/cmd/)**: All CLI command implementations
  - `root.go`: Root command with global flags and the pre-run that loads config and collects Git metadata
  - `init.go`: Project scaffolding command (directories, git, go module, `.gitignore`)
  - `build.go`: Binary and image build commands
  - `generate.go`: Config, Kubernetes, and Docker Compose generation commands
  - `example.go`: Example configuration file generation command (uses `example.project.yaml` from project root via `types.ExampleProjectYAML`)
  - `version.go`: Version information command
  - `util_config.go`: Project/environment loading and shared state
  - `util_*.go`: Utility functions for execution, rendering, and printing
- **[pkg/types/](pkg/types/)**: Configuration data structures and loading logic
  - `project.go`: Project, build, and generate structures, plus image name resolution
  - `env.go`: EnvSpec with environment merging
- **[plugins/gopro/](plugins/gopro/)**: The Claude Code plugin packaging the `gopro` skill

## Dependencies

Key dependencies:
- **[Cobra](https://github.com/spf13/cobra)**: CLI framework for command structure
- **[Sprig](https://github.com/Masterminds/sprig)**: Template function library
- **[framingo](https://github.com/xhanio/framingo)**: Build information and utilities
- **[uber-go/config](https://github.com/uber-go/config)**: Configuration merging and environment overlays
- **[gjson](https://github.com/tidwall/gjson)**: JSON path queries in templates
- **[go-gitignore](https://github.com/monochromegane/go-gitignore)**: .gitignore parsing
- **[golang.org/x/mod](https://pkg.go.dev/golang.org/x/mod)**: `go.mod` parsing to derive the module path
- **[color](https://github.com/fatih/color)**: Colored terminal output

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - see [LICENSE](LICENSE) file for details.

Copyright (c) 2025 Xi Han