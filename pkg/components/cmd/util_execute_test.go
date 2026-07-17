package cmd

import (
	"reflect"
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
