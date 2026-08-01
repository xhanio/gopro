package cmd

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func gitignoreLines(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
}

// Appending to a .gitignore that does not end in a newline must not fuse the
// new entry onto the last existing rule, which would destroy that rule and
// lose the entry at the same time.
func TestGitignoreAppendKeepsLastRuleWithoutTrailingNewline(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".gitignore", []byte("node_modules"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := createOrUpdateGitignore(); err != nil {
		t.Fatal(err)
	}

	lines := gitignoreLines(t)
	if !slices.Contains(lines, "node_modules") {
		t.Errorf("existing rule was corrupted: %q", lines)
	}
	if !slices.Contains(lines, "bin/") {
		t.Errorf("bin/ was not added: %q", lines)
	}
}

func TestGitignoreAppendWithTrailingNewline(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".gitignore", []byte("node_modules\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := createOrUpdateGitignore(); err != nil {
		t.Fatal(err)
	}

	lines := gitignoreLines(t)
	for _, want := range []string{"node_modules", "bin/", "dist/", "test/", "secret.env"} {
		if !slices.Contains(lines, want) {
			t.Errorf("%q missing from %q", want, lines)
		}
	}
}

// Entries are appended in a stable order so repeated runs and diffs are
// reviewable rather than reshuffling on every invocation.
func TestGitignoreEntriesAppendInStableOrder(t *testing.T) {
	var first []string
	for i := 0; i < 5; i++ {
		t.Chdir(t.TempDir())
		if err := os.WriteFile(".gitignore", []byte("node_modules\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := createOrUpdateGitignore(); err != nil {
			t.Fatal(err)
		}
		lines := gitignoreLines(t)
		if first == nil {
			first = lines
			continue
		}
		if !slices.Equal(lines, first) {
			t.Fatalf("order varies between runs:\n  %q\n  %q", first, lines)
		}
	}
}

// Running init twice must not duplicate entries.
func TestGitignoreIsIdempotent(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".gitignore", []byte("node_modules\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := createOrUpdateGitignore(); err != nil {
		t.Fatal(err)
	}
	once := gitignoreLines(t)
	if err := createOrUpdateGitignore(); err != nil {
		t.Fatal(err)
	}
	twice := gitignoreLines(t)

	if !slices.Equal(once, twice) {
		t.Errorf("second run changed the file:\n  %q\n  %q", once, twice)
	}
}
