package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xhanio/gopro/pkg/types"
)

func NewExampleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "example",
		Short: "Save an example project.yaml to the current directory",
		// Replaces the root's pre-run, which would load project.yaml, so it must
		// silence usage itself.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return nil
		},
		RunE: runExample,
	}
	return cmd
}

func runExample(cmd *cobra.Command, args []string) error {
	if dryRun {
		return fmt.Errorf("example has no dry run; run it without --dry-run")
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	target := filepath.Join(wd, "project.yaml")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("project.yaml already exists in %s", wd)
	}
	if err := os.WriteFile(target, withSchemaURL(types.ExampleProjectYAML), 0644); err != nil {
		return err
	}
	linef("example project.yaml saved to %s", target)
	return nil
}
