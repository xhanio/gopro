package types

import "go.uber.org/config"

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

func (p *Project) GetEnv(env string) EnvSpec {
	e, ok := p.Env[env]
	if !ok {
		return p.Default
	}
	ds := config.Static(p.Default)
	es := config.Static(e)
	provider, err := config.NewYAML(ds, es)
	if err != nil {
		return p.Default
	}
	var result EnvSpec
	err = provider.Get(config.Root).Populate(&result)
	if err != nil {
		return p.Default
	}
	return result
}
