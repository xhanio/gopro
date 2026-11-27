package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/xhanio/gopro/pkg/types"
)

// fakeTools puts go and docker stand-ins first on PATH, each recording its
// invocation, and returns a reader of what was recorded. git stays real: its
// queries run under --dry-run too.
func fakeTools(t *testing.T) func() []string {
	t.Helper()
	bin, log := t.TempDir(), filepath.Join(t.TempDir(), "calls")
	for _, tool := range []string{"go", "docker"} {
		script := "#!/bin/sh\necho \"" + tool + " $*\" >> \"$FAKE_TOOLS_LOG\"\n"
		if err := os.WriteFile(filepath.Join(bin, tool), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FAKE_TOOLS_LOG", log)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func() []string {
		b, err := os.ReadFile(log)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			t.Fatal(err)
		}
		return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	}
}

const dryRunProject = `product: demo
module: x.test/demo
default:
  binary_src: src
  binary_tgt: bin
  binaries: [app]
  binary_build_env: [CGO_ENABLED=0]
  binary_build_args: [-ldflags, '-s']
  image_build_src: images
  images: [base, app]
  config_src: cfg
  config_tgt: out
  configs: [api]
  docker_compose_src: compose
  docker_compose_tgt: out-compose
build:
  binaries:
    - name: app
      platforms:
        - name: linux/arm64
  images:
    - name: base
      build_from: alpine:3
    - name: app
      base: $base
      tag: v1
generate:
  configs:
    - name: api
`

func dryRunDir(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	writeTree(t, ".", "project.yaml", dryRunProject)
	writeTree(t, ".", "images/app/Dockerfile", "FROM scratch\n")
	writeTree(t, ".", "cfg/api/b.txt", "plain\n")
	writeTree(t, ".", "cfg/api/template.a.yaml", "name: [[ .Name ]]\n")
	writeTree(t, ".", "out/api/stale.txt", "from an earlier run\n")
	writeTree(t, ".", "compose/template.docker-compose.yaml", "name: [[ .Project.Product ]]\n")
}

func runCapture(t *testing.T, args ...string) []string {
	t.Helper()
	var err error
	out := captureStdout(t, func() { _, err = runRoot(t, args...) })
	if err != nil {
		t.Fatalf("gopro %s: %v", strings.Join(args, " "), err)
	}
	return strings.Split(strings.TrimRight(out, "\n"), "\n")
}

// Without --dry-run the stand-ins are what runs, so the tests below that see
// no calls are seeing the dry run, not a broken fake.
func TestFakeToolsAreWhatRuns(t *testing.T) {
	calls := fakeTools(t)
	dryRunDir(t)
	runCapture(t, "build", "binary")
	if got := calls(); len(got) != 2 || !strings.HasPrefix(got[0], "go build") {
		t.Fatalf("calls = %q, want two go builds", got)
	}
}

// --dry-run prints each command gopro would run, as a shell line carrying the
// environment gopro adds, and runs none of them.
func TestDryRunBuildBinary(t *testing.T) {
	calls := fakeTools(t)
	dryRunDir(t)
	lines := runCapture(t, "build", "binary", "--dry-run")
	if got := calls(); got != nil {
		t.Fatalf("--dry-run ran %q", got)
	}
	if len(lines) != 2 {
		t.Fatalf("printed %q, want one line per go build", lines)
	}
	host := regexp.MustCompile(`^CGO_ENABLED=0 go build -ldflags '-s -X .*' -o bin/app \S+/src/app$`)
	arm := regexp.MustCompile(`^CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags '-s -X .*' -o bin/app_linux_arm64 \S+/src/app$`)
	if !host.MatchString(lines[0]) || !arm.MatchString(lines[1]) {
		t.Fatalf("printed\n%s", strings.Join(lines, "\n"))
	}
}

// Every docker call is printed in the order it would run, pushes included
// only where --push would push.
func TestDryRunBuildImage(t *testing.T) {
	calls := fakeTools(t)
	dryRunDir(t)
	lines := runCapture(t, "build", "image", "--dry-run", "--push", "--latest")
	if got := calls(); got != nil {
		t.Fatalf("--dry-run ran %q", got)
	}
	var verbs []string
	for _, l := range lines {
		f := strings.Fields(l)
		if len(f) < 3 || f[0] != "docker" {
			t.Fatalf("line %q is not a docker command", l)
		}
		verbs = append(verbs, f[1]+" "+f[len(f)-1])
	}
	want := []string{
		"pull alpine:3",
		"tag base:latest",
		"push base:latest", // already :latest, so --latest adds nothing
		"build " + mustAbs(t, "."),
		"push app:v1",
		"tag app:latest",
		"push app:latest",
	}
	if !slices.Equal(verbs, want) {
		t.Fatalf("printed\n%s\nwant verbs %q", strings.Join(lines, "\n"), want)
	}
}

