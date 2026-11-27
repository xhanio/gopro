package cmd

import (
	"os"
	"path/filepath"
	"slices"

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
	for _, name := range env.Configs {
		if !filterRegex.MatchString(name) {
			continue
		}
		for _, config := range project.Generate.Configs {
			if name == config.Name {
				if err := generate(configTarget(name), config.Files); err != nil {
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
	for _, name := range env.KubernetesTemplates {
		if !filterRegex.MatchString(name) {
			continue
		}
		for _, template := range project.Generate.Kubernetes {
			if name == template.Name {
				if err := generate(kubernetesTarget(name), template.Files); err != nil {
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
	return generate(dockerComposeTarget(), project.Generate.DockerCompose.Files)
}

// target is where a component renders from, default layer first, and to.
type target struct {
	label, name string
	srcs        []string
	dst         string
	shared      bool // dst holds other output too, so it is not cleared
}

// An unset target renders in place. srcs are the directories actually read,
// so the in-place check also catches an unset config_src.
func configTarget(name string) target {
	return componentTarget("config", name, configOutput, env.ConfigTgt, project.Default.ConfigSrc, env.ConfigSrc)
}

func kubernetesTarget(name string) target {
	return componentTarget("kubernetes template", name, kubernetesOutput, env.KubernetesTgt, project.Default.KubernetesSrc, env.KubernetesSrc)
}

func componentTarget(kind, name, flag, tgt, defaultSrc, envSrc string) target {
	out := flag
	if out == "" {
		out = tgt
	}
	if out == "" {
		out = envSrc
	}
	return target{
		label: kind + " " + name,
		name:  name,
		srcs:  []string{filepath.Join(defaultSrc, name), filepath.Join(envSrc, name)},
		dst:   filepath.Join(out, name),
	}
}

func dockerComposeTarget() target {
	dst := env.DockerComposeTgt
	if dst == "" {
		dst = "."
	}
	return target{
		label:  "docker-compose",
		name:   "docker-compose",
		srcs:   []string{project.Default.DockerComposeSrc, env.DockerComposeSrc},
		dst:    dst,
		shared: true,
	}
}

// generate renders every layer before touching dst, so a template that fails
// leaves the previous output as it was.
func generate(t target, patterns []string) error {
	clear := !t.shared
	for _, src := range t.srcs {
		inPlace, err := overlaps(t.dst, src)
		if err != nil {
			return err
		}
		if inPlace {
			// clearing would delete the templates, so stale output survives
			clear = false
			if verbose {
				debugf("rendering in place into %s; leaving existing files alone", t.dst)
			}
			break
		}
	}
	var outs []output
	var read []string
	for _, src := range t.srcs {
		if src == "" || slices.Contains(read, src) {
			continue
		}
		read = append(read, src)
		if fi, err := os.Stat(src); err != nil || !fi.IsDir() {
			continue
		}
		titlef("Generate %s from %s", t.label, src)
		var err error
		if outs, err = renderLayer(outs, t.name, src, prefix, patterns); err != nil {
			return err
		}
	}
	switch {
	case clear:
		if err := removeAll(t.dst); err != nil {
			return err
		}
	case t.shared:
		if err := mkdirAll(t.dst); err != nil {
			return err
		}
	}
	return writeOutputs(t.dst, outs)
}
