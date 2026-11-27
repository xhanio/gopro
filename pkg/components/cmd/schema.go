package cmd

import (
	"bytes"
	"regexp"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/xhanio/framingo/pkg/types/info"

	"github.com/xhanio/gopro/pkg/types"
)

// example.project.yaml's first line, which gopro example rewrites
const localSchemaLine = "# yaml-language-server: $schema=./project.schema.json"

// only these name a tag; pseudo-versions and git describe output do not
var releaseVersion = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

func NewSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Print the JSON Schema of project.yaml for this gopro",
		// Replaces the root's pre-run, which would load project.yaml.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := cmd.OutOrStdout().Write(types.ProjectSchemaJSON)
			return err
		},
	}
}

func schemaURL(version string) string {
	ref := "main"
	if releaseVersion.MatchString(version) {
		ref = version
	}
	return "https://raw.githubusercontent.com/xhanio/gopro/" + ref + "/project.schema.json"
}

// The copy gopro example writes has no schema file beside it, so it points at
// the published schema instead.
func withSchemaURL(example []byte) []byte {
	rest, ok := bytes.CutPrefix(example, []byte(localSchemaLine+"\n"))
	if !ok {
		return example
	}
	bi, _ := debug.ReadBuildInfo()
	line := "# yaml-language-server: $schema=" + schemaURL(toolVersion(info.GitTag, bi)) + "\n"
	return append([]byte(line), rest...)
}