// generate --dry-run prints the target it would clear and each file it would
// render or copy, and changes nothing on disk.
func TestDryRunGenerateConfig(t *testing.T) {
	dryRunDir(t)
	lines := runCapture(t, "generate", "config", "--dry-run")
	want := []string{
		"rm -rf out/api",
		"cp cfg/api/b.txt out/api/b.txt",
		"render cfg/api/template.a.yaml out/api/a.yaml",
	}
	if !slices.Equal(lines, want) {
		t.Fatalf("printed %q, want %q", lines, want)
	}
	if _, err := os.Stat("out/api/stale.txt"); err != nil {
		t.Errorf("the target was cleared: %v", err)
	}
	if _, err := os.Stat("out/api/a.yaml"); err == nil {
		t.Error("a file was written")
	}
}

func TestDryRunGenerateDockerCompose(t *testing.T) {
	dryRunDir(t)
	lines := runCapture(t, "generate", "docker-compose", "--dry-run")
	want := []string{
		"mkdir -p out-compose",
		"render compose/template.docker-compose.yaml out-compose/docker-compose.yaml",
	}
	if !slices.Equal(lines, want) {
		t.Fatalf("printed %q, want %q", lines, want)
	}
	if _, err := os.Stat("out-compose"); err == nil {
		t.Error("the target was created")
	}
}

// Templates still render under --dry-run, so a broken one fails as it would.
func TestDryRunStillRendersTemplates(t *testing.T) {
	dryRunDir(t)
	writeTree(t, ".", "cfg/api/template.broken.yaml", "x: [[ .NoSuchField ]]\n")
	var err error
	captureStdout(t, func() { _, err = runRoot(t, "generate", "config", "--dry-run") })
	if err == nil || !strings.Contains(err.Error(), "NoSuchField") {
		t.Fatalf("error = %v, want the broken template's", err)
	}
}

// --dry-run is global, so init and example, which have no dry run, refuse it
// before changing anything rather than quietly running for real.
func TestDryRunRefusedByInitAndExample(t *testing.T) {
	for _, args := range [][]string{{"init", "--dry-run"}, {"--dry-run", "example"}} {
		t.Chdir(t.TempDir())
		if args[0] == "init" {
			writeTree(t, ".", "project.yaml", "product: demo\nmodule: x.test/demo\ndefault:\n  binary_tgt: bin\n")
		}
		var err error
		captureStdout(t, func() { _, err = runRoot(t, args...) })
		if err == nil || !strings.Contains(err.Error(), "--dry-run") {
			t.Errorf("gopro %s: error = %v, want a refusal of --dry-run", strings.Join(args, " "), err)
		}
		entries, _ := os.ReadDir(".")
		for _, e := range entries {
			if e.Name() != "project.yaml" || args[0] != "init" {
				t.Errorf("gopro %s created %s", strings.Join(args, " "), e.Name())
			}
		}
	}
}

// Commands with no side effects take --dry-run and run as usual.
func TestDryRunAcceptedBySchema(t *testing.T) {
	old := types.ProjectSchemaJSON
	t.Cleanup(func() { types.ProjectSchemaJSON = old })
	types.ProjectSchemaJSON = []byte("{}\n")
	t.Chdir(t.TempDir())
	out, err := runRoot(t, "schema", "--dry-run")
	if err != nil || out == "" {
		t.Fatalf("gopro schema --dry-run = %q, %v", out, err)
	}
}

func TestShellLine(t *testing.T) {
	got := shellLine([]string{"CGO_ENABLED=0", "CC=a b"}, "go", []string{"build", "-ldflags", "-s -X 'k=v'", "-o", "bin/my app"})
	want := `CGO_ENABLED=0 CC='a b' go build -ldflags '-s -X '\''k=v'\''' -o 'bin/my app'`
	if got != want {
		t.Fatalf("shellLine = %s\nwant          %s", got, want)
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
