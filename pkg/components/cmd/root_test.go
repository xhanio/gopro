package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/xhanio/framingo/pkg/types/info"

	"github.com/xhanio/gopro/pkg/types"
)

// resetInfo clears the framingo package vars this file asserts on, so one
// case cannot leak into the next.
func resetInfo(t *testing.T) {
	t.Helper()
	old := struct{ name, model, pversion, bversion, tag, module, aname, aversion string }{
		info.ProductName, info.ProductModel, info.ProductVersion,
		info.BuildVersion, info.GitTag, info.ProjectName,
		info.ApplicationName, info.ApplicationVersion,
	}
	t.Cleanup(func() {
		info.ProductName, info.ProductModel, info.ProductVersion = old.name, old.model, old.pversion
		info.BuildVersion, info.GitTag, info.ProjectName = old.bversion, old.tag, old.module
		info.ApplicationName, info.ApplicationVersion = old.aname, old.aversion
	})
	info.ProductName, info.ProductModel, info.ProductVersion = "", "", ""
	info.BuildVersion, info.GitTag, info.ProjectName = "", "", ""
	info.ApplicationName, info.ApplicationVersion = "", ""
}

// project.yaml declares a model, so it has to reach the injected metadata.
// The field was parsed and then never read, leaving info.ProductModel empty
// with no warning for anyone who set it.
func TestApplyProjectInfoCarriesModel(t *testing.T) {
	resetInfo(t)

	applyProjectInfo(types.Project{Product: "demo", Model: "rackmount"})

	if info.ProductModel != "rackmount" {
		t.Errorf("ProductModel = %q, want %q", info.ProductModel, "rackmount")
	}
}

// The --product-model flag still outranks the config, matching how
// --product-version overrides version.
func TestProductModelFlagOverridesConfig(t *testing.T) {
	resetInfo(t)
	oldFlag := productModel
	t.Cleanup(func() { productModel = oldFlag })

	applyProjectInfo(types.Project{Product: "demo", Model: "from-config"})
	productModel = "from-flag"
	overwriteBuildInfo()

	if info.ProductModel != "from-flag" {
		t.Errorf("ProductModel = %q, want the flag to win", info.ProductModel)
	}
}

// An unset flag must leave the configured value alone.
func TestUnsetProductModelFlagKeepsConfig(t *testing.T) {
	resetInfo(t)
	oldFlag := productModel
	t.Cleanup(func() { productModel = oldFlag })

	applyProjectInfo(types.Project{Product: "demo", Model: "from-config"})
	productModel = ""
	overwriteBuildInfo()

	if info.ProductModel != "from-config" {
		t.Errorf("ProductModel = %q, want the config value retained", info.ProductModel)
	}
}

// The surrounding metadata must keep behaving as it did.
func TestApplyProjectInfoExistingFields(t *testing.T) {
	tests := []struct {
		name                                 string
		project                              types.Project
		gitTag                               string
		wantName, wantModule                 string
		wantBuildVersion, wantProductVersion string
	}{
		{
			name:               "explicit version wins",
			project:            types.Project{Product: "demo", Module: "x.test/demo", Version: "v9"},
			gitTag:             "v1",
			wantName:           "demo",
			wantModule:         "x.test/demo",
			wantBuildVersion:   "v1",
			wantProductVersion: "v9",
		},
		{
			name:               "product version falls back to build version",
			project:            types.Project{Product: "demo"},
			gitTag:             "v1",
			wantName:           "demo",
			wantBuildVersion:   "v1",
			wantProductVersion: "v1",
		},
		{
			name:     "no git tag leaves versions empty",
			project:  types.Project{Product: "demo"},
			wantName: "demo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetInfo(t)
			info.GitTag = tt.gitTag

			applyProjectInfo(tt.project)

			if info.ProductName != tt.wantName {
				t.Errorf("ProductName = %q, want %q", info.ProductName, tt.wantName)
			}
			if info.ProjectName != tt.wantModule {
				t.Errorf("ProjectName = %q, want %q", info.ProjectName, tt.wantModule)
			}
			if info.BuildVersion != tt.wantBuildVersion {
				t.Errorf("BuildVersion = %q, want %q", info.BuildVersion, tt.wantBuildVersion)
			}
			if info.ProductVersion != tt.wantProductVersion {
				t.Errorf("ProductVersion = %q, want %q", info.ProductVersion, tt.wantProductVersion)
			}
		})
	}
}

func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	oldPath, oldEnv, oldVerbose := projectPath, envName, verbose
	t.Cleanup(func() { projectPath, envName, verbose = oldPath, oldEnv, oldVerbose })
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

