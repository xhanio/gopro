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
| `image_build_env` | `[]` | Docker build environment variables |
| `image_build_args` | `[]` | Docker build arguments |
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
| `platform` | No | Cross-compile targets: `["linux/amd64", "darwin/arm64"]` |
| `config_dir` | No | Config directory path used in templates |

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

When building images from Dockerfiles, these build args are automatically provided:

| Arg | Value |
|-----|-------|
| `NAME` | Image name |
| `BASE` | Base image |
| `CONFIG_TGT` | Config target directory |
| `CONFIG_DIR` | Component config directory |

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

Environment configs use `go.uber.org/config` for deep merging. Arrays are **replaced**, not appended:

```yaml
default:
  binary_build_env: [CGO_ENABLED=0, GOOS=linux]  # Both values

env:
  local:
    binary_build_env: [CGO_ENABLED=1]  # Replaces entire array
```

Use YAML anchors to extend:

```yaml
default:
  binary_build_args: &default_args
    - -v
    - -ldflags
    - '-s'

env:
  prod:
    binary_build_args:
      - *default_args
      - -ldflags
      - '-extldflags "-static"'
```

## Template Rendering Pipeline

1. Scan source directory for files matching `files` patterns
2. For files with `template.` prefix: render as Go template, strip prefix
3. For other files: copy as-is
4. Default layer renders first, then environment layer overlays
5. Templates use `[[` `]]` delimiters, receive `{Name, Project, Env}` context
