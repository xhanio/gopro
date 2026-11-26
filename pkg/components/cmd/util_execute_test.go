package cmd

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/xhanio/gopro/pkg/types"
)

// nil and empty both mean "nothing to pass through", so don't distinguish them.
func equalEntries(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func TestBuildArgsFor(t *testing.T) {
	tests := []struct {
		name     string
		env      types.EnvSpec
		binary   types.BinarySpec
		platform types.PlatformSpec
		want     []string
	}{
		{
			name: "env used when nothing more specific is declared",
			env:  types.EnvSpec{BinaryBuildArgs: []string{"-v"}},
			want: []string{"-v"},
		},
		{
			name:   "binary replaces env wholesale rather than merging",
			env:    types.EnvSpec{BinaryBuildArgs: []string{"-race", "-v"}},
			binary: types.BinarySpec{BuildArgs: []string{"-trimpath"}},
			want:   []string{"-trimpath"},
		},
		{
			name:     "platform replaces binary wholesale",
			env:      types.EnvSpec{BinaryBuildArgs: []string{"-race"}},
			binary:   types.BinarySpec{BuildArgs: []string{"-trimpath"}},
			platform: types.PlatformSpec{Args: []string{"-tags=netgo"}},
			want:     []string{"-tags=netgo"},
		},
		{
			name:     "platform inherits the binary's when it declares none",
			env:      types.EnvSpec{BinaryBuildArgs: []string{"-race"}},
			binary:   types.BinarySpec{BuildArgs: []string{"-trimpath"}},
			platform: types.PlatformSpec{},
			want:     []string{"-trimpath"},
		},
		{
			name:   "explicitly empty build_args opts out of the env's",
			env:    types.EnvSpec{BinaryBuildArgs: []string{"-race", "-v"}},
			binary: types.BinarySpec{BuildArgs: []string{}},
			want:   nil,
		},
		{
			name:     "explicitly empty platform args opts out of the binary's",
			binary:   types.BinarySpec{BuildArgs: []string{"-trimpath"}},
			platform: types.PlatformSpec{Args: []string{}},
			want:     nil,
		},
		{
			name: "nothing declared anywhere",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildArgsFor(tt.env, tt.binary, tt.platform); !equalEntries(got, tt.want) {
				t.Fatalf("buildArgsFor() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Image builds pass --no-cache unless --cache asks for the layer cache.
func TestDockerBuildArgsCache(t *testing.T) {
	for _, cache := range []bool{false, true} {
		args, err := dockerBuildArgs("api", "src", "api:v1", "alpine", cache, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := slices.Contains(args, "--no-cache"); got == cache {
			t.Errorf("cache=%v: args %q, want --no-cache only without cache", cache, args)
		}
	}
}

// image_build_args reach docker build in order, before -f and the context.
func TestDockerBuildArgsPassesImageBuildArgs(t *testing.T) {
	user := []string{"--platform", "linux/arm64", "--build-arg", "VERSION=1", "--progress=plain"}
	args, err := dockerBuildArgs("api", "src", "api:v1", "alpine", true, user)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(args, " "), strings.Join(user, " ")) {
		t.Fatalf("args %q lack %q in order", args, user)
	}
	if n := len(args); args[n-3] != "-f" || !strings.HasSuffix(args[n-2], "Dockerfile") {
		t.Fatalf("args %q do not end with -f <Dockerfile> <context>", args)
	}
}

// Flags gopro sets itself are rejected, not silently overridden.
func TestDockerBuildArgsRejectsReserved(t *testing.T) {
	tests := []struct {
		name    string
		user    []string
		wantErr string
	}{
		{name: "-t", user: []string{"-t", "x:y"}, wantErr: "-t"},
		{name: "-t attached", user: []string{"-tx:y"}, wantErr: "-t"},
		{name: "--tag=", user: []string{"--tag=x:y"}, wantErr: "-t"},
		{name: "-f", user: []string{"-f", "Other.Dockerfile"}, wantErr: "-f"},
		{name: "--file", user: []string{"--file", "Other.Dockerfile"}, wantErr: "-f"},
		{name: "injected build arg", user: []string{"--build-arg", "BASE=debian"}, wantErr: "BASE"},
		{name: "injected build arg with =", user: []string{"--build-arg=CONFIG_DIR=/x"}, wantErr: "CONFIG_DIR"},
		{name: "injected build arg from env", user: []string{"--build-arg", "NAME"}, wantErr: "NAME"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dockerBuildArgs("api", "src", "api:v1", "alpine", false, tt.user)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("dockerBuildArgs() error = %v, want one mentioning %q", err, tt.wantErr)
			}
		})
	}
}

// Flags that merely resemble the reserved ones pass.
func TestDockerBuildArgsAllowsLookAlikes(t *testing.T) {
	user := []string{"--target", "release", "--build-arg", "NAMESPACE=x", "--file-mode", "--tags-extra"}
	if _, err := dockerBuildArgs("api", "src", "api:v1", "alpine", false, user); err != nil {
		t.Fatalf("dockerBuildArgs() error = %v", err)
	}
}

func TestBuildImageHasCacheFlag(t *testing.T) {
	f := NewBuildImageCmd().Flags().Lookup("cache")
	if f == nil {
		t.Fatal("build image has no --cache flag")
	}
	if f.DefValue != "false" {
		t.Fatalf("--cache defaults to %s, want false", f.DefValue)
	}
}
