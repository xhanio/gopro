package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/xhanio/framingo/pkg/types/info"

	"github.com/xhanio/gopro/pkg/types"
)

// gopro schema prints the schema embedded in the binary, so an agent gets the
// fields of the gopro it runs, and needs no project.yaml to do so.
func TestSchemaCommandPrintsTheEmbeddedSchema(t *testing.T) {
	old := types.ProjectSchemaJSON
	t.Cleanup(func() { types.ProjectSchemaJSON = old })
	types.ProjectSchemaJSON = []byte("{\"title\": \"embedded\"}\n")
	t.Chdir(t.TempDir())

	out, err := runRoot(t, "schema")
	if err != nil {
		t.Fatal(err)
	}
	if out != string(types.ProjectSchemaJSON) {
		t.Fatalf("gopro schema printed %q, want the embedded schema", out)
	}
}

// A release points at its own tag; anything else, a local or pseudo version,
// has no tag of its own and points at main.
func TestSchemaURL(t *testing.T) {
	const base = "https://raw.githubusercontent.com/xhanio/gopro/"
	for version, want := range map[string]string{
		"v0.1.13": base + "v0.1.13/project.schema.json",
		"v1.0.0":  base + "v1.0.0/project.schema.json",
		"v0.1.13-0.20261126222033-3ef2e0ffbf15+dirty": base + "main/project.schema.json",
		"v0.1.12-3-gabc1234":                          base + "main/project.schema.json",
		"(devel)":                                     base + "main/project.schema.json",
		"unknown":                                     base + "main/project.schema.json",
	} {
		if got := schemaURL(version); got != want {
			t.Errorf("schemaURL(%q) = %q, want %q", version, got, want)
		}
	}
}

// The repository's example points its editors at the schema beside it; the
// copy gopro example writes cannot, so it points at the installed version's.
func TestExampleWritesTheSchemaURL(t *testing.T) {
	oldExample, oldTag := types.ExampleProjectYAML, info.GitTag
	t.Cleanup(func() { types.ExampleProjectYAML, info.GitTag = oldExample, oldTag })
	types.ExampleProjectYAML = []byte(localSchemaLine + "\nproduct: demo\n")
	info.GitTag = "v9.8.7"
	t.Chdir(t.TempDir())

	var err error
	captureStdout(t, func() { _, err = runRoot(t, "example") })
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("project.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := "# yaml-language-server: $schema=" + schemaURL("v9.8.7") + "\nproduct: demo\n"
	if string(b) != want {
		t.Fatalf("project.yaml = %q, want %q", b, want)
	}
}

// The repository's example has to carry the line gopro example rewrites.
func TestTheExampleCarriesTheLocalSchemaLine(t *testing.T) {
	b, err := os.ReadFile("../../../example.project.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if first, _, _ := strings.Cut(string(b), "\n"); first != localSchemaLine {
		t.Fatalf("example.project.yaml starts with %q, want %q", first, localSchemaLine)
	}
}
