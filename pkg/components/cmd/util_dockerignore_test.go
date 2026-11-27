package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// gopro init makes the root .dockerignore exclude secret.env at any depth,
// keeping whatever the file already says, and leaves bin/ and dist/ in the
// context, since Dockerfiles copy binaries and rendered config from them.
func TestDockerignoreCoversSecretEnv(t *testing.T) {
	tests := []struct {
		name, existing string
		want           []string // lines of the resulting file
	}{
		{name: "created when missing", want: []string{"**/secret.env"}},
		{name: "appended, existing rules kept", existing: ".git\nnode_modules\n", want: []string{".git", "node_modules", "**/secret.env"}},
		{name: "appended without fusing an unterminated rule", existing: ".git", want: []string{".git", "**/secret.env"}},
		{name: "a root-only rule does not cover nested files", existing: "secret.env\n", want: []string{"secret.env", "**/secret.env"}},
		{name: "already covered", existing: "**/*.env\n", want: []string{"**/*.env"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if tt.existing != "" {
				writeTree(t, ".", ".dockerignore", tt.existing)
			}
			for range 2 { // and running init again changes nothing
				if err := createOrUpdateDockerignore(); err != nil {
					t.Fatal(err)
				}
			}
			b, err := os.ReadFile(".dockerignore")
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Split(strings.TrimRight(string(b), "\n"), "\n"); !slices.Equal(got, tt.want) {
				t.Fatalf(".dockerignore = %q, want %q", got, tt.want)
			}
		})
	}
}

// The build context is the project root, so every secret.env under env/ is
// sent to docker unless .dockerignore excludes it.
func TestSecretsInBuildContext(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"env/default/config/api/secret.env",
		"env/local/config/api/secret.env",
		".git/secret.env", // skipped: .git holds no working-tree files
		"env/default/config/api/config.yaml",
	} {
		writeTree(t, root, rel, "K=v\n")
	}
	both := []string{"env/default/config/api/secret.env", "env/local/config/api/secret.env"}
	tests := []struct {
		name     string
		patterns []string
		want     []string
	}{
		{name: "no .dockerignore", want: both},
		{name: "excluded by name", patterns: []string{"**/secret.env"}},
		{name: "excluded with a parent directory", patterns: []string{"env"}},
		{name: "a later exception brings one back", patterns: []string{"**/secret.env", "!env/local/**"}, want: []string{"env/local/config/api/secret.env"}},
		{name: "unrelated rules", patterns: []string{"node_modules", "*.log"}, want: both},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets, err := findSecretEnvFiles(root)
			if err != nil {
				t.Fatal(err)
			}
			got, err := unexcluded(secrets, tt.patterns)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("exposed = %q, want %q", got, tt.want)
			}
		})
	}
}

// BuildKit reads <Dockerfile>.dockerignore beside the Dockerfile in place of
// the root .dockerignore.
func TestDockerignoreForPrefersTheDockerfileOwn(t *testing.T) {
	root := t.TempDir()
	dockerfile := filepath.Join(root, "build/image/api/Dockerfile")
	writeTree(t, root, "build/image/api/Dockerfile", "FROM scratch\n")
	writeTree(t, root, ".dockerignore", "root-rule\n")

	got, err := dockerignoreFor(root, dockerfile)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"root-rule"}) {
		t.Fatalf("patterns = %q, want the root .dockerignore", got)
	}

	writeTree(t, root, "build/image/api/Dockerfile.dockerignore", "own-rule\n")
	if got, err = dockerignoreFor(root, dockerfile); err != nil || !slices.Equal(got, []string{"own-rule"}) {
		t.Fatalf("patterns = %q, %v, want the Dockerfile's own", got, err)
	}

	if got, err = dockerignoreFor(t.TempDir(), dockerfile+"-missing"); err != nil || got != nil {
		t.Fatalf("patterns = %q, %v, want none without any ignore file", got, err)
	}
}

// build image warns, naming each secret.env the Dockerfile's context would
// send to docker, and says nothing once .dockerignore excludes them.
func TestWarnSecretsInContext(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, "env/local/config/api/secret.env", "K=v\n")
	dockerfile := filepath.Join(root, "build/image/api/Dockerfile")
	secrets := []string{"env/local/config/api/secret.env"}

	out := captureStdout(t, func() {
		if err := warnSecretsInContext(secrets, root, dockerfile); err != nil {
			t.Error(err)
		}
	})
	for _, want := range []string{"env/local/config/api/secret.env", dockerignoreSecretRule} {
		if !strings.Contains(out, want) {
			t.Errorf("warning %q does not mention %s", out, want)
		}
	}

	writeTree(t, root, ".dockerignore", dockerignoreSecretRule+"\n")
	if out := captureStdout(t, func() { _ = warnSecretsInContext(secrets, root, dockerfile) }); out != "" {
		t.Errorf("warned although .dockerignore excludes it: %q", out)
	}
}
