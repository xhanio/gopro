package types

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"go.uber.org/config"
	"golang.org/x/mod/modfile"
)

// ExampleProjectYAML holds the embedded example project.yaml, injected from main.
var ExampleProjectYAML []byte

type ResourceType string

var (
	ResourceTypeBinaries      = ResourceType("binaries")
	ResourceTypeImages        = ResourceType("images")
	ResourceTypeConfigs       = ResourceType("configs")
	ResourceTypeKubernetes    = ResourceType("kubernetes")
	ResourceTypeDockerCompose = ResourceType("docker-compose")
)

type Project struct {
	Product  string             `yaml:"product"`
	Model    string             `yaml:"model"`
	Version  string             `yaml:"version"`
	Domain   string             `yaml:"domain"`
	Module   string             `yaml:"module"`
	Default  EnvSpec            `yaml:"default"`
	Env      map[string]EnvSpec `yaml:"env"`
	Build    BuildSpec          `yaml:"build"`
	Generate GenerateSpec       `yaml:"generate"`
}

func (p *Project) Load(confPath string) error {
	provider, err := config.NewYAML(config.File(confPath))
	if err != nil {
		return err
	}
	err = provider.Get(config.Root).Populate(p)
	if err != nil {
		return err
	}
	if p.Module == "" {
		mb, err := os.ReadFile(filepath.Join(filepath.Dir(confPath), "go.mod"))
		if err != nil {
			return err
		}
		p.Module = modfile.ModulePath(mb)
	}
	return nil
}

type BuildSpec struct {
	Binaries []BinarySpec `yaml:"binaries"`
	Images   []ImageSpec  `yaml:"images"`
}

type BinarySpec struct {
	Name string `yaml:"name"`
	// Version is the application's own version, injected as
	// info.ApplicationVersion; unset, it inherits the product version.
	Version string `yaml:"version,omitempty"`
	Src     string `yaml:"src,omitempty"`
	// Deprecated: use Platforms, which also carries per-platform env and args.
	// Still honored, and folded into Platforms by GetPlatforms.
	Platform  []string       `yaml:"platform,omitempty"`
	Platforms []PlatformSpec `yaml:"platforms,omitempty"`
	BuildEnv  []string       `yaml:"build_env,omitempty"`
	BuildArgs []string       `yaml:"build_args,omitempty"`
	ConfigDir string         `yaml:"config_dir,omitempty"`
}

type PlatformSpec struct {
	Name string   `yaml:"name"`
	Env  []string `yaml:"env,omitempty"`
	Args []string `yaml:"args,omitempty"`
}

// GetPlatforms returns the platforms to build for, folding the plain platform
// list into the richer platforms spec. platform stays supported so existing
// configs keep working; a platform needing env or args moves to platforms.
//
// A name given in both keeps its first-seen position and takes the platforms
// entry, so the two can be mixed without duplicating a build.
func (b BinarySpec) GetPlatforms() []PlatformSpec {
	var result []PlatformSpec
	at := make(map[string]int, len(b.Platform)+len(b.Platforms))
	add := func(p PlatformSpec) {
		if i, ok := at[p.Name]; ok {
			result[i] = p
			return
		}
		at[p.Name] = len(result)
		result = append(result, p)
	}
	for _, name := range b.Platform {
		add(PlatformSpec{Name: name})
	}
	for _, p := range b.Platforms {
		add(p)
	}
	return result
}

type ImageSpec struct {
	Name      string `yaml:"name"`
	Base      string `yaml:"base,omitempty"`
	BuildSrc  string `yaml:"build_src,omitempty"`
	BuildFrom string `yaml:"build_from,omitempty"`
	Prefix    string `yaml:"prefix,omitempty"`
	Repo      string `yaml:"repo,omitempty"`
	Tag       string `yaml:"tag,omitempty"`
	NoPush    bool   `yaml:"no_push,omitempty"`
}

// GetImageNameWithTag resolves the fully-qualified image reference for an
// explicit tag, applying the same repo/prefix resolution as GetImageName.
func (i ImageSpec) GetImageNameWithTag(env EnvSpec, tag string) string {
	// image repo defined in build section
	repo := i.Repo
	if repo == "" {
		// image repo undefined, use image name instead
		repo = i.Name
	}
	// image prefix defined in build section
	prefix := i.Prefix
	if prefix == "" {
		prefix = env.ImagePrefix
	}
	if prefix != "" {
		return fmt.Sprintf("%s:%s", path.Join(prefix, repo), tag)
	}
	return fmt.Sprintf("%s:%s", repo, tag)
}

func (i ImageSpec) GetImageName(env EnvSpec) string {
	// image tag defined in build section
	tag := i.Tag
	if tag == "" {
		tag = env.ImageTag
	}
	if tag == "" {
		tag = "latest"
	}
	return i.GetImageNameWithTag(env, tag)
}

type GenerateSpec struct {
	Configs       []ConfigSpec      `yaml:"configs"`
	Kubernetes    []KubernetesSpec  `yaml:"kubernetes"`
	DockerCompose DockerComposeSpec `yaml:"docker_compose"`
}

type ConfigSpec struct {
	Name  string   `yaml:"name"`
	Src   string   `yaml:"src,omitempty"`
	Files []string `yaml:"files,omitempty"`
}

type KubernetesSpec struct {
	Name  string   `yaml:"name"`
	Src   string   `yaml:"src,omitempty"`
	Files []string `yaml:"files,omitempty"`
}

type DockerComposeSpec struct {
	Src   string   `yaml:"src,omitempty"`
	Files []string `yaml:"files,omitempty"`
}
