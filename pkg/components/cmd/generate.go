package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	prefix string

	kubernetesOutput string
	configOutput     string
)

func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "generate",
	}
	cmd.PersistentFlags().StringVarP(&prefix, "prefix", "x", "template.", "generate files with given prefix")

	cmd.AddCommand(NewGenerateConfigCmd())
	cmd.AddCommand(NewGenerateKubernetesCmd())
	cmd.AddCommand(NewGenerateDockerComposeCmd())
	return cmd
}

func NewGenerateConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "config",
		RunE: runGenerateConfig,
	}
	cmd.Flags().StringVarP(&configOutput, "output", "o", "", "render config output dir")
	return cmd
}

func runGenerateConfig(cmd *cobra.Command, args []string) error {
	out := configOutput
	if out == "" {
		out = env.ConfigTgt
	}
	for _, name := range env.Configs {
		if !filterRegex.MatchString(name) {
			continue
		}
		for _, config := range project.Generate.Configs {
			if name != config.Name {
				continue
			}
			// No target renders in place. Resolved per component, so the fallback
			// cannot leak into the next one.
			dst := out
			if dst == "" {
				dst = env.ConfigSrc
			}
			configDst := filepath.Join(dst, config.Name)
			// The directories actually read, so the overlap guard also catches an unset
			// config_src.
			defaultConfigSrc := filepath.Join(project.Default.ConfigSrc, config.Name)
			envConfigSrc := filepath.Join(env.ConfigSrc, config.Name)
			if err := clearTarget(configDst, defaultConfigSrc, envConfigSrc); err != nil {
				return err
			}
			patterns := config.Files
			if fi, err := os.Stat(defaultConfigSrc); err == nil && fi.IsDir() {
				titlef("Generate config %s from %s", config.Name, defaultConfigSrc)
				if err := render(config.Name, defaultConfigSrc, configDst, prefix, patterns); err != nil {
					return err
				}
			}
			if fi, err := os.Stat(envConfigSrc); err == nil && fi.IsDir() && envConfigSrc != defaultConfigSrc {
				titlef("Generate config %s from %s", config.Name, envConfigSrc)
				if err := render(config.Name, envConfigSrc, configDst, prefix, patterns); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func NewGenerateKubernetesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "kubernetes",
		RunE: runGenerateKubernetes,
	}
	cmd.PersistentFlags().StringVarP(&kubernetesOutput, "output", "t", "", "kubernetes output folder to store rendered templates")
	return cmd
}

func runGenerateKubernetes(cmd *cobra.Command, args []string) error {
	out := kubernetesOutput
	if out == "" {
		out = env.KubernetesTgt
	}
	for _, name := range env.KubernetesTemplates {
		if !filterRegex.MatchString(name) {
			continue
		}
		for _, template := range project.Generate.Kubernetes {
			if name != template.Name {
				continue
			}
			dst := out
			if dst == "" {
				dst = env.KubernetesSrc
			}
			kubernetesDst := filepath.Join(dst, template.Name)
			defaultKubernetesSrc := filepath.Join(project.Default.KubernetesSrc, template.Name)
			envKubernetesSrc := filepath.Join(env.KubernetesSrc, template.Name)
			if err := clearTarget(kubernetesDst, defaultKubernetesSrc, envKubernetesSrc); err != nil {
				return err
			}
			patterns := template.Files
			if fi, err := os.Stat(defaultKubernetesSrc); err == nil && fi.IsDir() {
				titlef("Generate kubernetes template %s from %s", template.Name, defaultKubernetesSrc)
				if err := render(template.Name, defaultKubernetesSrc, kubernetesDst, prefix, patterns); err != nil {
					return err
				}
			}
			if fi, err := os.Stat(envKubernetesSrc); err == nil && fi.IsDir() && envKubernetesSrc != defaultKubernetesSrc {
				titlef("Generate kubernetes template %s from %s", template.Name, envKubernetesSrc)
				if err := render(template.Name, envKubernetesSrc, kubernetesDst, prefix, patterns); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func NewGenerateDockerComposeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "docker-compose",
		RunE: runGenerateDockerCompose,
	}
	return cmd
}

func runGenerateDockerCompose(cmd *cobra.Command, args []string) error {
	outputDir := env.DockerComposeTgt
	if outputDir == "" {
		outputDir = "."
	}

	if err := mkdirAll(outputDir); err != nil {
		return err
	}

	patterns := project.Generate.DockerCompose.Files

	defaultSrc := project.Default.DockerComposeSrc
	if defaultSrc != "" {
		if fi, err := os.Stat(defaultSrc); err == nil && fi.IsDir() {
			titlef("Generate docker-compose from %s", defaultSrc)
			if err := render("docker-compose", defaultSrc, outputDir, prefix, patterns); err != nil {
				return err
			}
		}
	}

	envSrc := env.DockerComposeSrc
	if envSrc != "" && envSrc != defaultSrc {
		if fi, err := os.Stat(envSrc); err == nil && fi.IsDir() {
			titlef("Generate docker-compose from %s", envSrc)
			if err := render("docker-compose", envSrc, outputDir, prefix, patterns); err != nil {
				return err
			}
		}
	}

	return nil
}
