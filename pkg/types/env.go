package types

import (
	"fmt"
	"sort"
	"strings"

	"go.uber.org/config"
)

type EnvSpec struct {
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

	DockerComposeSrc string `yaml:"docker_compose_src,omitempty"`
	DockerComposeTgt string `yaml:"docker_compose_tgt,omitempty"`
}

// DefaultEnv names the default section, selected by an empty -e as well.
const DefaultEnv = "default"

// GetEnv returns the named environment laid over the default one. An empty
// name or "default" selects the default section alone. Any other name must
// be declared under env: a typo is an error rather than a silent build with
// default settings.
func (p *Project) GetEnv(env string) (EnvSpec, error) {
	e, ok := p.Env[env]
	if !ok {
		if env == "" || env == DefaultEnv {
			return p.Default, nil
		}
		names := make([]string, 0, len(p.Env))
		for name := range p.Env {
			names = append(names, name)
		}
		sort.Strings(names)
		return EnvSpec{}, fmt.Errorf("unknown environment %q (declared: %s)", env, strings.Join(names, ", "))
	}
	ds := config.Static(p.Default)
	es := config.Static(e)
	provider, err := config.NewYAML(ds, es)
	if err != nil {
		return EnvSpec{}, fmt.Errorf("merge environment %q: %w", env, err)
	}
	var result EnvSpec
	if err := provider.Get(config.Root).Populate(&result); err != nil {
		return EnvSpec{}, fmt.Errorf("merge environment %q: %w", env, err)
	}
	return result, nil
}
