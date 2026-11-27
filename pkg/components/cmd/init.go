package cmd

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/monochromegane/go-gitignore"
	"github.com/spf13/cobra"

	"github.com/xhanio/errors"
	"github.com/xhanio/framingo/pkg/utils/sliceutil"
	"github.com/xhanio/gopro/pkg/types"
)

func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "init",
		RunE: runInitProject,
	}
	return cmd
}

func runInitProject(cmd *cobra.Command, args []string) error {
	if dryRun {
		return errors.Newf("init has no dry run; run it without --dry-run")
	}
	if project.Product == "" {
		return errors.Newf("product name is required in project.yaml configuration")
	}

	titlef("initializing project directories")

	_, err := execute("git", []string{"rev-parse", "--git-dir"}, nil, false)
	if err != nil {
		linef("initializing git repository")
		_, err = execute("git", []string{"init"}, nil, false)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		linef("initializing go module")
		args := []string{"mod", "init"}
		if project.Module != "" {
			args = append(args, project.Module)
		}
		_, err = execute("go", args, nil, false)
		if err != nil && project.Module == "" {
			// execute discards go's message; outside GOPATH an unset module is the usual
			// cause.
			return errors.Newf("go mod init: %s; set module in project.yaml, go could not infer a module path here", err)
		}
		if err != nil {
			return err
		}
	}

	if err := createEnvDirectories("default", project.Default); err != nil {
		return err
	}
	if envName != "" {
		if err := createEnvDirectories(envName, env); err != nil {
			return err
		}
	} else {
		for envKey, cfg := range project.Env {
			if err := createEnvDirectories(envKey, cfg); err != nil {
				return err
			}
		}
	}
	if err := createOrUpdateGitignore(); err != nil {
		return err
	}
	return createOrUpdateDockerignore()
}

func createEnvDirectories(env string, config types.EnvSpec) error {
	resources := map[types.ResourceType][]string{
		types.ResourceTypeBinaries:      {config.BinarySrc, config.BinaryTgt},
		types.ResourceTypeConfigs:       {config.ConfigSrc, config.ConfigTgt},
		types.ResourceTypeImages:        {config.ImageBuildSrc},
		types.ResourceTypeKubernetes:    {config.KubernetesSrc, config.KubernetesTgt},
		types.ResourceTypeDockerCompose: {config.DockerComposeSrc, config.DockerComposeTgt},
	}
	var targets []string
	for res, dirs := range resources {
		for i, dir := range dirs {
			targets = append(targets, dir)
			switch res {
			case types.ResourceTypeBinaries:
				if i == 1 {
					// the target holds the binaries, not a directory per binary
					continue
				}
				for _, target := range config.Binaries {
					targets = append(targets, filepath.Join(dir, target))
				}
			case types.ResourceTypeConfigs:
				for _, target := range config.Configs {
					targets = append(targets, filepath.Join(dir, target))
				}
			case types.ResourceTypeImages:
				for _, target := range config.Images {
					targets = append(targets, filepath.Join(dir, target))
				}
			case types.ResourceTypeKubernetes:
				for _, target := range config.KubernetesTemplates {
					targets = append(targets, filepath.Join(dir, target))
				}
			case types.ResourceTypeDockerCompose:
				// docker-compose has no components
				continue
			}
		}
	}
	targets = sliceutil.Deduplicate(targets...)
	slices.Sort(targets)
	for _, target := range targets {
		if err := createDirectoryIfNotExists(env, target); err != nil {
			return err
		}
	}
	return nil
}

func createDirectoryIfNotExists(env, dirPath string) error {
	if dirPath == "" {
		return nil
	}
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return err
		}
		linef("create directories %s for environment %s successfully", dirPath, env)
	}
	return nil
}

func createOrUpdateGitignore() error {
	gitignorePath := ".gitignore"
	// A slice, not a map, so entries append in a stable order.
	requiredEntries := []struct {
		path  string
		isDir bool
	}{
		{"bin/", true},
		{"dist/", true},
		{"test/", true},
		{secretEnvFile, false},
	}
	linef("managing .gitignore file")
	if _, err := os.Stat(gitignorePath); err != nil {
		file, err := os.Create(gitignorePath)
		if err != nil {
			return err
		}
		file.Close()
		debugf(".gitignore file created")
	}
	ignore, err := gitignore.NewGitIgnore(gitignorePath)
	if err != nil {
		return err
	}
	var missingEntries []string
	for _, required := range requiredEntries {
		if !ignore.Match(required.path, required.isDir) {
			missingEntries = append(missingEntries, required.path)
		}
	}
	if len(missingEntries) == 0 {
		debugf("all required entries are already covered by .gitignore")
		return nil
	}
	existing, err := os.ReadFile(gitignorePath)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(gitignorePath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	// Without a final newline, the first entry would fuse onto the last rule.
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		if _, err := file.WriteString("\n"); err != nil {
			return err
		}
	}
	var errs []error
	for _, entry := range missingEntries {
		_, err := file.WriteString(entry + "\n")
		if err != nil {
			errs = append(errs, err)
		} else {
			linef("added to .gitignore: %s", entry)
		}
	}
	return errors.Combine(errs...)
}
