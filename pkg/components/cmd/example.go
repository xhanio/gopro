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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: runExample,
	}
	return cmd
}

func runExample(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	target := filepath.Join(wd, "project.yaml")
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("project.yaml already exists in %s", wd)
	}
	if err := os.WriteFile(target, types.ExampleProjectYAML, 0644); err != nil {
		return err
	}
	linef("example project.yaml saved to %s", target)
	return nil
}
