package types

import (
	"strings"
	"testing"
)

func validProject() Project {
	return Project{
		Default: EnvSpec{
			Binaries:            []string{"api"},
			Images:              []string{"base", "api"},
			Configs:             []string{"api"},
			KubernetesTemplates: []string{"api"},
		},
		Env: map[string]EnvSpec{
			"prod":  {Binaries: []string{"api", "cli"}},
			"local": {},
		},
		Build: BuildSpec{
			Binaries: []BinarySpec{{Name: "api"}, {Name: "cli"}},
			Images:   []ImageSpec{{Name: "base", BuildFrom: "alpine"}, {Name: "api", Base: "$base"}},
		},
		Generate: GenerateSpec{
			Configs:    []ConfigSpec{{Name: "api"}},
			Kubernetes: []KubernetesSpec{{Name: "api"}},
		},
	}
}

func TestValidateAcceptsAValidProject(t *testing.T) {
	p := validProject()
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	p.Default.Images = []string{"api"} // a base built elsewhere is fine
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate() with the base unlisted = %v", err)
	}
}

// A component name becomes a path that generation clears with RemoveAll, so
// ".." would clear the parent of the target.
func TestValidateRejects(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Project)
		wantErr string
	}{
		{"config named ..", func(p *Project) { p.Generate.Configs[0].Name = ".." }, `generate.configs[0].name ".."`},
		{"kubernetes named .", func(p *Project) { p.Generate.Kubernetes[0].Name = "." }, `generate.kubernetes[0].name "."`},
		{"binary with a separator", func(p *Project) { p.Build.Binaries[1].Name = "a/b" }, `build.binaries[1].name "a/b"`},
		{"image without a name", func(p *Project) { p.Build.Images[0].Name = "" }, `build.images[0].name ""`},
		{"duplicate definition", func(p *Project) {
			p.Generate.Configs = append(p.Generate.Configs, ConfigSpec{Name: "api"})
		}, `generate.configs: "api" is defined twice`},
		{"undefined in default", func(p *Project) { p.Default.Binaries = []string{"apii"} }, `default.binaries: "apii" is not defined in build.binaries`},
		{"undefined in an env", func(p *Project) { p.Env["prod"] = EnvSpec{Configs: []string{"web"}} }, `env.prod.configs: "web" is not defined in generate.configs`},
		{"listed twice", func(p *Project) { p.Default.KubernetesTemplates = []string{"api", "api"} }, `default.kubernetes_templates: "api" is listed twice`},
		{"undefined base image", func(p *Project) { p.Build.Images[1].Base = "$bse" }, `build.images[1].base "$bse": "bse" is not defined in build.images`},
		// The images build in list order, so a base listed later is not built yet.
		{"base listed after its image", func(p *Project) { p.Default.Images = []string{"api", "base"} }, `default.images: "api" is listed before its base "base"`},
		{"base listed after its image in an env", func(p *Project) { p.Env["prod"] = EnvSpec{Images: []string{"api", "base"}} }, `env.prod.images: "api" is listed before its base "base"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validProject()
			tt.mutate(&p)
			err := p.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want an error containing %s", err, tt.wantErr)
			}
		})
	}
}

// Every problem is reported at once, so one run shows all there is to fix.
func TestValidateReportsEveryProblem(t *testing.T) {
	p := validProject()
	p.Generate.Configs[0].Name = ".."
	p.Default.Binaries = []string{"apii"}
	err := p.Validate()
	if err == nil {
		t.Fatal("Validate() = nil")
	}
	for _, want := range []string{`".."`, `"apii"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Validate() = %v, want it to mention %s", err, want)
		}
	}
}