// Runtime errors print no usage.
func TestRuntimeErrorOmitsUsage(t *testing.T) {
	out, err := runRoot(t, "build", "binary", "-c", filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("want an error for a missing config")
	}
	if strings.Contains(out, "Usage:") {
		t.Fatalf("runtime error printed usage:\n%s", out)
	}
}

// Flag errors still print usage.
func TestFlagErrorKeepsUsage(t *testing.T) {
	out, err := runRoot(t, "build", "binary", "--no-such-flag")
	if err == nil {
		t.Fatal("want an error for an unknown flag")
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("flag error printed no usage:\n%s", out)
	}
}

func TestResolveProjectPath(t *testing.T) {
	abs := func(elem ...string) string {
		return filepath.Join(append([]string{string(filepath.Separator)}, elem...)...)
	}
	gopath, other, outside := abs("home", "me", "go"), abs("opt", "go"), abs("home", "me", "code", "gopro")
	tests := []struct {
		name    string
		wd      string
		gopaths []string
		module  string
		want    string
	}{
		{
			name:    "under GOPATH/src stays GOPATH-relative",
			wd:      filepath.Join(gopath, "src", "xhanio", "gopro"),
			gopaths: []string{gopath},
			module:  "github.com/xhanio/gopro",
			want:    "xhanio/gopro",
		},
		{
			name:    "any GOPATH entry counts",
			wd:      filepath.Join(other, "src", "xhanio", "gopro"),
			gopaths: []string{gopath, other},
			want:    "xhanio/gopro",
		},
		{
			name:    "outside GOPATH uses the module path",
			wd:      outside,
			gopaths: []string{gopath},
			module:  "github.com/xhanio/gopro",
			want:    "github.com/xhanio/gopro",
		},
		{
			name:    "a sibling of GOPATH/src is not inside it",
			wd:      filepath.Join(gopath, "srcx", "gopro"),
			gopaths: []string{gopath},
			module:  "github.com/xhanio/gopro",
			want:    "github.com/xhanio/gopro",
		},
		{
			name:    "no GOPATH match and no module leaves it empty",
			wd:      outside,
			gopaths: []string{gopath},
			want:    "",
		},
		{
			name:   "unset GOPATH uses the module path",
			wd:     outside,
			module: "github.com/xhanio/gopro",
			want:   "github.com/xhanio/gopro",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveProjectPath(tt.wd, tt.gopaths, tt.module); got != tt.want {
				t.Fatalf("resolveProjectPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// captureStdout returns what f prints to os.Stdout, where debugf writes.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	defer func() { os.Stdout = old }()
	f()
	w.Close()
	return <-done
}

// -v prints the env a command adds, never the inherited environment.
func TestVerboseDoesNotPrintEnvironment(t *testing.T) {
	const secret = "gopro-test-secret-value"
	t.Setenv("GOPRO_TEST_SECRET", secret)
	oldTag := info.GitTag
	t.Cleanup(func() { info.GitTag = oldTag })
	conf := filepath.Join(t.TempDir(), "project.yaml")
	if err := os.WriteFile(conf, []byte("product: demo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"-v", "-c", conf, "generate", "config"},
		{"-v", "version"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			info.GitTag = "" // so version falls back to git
			var err error
			out := captureStdout(t, func() { _, err = runRoot(t, args...) })
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "executing git") {
				t.Fatalf("no git call traced, so nothing was checked:\n%s", out)
			}
			if strings.Contains(out, secret) {
				t.Fatalf("verbose output leaked the environment:\n%s", out)
			}
		})
	}
}

// Help names the program and keeps cobra's -h, --help flag.
func TestHelpNamesTheProgram(t *testing.T) {
	out, err := runRoot(t, "build", "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"gopro build [command]", `Use "gopro build [command] --help"`} {
		if !strings.Contains(out, want) {
			t.Errorf("help lacks %q:\n%s", want, out)
		}
	}
	if !regexp.MustCompile(`-h, --help\s+help for build`).MatchString(out) {
		t.Errorf("help lacks cobra's -h, --help line:\n%s", out)
	}
}

// example replaces the root's pre-run to skip config loading, so it has to
// silence usage on its own.
func TestExampleErrorOmitsUsage(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("project.yaml", []byte("product: demo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := runRoot(t, "example")
	if err == nil {
		t.Fatal("want an error when project.yaml already exists")
	}
	if strings.Contains(out, "Usage:") {
		t.Fatalf("runtime error printed usage:\n%s", out)
	}
}
