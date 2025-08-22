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
	if configOutput == "" {
		configOutput = envConf.ConfigTgt
	}
	for _, name := range envConf.Configs {
		if !filterRegex.MatchString(name) {
			continue
		}
		for _, config := range conf.Generate.Configs {
			if name != config.Name {
				continue
			}
			// generate config
			if configOutput == "" {
				configOutput = envConf.ConfigSrc
			}
			configDst := filepath.Join(configOutput, config.Name)
			if err := os.RemoveAll(configDst); err != nil {
				return err
			}
			patterns := config.Files
			// render default config
			defaultConfigSrc := filepath.Join(conf.Default.ConfigSrc, config.Name)
			if fi, err := os.Stat(defaultConfigSrc); err == nil && fi.IsDir() {
				titlef("Generate config %s from %s", config.Name, defaultConfigSrc)
				if err := render(config.Name, defaultConfigSrc, configDst, prefix, patterns); err != nil {
					return err
				}
			}
			// render env config
			envConfigSrc := filepath.Join(envConf.ConfigSrc, config.Name)
			if fi, err := os.Stat(envConfigSrc); err == nil && fi.IsDir() {
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
	if kubernetesOutput == "" {
		kubernetesOutput = envConf.KubernetesTgt
	}
	for _, name := range envConf.KubernetesTemplates {
		if !filterRegex.MatchString(name) {
			continue
		}
		for _, template := range conf.Generate.Kubernetes {
			if name != template.Name {
				continue
			}
			// generate kubernetes template
			if kubernetesOutput == "" {
				kubernetesOutput = envConf.KubernetesSrc
			}
			kubernetesDst := filepath.Join(kubernetesOutput, template.Name)
			if err := os.RemoveAll(kubernetesDst); err != nil {
				return err
			}
			patterns := template.Files
			// render default kubernetes template
			defaultKubernetesSrc := filepath.Join(conf.Default.KubernetesSrc, template.Name)
			if fi, err := os.Stat(defaultKubernetesSrc); err == nil && fi.IsDir() {
				titlef("Generate kubernetes template %s from %s", template.Name, defaultKubernetesSrc)
				if err := render(template.Name, defaultKubernetesSrc, kubernetesDst, prefix, patterns); err != nil {
					return err
				}
			}
			// render env kubernetes template
			envKubernetesSrc := filepath.Join(envConf.KubernetesSrc, template.Name)
			if fi, err := os.Stat(envKubernetesSrc); err == nil && fi.IsDir() {
				titlef("Generate kubernetes template %s from %s", template.Name, envKubernetesSrc)
				if err := render(template.Name, envKubernetesSrc, kubernetesDst, prefix, patterns); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
