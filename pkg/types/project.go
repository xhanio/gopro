package types

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"go.uber.org/config"
	"golang.org/x/mod/modfile"

	"github.com/xhanio/framingo/pkg/utils/envutil"
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
	Name      string   `yaml:"name"`
	Src       string   `yaml:"src,omitempty"`
	Platform  []string `yaml:"platform,omitempty"`
	BuildEnv  []string `yaml:"build_env,omitempty"`
	BuildArgs []string `yaml:"build_args,omitempty"`
	ConfigDir string   `yaml:"config_dir,omitempty"`
}

// GetBuildEnv layers a binary's build_env over the environment's
// binary_build_env, keyed on the variable name, so a binary overrides only the
// variables it names and inherits the rest.
func (b BinarySpec) GetBuildEnv(env EnvSpec) []string {
	return envutil.Merge(env.BinaryBuildEnv, b.BuildEnv)
}

// GetBuildArgs returns a binary's build_args, falling back to the
// environment's binary_build_args. Unlike build_env these replace rather than
// merge: go build flags are positional and repeatable, so a key-wise merge
// cannot tell an override from an accumulation.
//
// Only an unset build_args inherits. Setting it to an empty list is a way to
// build with no arguments at all, so the test is nil rather than empty.
func (b BinarySpec) GetBuildArgs(env EnvSpec) []string {
	if b.BuildArgs != nil {
		return b.BuildArgs
	}
	return env.BinaryBuildArgs
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
