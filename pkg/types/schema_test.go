package types

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

const schemaFile = "../../project.schema.json"

func readSchema(t *testing.T) map[string]any {
	t.Helper()
	b, err := os.ReadFile(schemaFile)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func compileSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	b, err := os.ReadFile(schemaFile)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("project.schema.json", doc); err != nil {
		t.Fatal(err)
	}
	s, err := c.Compile("project.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// validateYAML checks a project.yaml body against the schema, decoding it the
// way an editor does: YAML to JSON values.
func validateYAML(t *testing.T, s *jsonschema.Schema, body string) error {
	t.Helper()
	var v any
	if err := yaml.Unmarshal([]byte(body), &v); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return s.Validate(inst)
}

// The schema is written by hand, so every struct the loader populates must
// have its yaml fields, no more and no fewer, at the matching node. A field
// added to a struct without the schema fails here.
func TestSchemaMatchesTypes(t *testing.T) {
	doc := readSchema(t)
	node := func(path ...string) map[string]any {
		var n any = doc
		for _, p := range path {
			n = n.(map[string]any)[p]
		}
		return n.(map[string]any)
	}
	nodes := map[reflect.Type]map[string]any{
		reflect.TypeFor[Project]():           doc,
		reflect.TypeFor[EnvSpec]():           node("$defs", "env"),
		reflect.TypeFor[BuildSpec]():         node("$defs", "build"),
		reflect.TypeFor[BinarySpec]():        node("$defs", "binary"),
		reflect.TypeFor[PlatformSpec]():      node("$defs", "platform"),
		reflect.TypeFor[ImageSpec]():         node("$defs", "image"),
		reflect.TypeFor[GenerateSpec]():      node("$defs", "generate"),
		reflect.TypeFor[ConfigSpec]():        node("$defs", "component"),
		reflect.TypeFor[KubernetesSpec]():    node("$defs", "component"),
		reflect.TypeFor[DockerComposeSpec](): node("$defs", "generate", "properties", "docker_compose"),
	}

	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type)
	walk = func(typ reflect.Type) {
		for typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map || typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}
		seen[typ] = true
		n, ok := nodes[typ]
		if !ok {
			t.Errorf("%s is loaded from project.yaml but has no schema node in this test", typ)
			return
		}
		var fields []string
		for i := range typ.NumField() {
			f := typ.Field(i)
			name, _, _ := strings.Cut(f.Tag.Get("yaml"), ",")
			if name == "" || name == "-" {
				t.Errorf("%s.%s has no yaml name", typ, f.Name)
				continue
			}
			fields = append(fields, name)
			walk(f.Type)
		}
		var props []string
		for p := range n["properties"].(map[string]any) {
			props = append(props, p)
		}
		slices.Sort(fields)
		slices.Sort(props)
		if !slices.Equal(fields, props) {
			t.Errorf("%s fields %q, schema properties %q", typ, fields, props)
		}
		if n["additionalProperties"] != false {
			t.Errorf("%s: schema allows unknown fields, which loading rejects", typ)
		}
	}
	walk(reflect.TypeFor[Project]())
}

// The embedded example is the first project.yaml most users see in an editor,
// lists left empty with only comments under them included.
func TestSchemaAcceptsTheExample(t *testing.T) {
	b, err := os.ReadFile("../../example.project.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateYAML(t, compileSchema(t), string(b)); err != nil {
		t.Fatal(err)
	}
}

func TestSchema(t *testing.T) {
	s := compileSchema(t)
	accept := map[string]string{
		"empty keys are null": "product: demo\nmodel:\ndefault:\n  binary_build_env:\n  binaries:\nenv:\n  local:\nbuild:\ngenerate:\n",
		"a full binary": `product: demo
build:
  binaries:
    - name: api
      build_env: [CGO_ENABLED=1]
      build_args: []
      platforms:
        - name: linux/arm64
          env: [CC=aarch64-linux-gnu-gcc]
`,
		"a $name base": "product: demo\nbuild:\n  images:\n    - name: base\n      build_from: alpine\n    - name: api\n      base: $base\n      no_push: true\n",
	}
	for name, body := range accept {
		if err := validateYAML(t, s, body); err != nil {
			t.Errorf("%s: rejected: %v", name, err)
		}
	}
	reject := map[string]string{
		"product missing":          "default:\n  binaries: []\n",
		"unknown top-level field":  "product: demo\nproject: x.test/demo\n",
		"unknown env field":        "product: demo\ndefault:\n  binary_build_envs: [A=1]\n",
		"component named ..":       "product: demo\ngenerate:\n  configs:\n    - name: ..\n",
		"component name with a /":  "product: demo\nbuild:\n  binaries:\n    - name: cmd/api\n",
		"component without a name": "product: demo\nbuild:\n  images:\n    - base: alpine\n",
		"a list naming one twice":  "product: demo\ndefault:\n  binaries: [api, api]\n",
		"platform not GOOS/GOARCH": "product: demo\nbuild:\n  binaries:\n    - name: api\n      platforms:\n        - name: linux\n",
		"env entry without =":      "product: demo\ndefault:\n  binary_build_env: [CGO_ENABLED]\n",
		"no_push not a boolean":    "product: demo\nbuild:\n  images:\n    - name: api\n      no_push: sometimes\n",
	}
	for name, body := range reject {
		if err := validateYAML(t, s, body); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
