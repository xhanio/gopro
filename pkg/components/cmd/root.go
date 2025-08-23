package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/xhanio/framingo/pkg/types/info"
)

var (
	help        bool
	verbose     bool
	filter      string
	filterRegex *regexp.Regexp
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		// SilenceErrors: true,
		// SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if help {
				return cmd.Help()
			}
			// load make.yaml to setup conf & envConf
			err := loadConfig()
			if err != nil {
				return err
			}
			// get current workdir
			wd, err := os.Getwd()
			if err != nil {
				return err
			}

			// check if git is available and repository is initialized
			_, err = execute("git", []string{"--version"}, os.Environ(), false)
			if err == nil {
				// git is available, check if current directory is a git repository
				_, err = execute("git", []string{"rev-parse", "--git-dir"}, os.Environ(), false)
				if err == nil {
					// workdir is a git repo, get git info
					branch, err := execute("git", []string{"rev-parse", "--abbrev-ref", "HEAD"}, os.Environ(), false)
					if err == nil {
						info.GitBranch = strings.Trim(branch, " \n\t")
					}
					tag, err := execute("git", []string{"describe", "--tags", "--always"}, os.Environ(), false)
					if err == nil {
						info.GitTag = strings.Trim(tag, " \n\t")
					}
				}
			}

			info.BuildTime = time.Now().Format(time.RFC3339)
			info.ProjectRoot = wd
			info.ProjectName = conf.Project
			info.ProjectPath = strings.Trim(strings.TrimPrefix(wd, filepath.Join(os.Getenv("GOPATH"), "src")), string(filepath.Separator))
			info.ProductName = conf.Product
			info.ProductVersion = conf.Version
			// use default version from git tag
			if info.BuildVersion == "" {
				info.BuildVersion = info.GitTag
			}
			if info.ProductVersion == "" {
				info.ProductVersion = info.BuildVersion
			}
			// compile filter regex
			r, err := regexp.Compile(filter)
			if err != nil {
				return err
			}
			filterRegex = r
			return nil
		},
	}
	root.PersistentFlags().BoolVar(&help, "help", false, "")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")
	root.PersistentFlags().StringVarP(&confPath, "config", "c", "project.yaml", "config file path")
	root.PersistentFlags().StringVarP(&envName, "environment", "e", "", "select an environment to generate for")
	root.PersistentFlags().StringVarP(&filter, "filter", "f", ".*", "filter targets by regex")

	root.AddCommand(NewInitCmd())
	root.AddCommand(NewBuildCmd())
	root.AddCommand(NewGenerateCmd())
	return root
}
