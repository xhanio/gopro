package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/xhanio/framingo/pkg/types/info"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			// fallback to git when build info wasn't injected via ldflags
			if info.GitTag == "" {
				if tag, err := execute("git", []string{"describe", "--tags", "--always"}, os.Environ(), false); err == nil {
					info.GitTag = strings.Trim(tag, " \n\t")
				}
			}
			fmt.Printf("Version:  %s\n", info.GitTag)
			fmt.Printf("Build Time: %s\n", time.Now())
			return nil
		},
	}
}
