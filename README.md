# GoPro

A Go-based project generator and build tool that helps manage Go projects with configurable environments. GoPro provides commands for building binaries, Docker images, and generating configuration/Kubernetes templates from environment-specific configurations.

## Installation

Install directly from source:

```bash
go install github.com/xhanio/gopro@latest
```

Or build the project locally:

```bash
go build -o gopro main.go
```

## Configuration

GoPro uses a YAML-based configuration system (`project.yaml`) with:

- **Default environment settings**: Base configuration for all environments
- **Environment-specific overrides**: Local, prod, or custom environments
- **Build definitions**: Binary and image build specifications
- **Generate definitions**: Configuration and Kubernetes template specifications

The configuration system supports:
- Multi-environment builds (local with CGO, prod static binaries)
- Docker image building from Dockerfiles or base images
- Template rendering with Go templating and Sprig functions
- Git-based version information injection

## Usage

### Initialize a new project

```bash
./gopro init
```

### Build Commands

Build binaries:
```bash
./gopro build binary                   # Build for default environment
./gopro build binary -e local          # Build for local environment
./gopro build binary -e prod           # Build for production environment
```

Build Docker images:
```bash
./gopro build image                    # Build images
./gopro build image --push             # Build and push images
```

### Generate Commands

Generate configuration files:
```bash
./gopro generate config -e local       # Generate config for local env
./gopro generate kubernetes -e prod    # Generate k8s templates for prod
```

### Global Options

- `-c, --config`: Specify configuration file path (default: `project.yaml`)
- `-f, --filter`: Filter components using regex pattern
- `-e, --environment`: Specify target environment

## Architecture

The project follows a modular CLI architecture using Cobra:

- **main.go**: Entry point that initializes and executes the root command
- **pkg/components/cmd/**: Contains all CLI command implementations
- **pkg/types/**: Configuration data structures and loading logic

## Dependencies

- [Cobra](https://github.com/spf13/cobra): CLI framework
- [Sprig](https://github.com/Masterminds/sprig): Template functions
- [go-gitignore](https://github.com/sabhiram/go-gitignore): .gitignore parsing
- [framingo](https://github.com/xhanio/framingo): Build information utilities
- [uber-go/config](https://github.com/uber-go/config): Configuration management