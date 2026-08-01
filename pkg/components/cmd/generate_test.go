package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/xhanio/gopro/pkg/types"
)

// withProject points the package-level command state at a project for the
// duration of one test and restores it afterwards.
func withProject(t *testing.T, p types.Project, e types.EnvSpec) {
	t.Helper()
	oldProject, oldEnv := project, env
	oldConfigOut, oldK8sOut, oldFilter := configOutput, kubernetesOutput, filterRegex
	oldPrefix := prefix
	t.Cleanup(func() {
		project, env = oldProject, oldEnv
		configOutput, kubernetesOutput, filterRegex = oldConfigOut, oldK8sOut, oldFilter
		prefix = oldPrefix
	})
	project, env = p, e
	configOutput, kubernetesOutput = "", ""
	filterRegex = regexp.MustCompile(".*")
	// cobra normally supplies this via the --prefix default.
	prefix = "template."
}

func configProject(src, tgt string) (types.Project, types.EnvSpec) {
	spec := types.EnvSpec{ConfigSrc: src, ConfigTgt: tgt, Configs: []string{"api"}}
	return types.Project{
		Default:  spec,
		Generate: types.GenerateSpec{Configs: []types.ConfigSpec{{Name: "api"}}},
	}, spec
}

func seedConfigSource(t *testing.T) string {
	t.Helper()
	t.Chdir(t.TempDir())
	src := filepath.Join("env", "default", "config", "api")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "template.conf.yaml"), []byte("a: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return src
}

// With no target configured, output lands beside the templates so a component
// renders in place. Generation clears its target first, so the clear has to be
// skipped here -- it would otherwise delete the very templates being rendered.
func TestGenerateConfigWithoutTargetRendersInPlace(t *testing.T) {
	src := seedConfigSource(t)
	p, e := configProject("env/default/config", "")
	withProject(t, p, e)

	if err := runGenerateConfig(nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(src, "template.conf.yaml")); err != nil {
		t.Errorf("source template was deleted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(src, "conf.yaml")); err != nil {
		t.Errorf("template was not rendered in place: %v", err)
	}
}

// Rendering in place also happens when the target is set onto the source
// deliberately, and must behave the same way.
func TestGenerateConfigWithTargetOverlappingSourceRendersInPlace(t *testing.T) {
	src := seedConfigSource(t)
	p, e := configProject("env/default/config", "env/default/config")
	withProject(t, p, e)

	if err := runGenerateConfig(nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(src, "template.conf.yaml")); err != nil {
		t.Errorf("source template was deleted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(src, "conf.yaml")); err != nil {
		t.Errorf("template was not rendered in place: %v", err)
	}
}

// With neither source nor target set the destination collapses to a bare
// component name, which resolves onto whatever directory in the project root
// happens to share it. That is also the source, so nothing may be deleted.
func TestGenerateConfigWithNoPathsConfiguredDeletesNothing(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("api", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("api", "handler.go"), []byte("package api\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, e := configProject("", "")
	withProject(t, p, e)

	if err := runGenerateConfig(nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join("api", "handler.go")); err != nil {
		t.Fatalf("unrelated directory was deleted: %v", err)
	}
}

// The guard must not break the ordinary case: a distinct target still renders,
// and is still cleared of stale output first.
func TestGenerateConfigRendersAndClearsDistinctTarget(t *testing.T) {
	seedConfigSource(t)
	p, e := configProject("env/default/config", "dist/config")
	withProject(t, p, e)

	stale := filepath.Join("dist", "config", "api", "stale.yaml")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := runGenerateConfig(nil, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join("dist", "config", "api", "conf.yaml")); err != nil {
		t.Errorf("template was not rendered: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale output should have been cleared (err=%v)", err)
	}
}

func kubernetesProject(src, tgt string) (types.Project, types.EnvSpec) {
	spec := types.EnvSpec{KubernetesSrc: src, KubernetesTgt: tgt, KubernetesTemplates: []string{"api"}}
	return types.Project{
		Default:  spec,
		Generate: types.GenerateSpec{Kubernetes: []types.KubernetesSpec{{Name: "api"}}},
	}, spec
}

func TestGenerateKubernetesWithoutTargetRendersInPlace(t *testing.T) {
	t.Chdir(t.TempDir())
	src := filepath.Join("env", "default", "kubernetes", "api")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "template.deployment.yaml"), []byte("kind: Deployment\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, e := kubernetesProject("env/default/kubernetes", "")
	withProject(t, p, e)

	if err := runGenerateKubernetes(nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(src, "template.deployment.yaml")); err != nil {
		t.Errorf("source template was deleted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(src, "deployment.yaml")); err != nil {
		t.Errorf("template was not rendered in place: %v", err)
	}
}
