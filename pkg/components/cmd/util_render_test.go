package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTree writes body to rel under dir, creating parents as needed.
func writeTree(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A template's directory is part of its output path. Stripping the prefix from
// the file name must not also flatten the file into the output root, which
// would silently overwrite same-named templates from sibling directories.
func TestRenderKeepsTemplatesInTheirSubdirectories(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	writeTree(t, src, "template.conf.yaml", "from: root\n")
	writeTree(t, src, "sub/template.conf.yaml", "from: sub\n")
	writeTree(t, src, "sub/deep/template.conf.yaml", "from: deep\n")
	writeTree(t, src, "other/template.conf.yaml", "from: other\n")

	if err := render("api", src, dst, "template.", nil); err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"conf.yaml":          "from: root\n",
		"sub/conf.yaml":      "from: sub\n",
		"sub/deep/conf.yaml": "from: deep\n",
		"other/conf.yaml":    "from: other\n",
	}
	for rel, body := range want {
		got, err := os.ReadFile(filepath.Join(dst, rel))
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		if string(got) != body {
			t.Errorf("%s = %q, want %q", rel, got, body)
		}
	}
}

// Non-template files already kept their directory; templates must behave the
// same way so the two don't diverge.
func TestRenderKeepsPlainFilesInTheirSubdirectories(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	writeTree(t, src, "sub/plain.txt", "plain\n")

	if err := render("api", src, dst, "template.", nil); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dst, "sub", "plain.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "plain\n" {
		t.Errorf("sub/plain.txt = %q, want %q", got, "plain\n")
	}
}

// A glob names a file, not a depth. filepath.Match's * never crosses a
// separator, so matching only the full relative path silently drops every
// nested file.
func TestRenderPatternMatchesNestedFiles(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	writeTree(t, src, "template.root.yaml", "a: root\n")
	writeTree(t, src, "sub/template.nested.yaml", "b: nested\n")
	writeTree(t, src, "sub/skipped.txt", "not yaml\n")

	if err := render("api", src, dst, "template.", []string{"*.yaml"}); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{"root.yaml", "sub/nested.yaml"} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Errorf("%s should have been rendered: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dst, "sub", "skipped.txt")); !os.IsNotExist(err) {
		t.Errorf("sub/skipped.txt should not match *.yaml (err=%v)", err)
	}
}

// A pattern that names a directory still selects by path, so it must not be
// widened into matching that name at any depth.
func TestRenderDirectoryPatternStillScopesToThatDirectory(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	writeTree(t, src, "cert/server.pem", "cert\n")
	writeTree(t, src, "elsewhere/server.pem", "other\n")

	if err := render("api", src, dst, "template.", []string{"cert/*"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dst, "cert", "server.pem")); err != nil {
		t.Errorf("cert/server.pem should have been copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "elsewhere", "server.pem")); !os.IsNotExist(err) {
		t.Errorf("elsewhere/server.pem should not match cert/* (err=%v)", err)
	}
}
