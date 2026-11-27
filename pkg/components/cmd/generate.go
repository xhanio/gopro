package cmd

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/xhanio/gopro/pkg/types"
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
	t := dockerComposeTarget()
	if t.srcs[0] == "" && t.srcs[1] == "" {
		return errors.New("docker-compose: docker_compose_src is not set")
	}
	return generate(t, project.Generate.DockerCompose.Files)
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
	if clear {
		if err := checkClear(t); err != nil {
			return err
		}
	}

	var outs []output
	var read []string
	found := false
	for _, src := range t.srcs {
		if src == "" || slices.Contains(read, src) {
			continue
		}
		read = append(read, src)
		if fi, err := os.Stat(src); err != nil || !fi.IsDir() {
			continue
		}
		found = true
		titlef("Generate %s from %s", t.label, src)
		var err error
		if outs, err = renderLayer(outs, t.name, src, prefix, patterns); err != nil {
			return err
		}
	}
	if !found {
		return fmt.Errorf("%s has no templates: %s does not exist", t.label, strings.Join(read, " nor "))
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

// checkClear refuses to clear t.dst when that would delete what gopro did not
// generate there: the project root, a source gopro reads, or another
// component's output.
func checkClear(t target) error {
	root, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	dst, err := filepath.Abs(t.dst)
	if err != nil {
		return err
	}
	if within(root, filepath.Dir(dst)) {
		return fmt.Errorf("%s: clearing %s would clear a directory of the project root; give its output a directory of its own, such as dist/", t.label, t.dst)
	}
	for _, src := range sourceDirs() {
		hit, err := overlaps(t.dst, src)
		if err != nil {
			return err
		}
		if hit {
			return fmt.Errorf("%s: clearing %s would delete %s, which gopro reads sources from", t.label, t.dst, src)
		}
	}
	for _, other := range envTargets() {
		otherDst, err := filepath.Abs(other.dst)
		if err != nil {
			return err
		}
		if other.label != t.label && within(otherDst, dst) {
			return fmt.Errorf("%s: clearing %s would delete the output of %s in %s", t.label, t.dst, other.label, other.dst)
		}
	}
	return nil
}

// sourceDirs returns every directory gopro reads a component's sources from,
// in any environment.
func sourceDirs() []string {
	specs := []types.EnvSpec{project.Default}
	for _, name := range slices.Sorted(maps.Keys(project.Env)) {
		if spec, err := project.GetEnv(name); err == nil {
			specs = append(specs, spec)
		}
	}
	var dirs []string
	add := func(dir string) {
		if dir != "" && !slices.Contains(dirs, dir) {
			dirs = append(dirs, dir)
		}
	}
	for _, s := range specs {
		for _, b := range project.Build.Binaries {
			if b.Src != "" {
				add(b.Src)
			} else {
				add(filepath.Join(s.BinarySrc, b.Name))
			}
		}
		for _, image := range project.Build.Images {
			if image.BuildSrc != "" {
				add(image.BuildSrc)
			} else if image.BuildFrom == "" {
				add(filepath.Join(s.ImageBuildSrc, image.Name))
			}
		}
		for _, c := range project.Generate.Configs {
			add(filepath.Join(s.ConfigSrc, c.Name))
		}
		for _, k := range project.Generate.Kubernetes {
			add(filepath.Join(s.KubernetesSrc, k.Name))
		}
		add(s.DockerComposeSrc)
	}
	return dirs
}

// envTargets returns the output of every component the selected environment
// generates.
func envTargets() []target {
	var targets []target
	for _, name := range env.Configs {
		targets = append(targets, configTarget(name))
	}
	for _, name := range env.KubernetesTemplates {
		targets = append(targets, kubernetesTarget(name))
	}
	if env.DockerComposeSrc != "" {
		targets = append(targets, dockerComposeTarget())
	}
	return targets
}
