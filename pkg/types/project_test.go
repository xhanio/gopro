package types

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// nil and empty both mean "nothing to pass through", so don't distinguish them.
func equalEntries(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func TestGetBuildEnv(t *testing.T) {
	tests := []struct {
		name   string
		binary BinarySpec
		env    EnvSpec
		want   []string
	}{
		{
			name:   "binary overrides env on the same key",
			binary: BinarySpec{BuildEnv: []string{"CGO_ENABLED=1"}},
			env:    EnvSpec{BinaryBuildEnv: []string{"CGO_ENABLED=0", "GOFLAGS=-mod=vendor"}},
			want:   []string{"CGO_ENABLED=1", "GOFLAGS=-mod=vendor"},
		},
		{
			name:   "key absent from env is appended",
			binary: BinarySpec{BuildEnv: []string{"GOEXPERIMENT=loopvar"}},
			env:    EnvSpec{BinaryBuildEnv: []string{"CGO_ENABLED=0"}},
			want:   []string{"CGO_ENABLED=0", "GOEXPERIMENT=loopvar"},
		},
		{
			name: "env inherited when binary declares none",
			env:  EnvSpec{BinaryBuildEnv: []string{"CGO_ENABLED=0"}},
			want: []string{"CGO_ENABLED=0"},
		},
		{
			name:   "binary used when env declares none",
			binary: BinarySpec{BuildEnv: []string{"CGO_ENABLED=1"}},
			want:   []string{"CGO_ENABLED=1"},
		},
		{
			name:   "key splits on first equals so values may contain equals",
			binary: BinarySpec{BuildEnv: []string{"GOFLAGS=-mod=mod"}},
			env:    EnvSpec{BinaryBuildEnv: []string{"GOFLAGS=-mod=vendor"}},
			want:   []string{"GOFLAGS=-mod=mod"},
		},
		{
			name: "nothing declared anywhere",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.binary.GetBuildEnv(tt.env); !equalEntries(got, tt.want) {
				t.Fatalf("GetBuildEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetBuildArgs(t *testing.T) {
	tests := []struct {
		name   string
		binary BinarySpec
		env    EnvSpec
		want   []string
	}{
		{
			name:   "binary replaces env wholesale rather than merging",
			binary: BinarySpec{BuildArgs: []string{"-trimpath"}},
			env:    EnvSpec{BinaryBuildArgs: []string{"-race", "-v"}},
			want:   []string{"-trimpath"},
		},
		{
			name: "env inherited when binary declares none",
			env:  EnvSpec{BinaryBuildArgs: []string{"-race"}},
			want: []string{"-race"},
		},
		{
			name:   "explicitly empty build_args opts out of the env's",
			binary: BinarySpec{BuildArgs: []string{}},
			env:    EnvSpec{BinaryBuildArgs: []string{"-race", "-v"}},
			want:   nil,
		},
		{
			name:   "binary used when env declares none",
			binary: BinarySpec{BuildArgs: []string{"-trimpath"}},
			want:   []string{"-trimpath"},
		},
		{
			name: "nothing declared anywhere",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.binary.GetBuildArgs(tt.env); !equalEntries(got, tt.want) {
				t.Fatalf("GetBuildArgs() = %q, want %q", got, tt.want)
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
