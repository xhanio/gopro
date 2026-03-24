# GoPro

A Go-based project generator and build tool for managing Go projects with multi-environment configurations. GoPro streamlines building binaries, Docker images, and generating configuration/Kubernetes templates with environment-specific overrides.

## Features

- **Multi-environment builds**: Separate configurations for local, production, and custom environments
- **Cross-platform compilation**: Build Go binaries for multiple OS/architecture combinations
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

## Quick Start

Generate an example configuration file:

```bash
gopro example
```

This saves an example `project.yaml` to the current directory.

Then initialize the project structure:

```bash
gopro init
```

## Configuration

GoPro uses a YAML-based configuration file (`project.yaml`) with the following structure:

- **product**: Product name (used for environment variable prefixes)
- **default**: Base configuration shared across all environments
- **env**: Environment-specific overrides (local, prod, custom)
- **build**: Binary and Docker image build specifications
- **generate**: Template generation specifications for configs and Kubernetes manifests

### Configuration Structure

Run `gopro example` to generate a complete example, or see the embedded [example.project.yaml](example.project.yaml). Key sections:

```yaml
product: myapp

default:
  binary_src: build/binary      # Source directory for binaries
  binary_tgt: bin/               # Output directory for binaries
  binary_build_env:              # Environment variables for all builds
    - CGO_ENABLED=0
  binary_build_args:             # Go build arguments
    - -v
    - -ldflags
    - '-s -w'

  image_build_src: build/image   # Dockerfile source directory
  image_prefix: myregistry.io    # Default image registry

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
      platform: [linux/amd64, darwin/arm64]
      config_dir: /etc/myapp          # Config directory in container

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
      files: ["*.yaml", "*.json", "secret.env"]  # File patterns to process

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
- `-e, --environment <name>`: Target environment (default, local, prod, or custom)
- `-f, --filter <regex>`: Filter components using regex pattern
- `-v, --verbose`: Enable verbose output for debugging

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
  - Platform-specific binaries named as `{name}_{GOOS}_{GOARCH}` (e.g., `api_linux_amd64`)
  - Configure platforms via `platform` array (e.g., `[linux/amd64, darwin/arm64]`)

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
```

Additional flags:
- `-p, --push`: Push images to registry after building

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
  - Skip specific images with `no_push: true`
  - Only pushes successfully built images

### Generate Commands

#### Generate Configurations

```bash
gopro generate config -e local         # Generate config for local environment
gopro generate config -o ./output      # Custom output directory
gopro generate config -f "api.*"       # Generate specific configs
```

Additional flags:
- `-o, --output <path>`: Specify custom output directory
- `-x, --prefix <prefix>`: Template file prefix (default: `template.`)

#### Generate Kubernetes Manifests

```bash
gopro generate kubernetes -e prod      # Generate k8s manifests for prod
gopro generate kubernetes -t ./k8s     # Custom output directory
gopro generate kubernetes -f "api.*"   # Generate specific manifests
```

Additional flags:
- `-t, --output <path>`: Specify custom output directory

#### Generate Docker Compose

```bash
gopro generate docker-compose -e local # Generate docker-compose.yaml
```

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
  - **`GetImageName`**: Get fully qualified image name
  - **`FromFile`**: Read file content from any path
  - **`FromConfigFile`**: Read from generated config files
  - **`FromConfigJSON`**: Extract JSON values via JSONPath
  - **`FromSecretEnv`**: Read key-value pairs from `secret.env`

- **Template context** (available in templates):
  ```go
  .Name    // Component name being generated
  .Project // Full project configuration
  .Env     // Current environment configuration
  ```

- **File filtering**:
  - Use `files` array with glob patterns in configuration
  - Example: `["*.yaml", "*.json"]` to process only specific file types
  - Useful for excluding sensitive files or controlling what gets generated

## Examples

### Multi-Environment Binary Build

```yaml
# project.yaml
build:
  binaries:
    - name: api
      src: cmd/api
      platform: [linux/amd64, darwin/arm64, windows/amd64]

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
    config_tgt: /etc/myapp

# configs/api/template.config.yaml
deployment:
  image: [[ GetImageName "api" ]]
  config_path: [[ .Env.ConfigTgt ]]

service:
  name: [[ .Name ]]
  version: [[ .Project.Version ]]
```

```bash
gopro generate config -e prod
# Outputs: configs_generated/api/config.yaml with evaluated template values
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

- **[main.go](main.go)**: Entry point that initializes and executes the root command
- **[pkg/components/cmd/](pkg/components/cmd/)**: All CLI command implementations
  - `root.go`: Root command with global flags
  - `build.go`: Binary and image build commands
  - `generate.go`: Config, Kubernetes, and Docker Compose generation commands
  - `example.go`: Example configuration file generation command (uses `example.project.yaml` from project root via `types.ExampleProjectYAML`)
  - `util_*.go`: Utility functions for execution, rendering, and printing
- **[pkg/types/](pkg/types/)**: Configuration data structures and loading logic
  - `project.go`: Project, build, and generate structures
  - `env.go`: EnvSpec with environment merging

## Dependencies

Key dependencies:
- **[Cobra](https://github.com/spf13/cobra)**: CLI framework for command structure
- **[Sprig](https://github.com/Masterminds/sprig)**: Template function library
- **[framingo](https://github.com/xhanio/framingo)**: Build information and utilities
- **[uber-go/config](https://github.com/uber-go/config)**: Configuration merging and environment overlays
- **[gjson](https://github.com/tidwall/gjson)**: JSON path queries in templates
- **[go-gitignore](https://github.com/sabhiram/go-gitignore)**: .gitignore parsing

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.

Copyright 2025 Xi Han