package types

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"go.uber.org/config"
	"golang.org/x/mod/modfile"
)

type Config struct {
	Product  string               `yaml:"product"`
	Model    string               `yaml:"model"`
	Version  string               `yaml:"version"`
	Domain   string               `yaml:"domain"`
	Project  string               `yaml:"project"`
	Default  EnvConfig            `yaml:"default"`
	Env      map[string]EnvConfig `yaml:"env"`
	Build    BuildConfig          `yaml:"build"`
	Generate GenerateConfig       `yaml:"generate"`
}

func (c *Config) GetEnv(env string) EnvConfig {
	e, ok := c.Env[env]
	if !ok {
		return c.Default
	}
	ds := config.Static(c.Default)
	es := config.Static(e)
	p, err := config.NewYAML(ds, es)
	if err != nil {
		return c.Default
	}
	var result EnvConfig
	err = p.Get(config.Root).Populate(&result)
	if err != nil {
		return c.Default
	}
	return result
}

func (c *Config) Load(confPath string) error {
	p, err := config.NewYAML(config.File(confPath))
	if err != nil {
		return err
	}
	err = p.Get(config.Root).Populate(c)
	if err != nil {
		return err
	}
	if c.Project == "" {
		mb, err := os.ReadFile(filepath.Join(filepath.Dir(confPath), "go.mod"))
		if err != nil {
			return err
		}
		c.Project = modfile.ModulePath(mb)
	}
	return nil
}

type BuildConfig struct {
	Binaries []BinaryDefinition `yaml:"binaries"`
	Images   []ImageDefinition  `yaml:"images"`
}

type BinaryDefinition struct {
	Name      string   `yaml:"name"`
	Src       string   `yaml:"src,omitempty"`
	Platform  []string `yaml:"platform,omitempty"`
	ConfigDir string   `yaml:"config_dir,omitempty"`
}

type ImageDefinition struct {
	Name      string `yaml:"name"`
	Base      string `yaml:"base,omitempty"`
	BuildSrc  string `yaml:"build_src,omitempty"`
	BuildFrom string `yaml:"build_from,omitempty"`
	Prefix    string `yaml:"prefix,omitempty"`
	Repo      string `yaml:"repo,omitempty"`
	Tag       string `yaml:"tag,omitempty"`
	NoPush    bool   `yaml:"no_push,omitempty"`
}

func (i ImageDefinition) GetImageName(env EnvConfig) string {
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

type EnvConfig struct {
	ConfigSrc string   `yaml:"config_src,omitempty"`
	ConfigTgt string   `yaml:"config_tgt,omitempty"`
	Configs   []string `yaml:"configs,omitempty"`

	BinarySrc       string   `yaml:"binary_src,omitempty"`
	BinaryTgt       string   `yaml:"binary_tgt,omitempty"`
	Binaries        []string `yaml:"binaries,omitempty"`
	BinaryBuildEnv  []string `yaml:"binary_build_env,omitempty"`
	BinaryBuildArgs []string `yaml:"binary_build_args,omitempty"`

	ImageBuildSrc  string   `yaml:"image_build_src,omitempty"`
	Images         []string `yaml:"images,omitempty"`
	ImagePrefix    string   `yaml:"image_prefix,omitempty"`
	ImageTag       string   `yaml:"image_tag,omitempty"`
	ImageBuildEnv  []string `yaml:"image_build_env,omitempty"`
	ImageBuildArgs []string `yaml:"image_build_args,omitempty"`

	KubernetesSrc       string   `yaml:"kubernetes_src,omitempty"`
	KubernetesTgt       string   `yaml:"kubernetes_tgt,omitempty"`
	KubernetesTemplates []string `yaml:"kubernetes_templates,omitempty"`
}

type GenerateConfig struct {
	Configs    []ConfigDefinition     `yaml:"configs"`
	Kubernetes []KubernetesDefinition `yaml:"kubernetes"`
}

type ConfigDefinition struct {
	Name  string   `yaml:"name"`
	Src   string   `yaml:"src,omitempty"`
	Files []string `yaml:"files,omitempty"`
}

type KubernetesDefinition struct {
	Name  string   `yaml:"name"`
	Src   string   `yaml:"src,omitempty"`
	Files []string `yaml:"files,omitempty"`
}
