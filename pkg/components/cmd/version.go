package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/xhanio/framingo/pkg/types/info"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			bi, _ := debug.ReadBuildInfo()
			buildTime := info.BuildTime
			if buildTime == "" {
				buildTime = "unknown"
			}
			fmt.Printf("Version:    %s\n", toolVersion(info.GitTag, bi))
			fmt.Printf("Build Time: %s\n", buildTime)
			return nil
		},
	}
}

// toolVersion returns gopro's own version: the tag injected when gopro is
// built with gopro, else the module version the go command recorded (set by
// go install, "(devel)" for a local build).
func toolVersion(injected string, bi *debug.BuildInfo) string {
	if injected != "" {
		return injected
	}
	if bi != nil && bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "unknown"
}
