package cmd

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

const testInfo = infoPackage + ".BuildTime=t"

func TestAssembleBuildArgs(t *testing.T) {
	inject := []string{"-X", testInfo}
	tests := []struct {
		name string
		user []string
		want []string
	}{
		{
			name: "injection alone when the user passes no ldflags",
			user: []string{"-v"},
			want: []string{"-v", "-ldflags", "-X " + testInfo},
		},
		{
			name: "user ldflags and injection share one flag",
			user: []string{"-ldflags", "-s -w"},
			want: []string{"-ldflags", "-s -w -X " + testInfo},
		},
		{
			name: "repeated ldflags accumulate in order instead of the last winning",
			user: []string{"-ldflags", "-s", "-ldflags", `-extldflags "-static"`},
			want: []string{"-ldflags", `-s -extldflags "-static" -X ` + testInfo},
		},
		{
			name: "equals and double-dash spellings are the same flag",
			user: []string{"-ldflags=-s", "--ldflags", "-w", "--ldflags=-v"},
			want: []string{"-ldflags", "-s -w -v -X " + testInfo},
		},
		{
			name: "all= reaches the link like the unpatterned form, so they merge",
			user: []string{"-ldflags", "all=-s", "-ldflags", "-w"},
			want: []string{"-ldflags", "-s -w -X " + testInfo},
		},
		{
			name: "an empty value resets what came before, as it does in go build",
			user: []string{"-ldflags", "-s", "-ldflags", ""},
			want: []string{"-ldflags", "-X " + testInfo},
		},
		{
			name: "merged flag sits where the family first appeared",
			user: []string{"-v", "-ldflags", "-s", "-trimpath", "-ldflags", "-w"},
			want: []string{"-v", "-ldflags", "-s -w -X " + testInfo, "-trimpath"},
		},
		{
			name: "gcflags accumulate the same way",
			user: []string{"-gcflags", "-N", "-gcflags=-l"},
			want: []string{"-gcflags", "-N -l", "-ldflags", "-X " + testInfo},
		},
		{
			name: "gcflags patterns stay separate, each one merged",
			user: []string{"-gcflags", "all=-N", "-gcflags", "-m", "-gcflags", "all=-l"},
			want: []string{"-gcflags", "all=-N -l", "-gcflags", "-m", "-ldflags", "-X " + testInfo},
		},
		{
			name: "other flags pass through, values included",
			user: []string{"-tags", "netgo", "-trimpath", "-mod=vendor", "-p", "4"},
			want: []string{"-tags", "netgo", "-trimpath", "-mod=vendor", "-p", "4", "-ldflags", "-X " + testInfo},
		},
		{
			name: "a value that looks like a flag is not taken for one",
			user: []string{"-toolexec", "-o"},
			want: []string{"-toolexec", "-o", "-ldflags", "-X " + testInfo},
		},
		{
			name: "user -X outside the info package is kept",
			user: []string{"-ldflags", "-X main.version=1"},
			want: []string{"-ldflags", "-X main.version=1 -X " + testInfo},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := assembleBuildArgs(tt.user, inject)
			if err != nil {
				t.Fatalf("assembleBuildArgs() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("assembleBuildArgs()\n got  %q\n want %q", got, tt.want)
			}
		})
	}
}

func TestAssembleBuildArgsRejects(t *testing.T) {
	inject := []string{"-X", testInfo}
	tests := []struct {
		name    string
		user    []string
		inject  []string
		wantErr string
	}{
		{name: "-o is set by gopro", user: []string{"-o", "x"}, wantErr: "-o"},
		{name: "-o= is set by gopro", user: []string{"--o=x"}, wantErr: "-o"},
		{name: "info -X is set by gopro", user: []string{"-ldflags", "-X " + infoPackage + ".GitTag=v1"}, wantErr: infoPackage},
		{name: "info -X= is set by gopro", user: []string{"-ldflags", "-X=" + infoPackage + ".GitTag=v1"}, wantErr: infoPackage},
		{name: "quoted info -X is set by gopro", user: []string{"-ldflags", "-X '" + infoPackage + ".GitTag=v 1'"}, wantErr: infoPackage},
		{name: "ldflags pattern other than all", user: []string{"-ldflags", "./cmd/api=-s"}, wantErr: "pattern"},
		{name: "missing value", user: []string{"-v", "-ldflags"}, wantErr: "needs a value"},
		{name: "unterminated quote", user: []string{"-ldflags", `-extldflags "-static`}, wantErr: "unterminated"},
		{name: "injected value that cannot be quoted", inject: []string{"-X", infoPackage + `.ProductName=it's "x"`}, wantErr: "quote"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := inject
			if tt.inject != nil {
				in = tt.inject
			}
			_, err := assembleBuildArgs(tt.user, in)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("assembleBuildArgs() error = %v, want one mentioning %q", err, tt.wantErr)
			}
		})
	}
}

// A value holding a space must reach the linker as one -X field. Unquoted it
// split into a stray token and the link failed with a usage dump.
func TestAssembleBuildArgsQuotesInjectedValues(t *testing.T) {
	want := []string{"-X", infoPackage + ".ProjectRoot=/home/me/My Projects/app", "-X", infoPackage + `.ProductName=it's`}
	got, err := assembleBuildArgs(nil, want)
	if err != nil {
		t.Fatal(err)
	}
	fields, err := splitQuoted(got[1])
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fields, want) {
		t.Fatalf("linker would see %q, want %q", fields, want)
	}
}

// The command line has to be reproducible: info.INJECTION is a map, so
// ranging over it straight reordered the -X flags from run to run.
func TestInjectInfoIsSortedByKey(t *testing.T) {
	fields := injectInfo()
	var keys []string
	for i := 0; i < len(fields); i += 2 {
		if fields[i] != "-X" {
			t.Fatalf("field %d = %q, want -X", i, fields[i])
		}
		key, _, _ := strings.Cut(strings.TrimPrefix(fields[i+1], infoPackage+"."), "=")
		keys = append(keys, key)
	}
	if len(keys) == 0 || !sort.StringsAreSorted(keys) {
		t.Fatalf("injected keys %q are not sorted", keys)
	}
}
