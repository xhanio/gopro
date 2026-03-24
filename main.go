package main

import (
	_ "embed"
	"os"

	"github.com/xhanio/gopro/pkg/components/cmd"
	"github.com/xhanio/gopro/pkg/types"
)

//go:embed example.project.yaml
var exampleProjectYAML []byte

func main() {
	types.ExampleProjectYAML = exampleProjectYAML
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(-1)
	}
}
