package cmd

import (
	"go/build"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/xhanio/framingo/pkg/types/info"

	"github.com/xhanio/gopro/pkg/types"
)

var (
	verbose     bool
	filter      string
	filterRegex *regexp.Regexp
)

// applyProjectInfo copies project.yaml metadata onto the framingo info package
// vars, which are what the -ldflags injection reads at build time.
func applyProjectInfo(p types.Project) {
	info.ProjectName = p.Module
	info.ProductName = p.Product
	info.ProductModel = p.Model
	info.ProductVersion = p.Version
	// use default version from git tag
	if info.BuildVersion == "" {
		info.BuildVersion = info.GitTag
	}
	if info.ProductVersion == "" {
		info.ProductVersion = info.BuildVersion
	}
}

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "gopro",
		Short: "Build, package, and render multi-environment Go projects",
		// SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Flags parsed, so later errors are not usage mistakes.
			cmd.SilenceUsage = true
			// skip config loading for version command
			if cmd.Name() == "version" {
				return nil
			}
			// load project.yaml to setup project & env
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
			_, err = execute("git", []string{"--version"}, nil, false)
			if err == nil {
				// git is available, try to get branch info
				branch, err := execute("git", []string{"rev-parse", "--abbrev-ref", "HEAD"}, nil, false)
				if err == nil {
					info.GitBranch = strings.Trim(branch, " \n\t")
				}
				// try to get tag info
				tag, err := execute("git", []string{"describe", "--tags", "--always"}, nil, false)
				if err == nil {
					info.GitTag = strings.Trim(tag, " \n\t")
				}
				// try to get commit info
				commit, err := execute("git", []string{"rev-parse", "HEAD"}, nil, false)
				if err == nil {
					info.GitCommit = strings.Trim(commit, " \n\t")
				}
			}

			info.BuildTime = time.Now().Format(time.RFC3339)
			info.ProjectRoot = wd
			info.ProjectPath = resolveProjectPath(wd, filepath.SplitList(build.Default.GOPATH), project.Module)
			applyProjectInfo(project)
			// compile filter regex
			r, err := regexp.Compile(filter)
			if err != nil {
				return err
			}
			filterRegex = r
			return nil
		},
	}
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")
	root.PersistentFlags().StringVarP(&projectPath, "config", "c", "project.yaml", "config file path")
	root.PersistentFlags().StringVarP(&envName, "environment", "e", "", "select an environment to generate for")
	root.PersistentFlags().StringVarP(&filter, "filter", "f", ".*", "filter targets by regex")

	root.AddCommand(NewInitCmd())
	root.AddCommand(NewBuildCmd())
	root.AddCommand(NewGenerateCmd())
	root.AddCommand(NewExampleCmd())
	root.AddCommand(NewSchemaCmd())
	root.AddCommand(NewVersionCmd())
	return root
}

// resolveProjectPath returns the path framingo's nameutil trims import paths
// with: wd relative to a GOPATH/src containing it, else the module path, which
// overlaps import paths the same way. Empty leaves names untrimmed.
func resolveProjectPath(wd string, gopaths []string, module string) string {
	for _, gopath := range gopaths {
		if gopath == "" {
			continue
		}
		rel, err := filepath.Rel(filepath.Join(gopath, "src"), wd)
		if err == nil && rel != "." && filepath.IsLocal(rel) {
			return filepath.ToSlash(rel)
		}
	}
	return module
}
