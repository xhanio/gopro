package types

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"go.uber.org/config"
	"golang.org/x/mod/modfile"
)

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
	ConfigDir string   `yaml:"config_dir,omitempty"`
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

func (i ImageSpec) GetImageName(env EnvSpec) string {
	// image repo defined in build section
	repo := i.Repo
	if repo == "" {
		// image repo undefined, use image name instead
		repo = i.Name
	}
	// image tag defined in build section
	tag := i.Tag
	if tag == "" {
		tag = env.ImageTag
	}
	if tag == "" {
		tag = "latest"
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
