package cmd

import (
	"bytes"
	"regexp"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/xhanio/framingo/pkg/types/info"

	"github.com/xhanio/gopro/pkg/types"
)

// localSchemaLine is the first line of the repository's example.project.yaml,
// pointing editors at the schema file beside it.
const localSchemaLine = "# yaml-language-server: $schema=./project.schema.json"

// releaseVersion matches a version that names a tag of its own, unlike a
// pseudo-version or git describe output.
var releaseVersion = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

func NewSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Print the JSON Schema of project.yaml for this gopro",
		// Like example, needs no project.yaml, so the root's pre-run, which
		// loads one, is replaced.
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

// schemaURL returns where editors fetch the schema of the given gopro
// version: its own tag for a release, main for anything else.
func schemaURL(version string) string {
	ref := "main"
	if releaseVersion.MatchString(version) {
		ref = version
	}
	return "https://raw.githubusercontent.com/xhanio/gopro/" + ref + "/project.schema.json"
}

// withSchemaURL points an example's schema line at the running gopro's
// schema, since the copy gopro example writes has no schema file beside it.
func withSchemaURL(example []byte) []byte {
	rest, ok := bytes.CutPrefix(example, []byte(localSchemaLine+"\n"))
	if !ok {
		return example
	}
	bi, _ := debug.ReadBuildInfo()
	line := "# yaml-language-server: $schema=" + schemaURL(toolVersion(info.GitTag, bi)) + "\n"
	return append([]byte(line), rest...)
}
