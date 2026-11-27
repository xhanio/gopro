package main

import (
	_ "embed"
	"os"

	"github.com/xhanio/gopro/pkg/components/cmd"
	"github.com/xhanio/gopro/pkg/types"
)

//go:embed example.project.yaml
var exampleProjectYAML []byte

//go:embed project.schema.json
var projectSchemaJSON []byte

func main() {
	types.ExampleProjectYAML = exampleProjectYAML
	types.ProjectSchemaJSON = projectSchemaJSON
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
