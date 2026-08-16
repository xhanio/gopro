package types

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The legacy platform list keeps loading alongside the richer platforms spec.
func TestLoadBothPlatformForms(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "project.yaml")
	// module is set so Load doesn't go looking for a go.mod.
	body := `product: demo
module: demo.test/demo
build:
  binaries:
    - name: hello
      platform:
        - darwin/arm64
        - darwin/amd64
      platforms:
        - name: linux/amd64
          env:
            - CGO_ENABLED=1
          args:
            - -tags=netgo
`
	if err := os.WriteFile(conf, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	var p Project
	if err := p.Load(conf); err != nil {
		t.Fatal(err)
	}
	b := p.Build.Binaries[0]
	if !reflect.DeepEqual(b.Platform, []string{"darwin/arm64", "darwin/amd64"}) {
		t.Fatalf("legacy platform = %q", b.Platform)
	}
	if len(b.Platforms) != 1 || b.Platforms[0].Name != "linux/amd64" {
		t.Fatalf("platforms = %+v", b.Platforms)
	}
	if !reflect.DeepEqual(b.Platforms[0].Env, []string{"CGO_ENABLED=1"}) {
		t.Fatalf("platforms[0].Env = %q", b.Platforms[0].Env)
	}
}

// A binary's version key must load, since it feeds the injected
// ApplicationVersion.
func TestLoadBinaryVersion(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "project.yaml")
	// module is set so Load doesn't go looking for a go.mod.
	body := `product: demo
module: demo.test/demo
build:
  binaries:
    - name: hello
      version: v2.3.4
`
	if err := os.WriteFile(conf, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	var p Project
	if err := p.Load(conf); err != nil {
		t.Fatal(err)
	}
	if got := p.Build.Binaries[0].Version; got != "v2.3.4" {
		t.Fatalf("binary version = %q, want %q", got, "v2.3.4")
	}
}

func TestGetPlatforms(t *testing.T) {
	tests := []struct {
		name   string
		binary BinarySpec
		want   []PlatformSpec
	}{
		{
			name:   "legacy platform entries carry no overrides",
			binary: BinarySpec{Platform: []string{"darwin/arm64", "linux/amd64"}},
			want:   []PlatformSpec{{Name: "darwin/arm64"}, {Name: "linux/amd64"}},
		},
		{
			name:   "platforms entries pass through",
			binary: BinarySpec{Platforms: []PlatformSpec{{Name: "linux/amd64", Env: []string{"CGO_ENABLED=1"}}}},
			want:   []PlatformSpec{{Name: "linux/amd64", Env: []string{"CGO_ENABLED=1"}}},
		},
		{
			name: "disjoint names concatenate, legacy first",
			binary: BinarySpec{
				Platform:  []string{"darwin/arm64"},
				Platforms: []PlatformSpec{{Name: "linux/amd64", Env: []string{"CGO_ENABLED=1"}}},
			},
			want: []PlatformSpec{{Name: "darwin/arm64"}, {Name: "linux/amd64", Env: []string{"CGO_ENABLED=1"}}},
		},
		{
			name: "a name in both keeps its position and takes the overrides",
			binary: BinarySpec{
				Platform:  []string{"darwin/arm64", "linux/amd64"},
				Platforms: []PlatformSpec{{Name: "linux/amd64", Env: []string{"CGO_ENABLED=1"}}},
			},
			want: []PlatformSpec{{Name: "darwin/arm64"}, {Name: "linux/amd64", Env: []string{"CGO_ENABLED=1"}}},
		},
		{
			name:   "no platforms declared",
			binary: BinarySpec{},
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.binary.GetPlatforms()
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetPlatforms() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// GetBuildArgs distinguishes an unset build_args from one explicitly set to
// empty, which only works if Load preserves nil vs empty off the YAML.
func TestLoadPreservesUnsetVersusEmptyBuildArgs(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantNil bool
	}{
		{
			name:    "key absent",
			yaml:    "      src: cmd/hello\n",
			wantNil: true,
		},
		{
			name:    "key present but null",
			yaml:    "      build_args:\n",
			wantNil: true,
		},
		{
			name:    "key set to an empty list",
			yaml:    "      build_args: []\n",
			wantNil: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			conf := filepath.Join(dir, "project.yaml")
			// module is set so Load doesn't go looking for a go.mod.
			body := "product: demo\nmodule: demo.test/demo\nbuild:\n  binaries:\n    - name: hello\n" + tt.yaml
			if err := os.WriteFile(conf, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			var p Project
			if err := p.Load(conf); err != nil {
				t.Fatal(err)
			}
			got := p.Build.Binaries[0].BuildArgs
			if (got == nil) != tt.wantNil {
				t.Fatalf("BuildArgs = %#v (nil=%v), want nil=%v", got, got == nil, tt.wantNil)
			}
		})
	}
}

func TestGetImageNameWithTag(t *testing.T) {
	tests := []struct {
		name  string
		image ImageSpec
		env   EnvSpec
		tag   string
		want  string
	}{
		{
			name:  "prefix from image",
			image: ImageSpec{Name: "api", Repo: "api", Prefix: "reg.io/team"},
			tag:   "v1.2.3",
			want:  "reg.io/team/api:v1.2.3",
		},
		{
			name:  "prefix from env",
			image: ImageSpec{Name: "api", Repo: "api"},
			env:   EnvSpec{ImagePrefix: "reg.io/team"},
			tag:   "v1.2.3",
			want:  "reg.io/team/api:v1.2.3",
		},
		{
			name:  "no prefix",
			image: ImageSpec{Name: "api", Repo: "api"},
			tag:   "v1.2.3",
			want:  "api:v1.2.3",
		},
		{
			name:  "repo falls back to name",
			image: ImageSpec{Name: "worker", Prefix: "reg.io"},
			tag:   "latest",
			want:  "reg.io/worker:latest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.image.GetImageNameWithTag(tt.env, tt.tag); got != tt.want {
				t.Fatalf("GetImageNameWithTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetImageNameTagPrecedence(t *testing.T) {
	tests := []struct {
		name  string
		image ImageSpec
		env   EnvSpec
		want  string
	}{
		{
			name:  "image tag wins over env tag",
			image: ImageSpec{Name: "api", Repo: "api", Tag: "v1"},
			env:   EnvSpec{ImageTag: "v2"},
			want:  "api:v1",
		},
		{
			name:  "env tag used when image tag empty",
			image: ImageSpec{Name: "api", Repo: "api"},
			env:   EnvSpec{ImageTag: "v2"},
			want:  "api:v2",
		},
		{
			name:  "defaults to latest",
			image: ImageSpec{Name: "api", Repo: "api"},
			want:  "api:latest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.image.GetImageName(tt.env); got != tt.want {
				t.Fatalf("GetImageName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetImageNameLatestGuard(t *testing.T) {
	// When the primary tag resolves to "latest", the :latest sibling equals it.
	// This is exactly the condition the build-image no-op guard relies on.
	image := ImageSpec{Name: "api", Repo: "api", Prefix: "reg.io"}
	env := EnvSpec{} // no tag anywhere -> defaults to "latest"
	if primary, latest := image.GetImageName(env), image.GetImageNameWithTag(env, "latest"); primary != latest {
		t.Fatalf("expected primary %q to equal latest sibling %q", primary, latest)
	}
}
