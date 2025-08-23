package cmd

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/monochromegane/go-gitignore"
	"github.com/spf13/cobra"

	"github.com/xhanio/framingo/pkg/utils/errors"
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
	titlef("initializing project directories")

	// initialize git repository if not already initialized
	_, err := execute("git", []string{"rev-parse", "--git-dir"}, os.Environ(), false)
	if err != nil {
		linef("initializing git repository")
		_, err = execute("git", []string{"init"}, os.Environ(), false)
		if err != nil {
			return err
		}
	}

	// initialize go module if go.mod doesn't exist
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		linef("initializing go module")
		args := []string{"mod", "init"}
		if conf.Project != "" {
			args = append(args, conf.Project)
		}
		_, err = execute("go", args, os.Environ(), false)
		if err != nil {
			return err
		}
	}

	// create directories from default configuration
	if err := createEnvDirectories("default", conf.Default); err != nil {
		return err
	}
	if envName != "" {
		// create directories from given environment configurations
		if err := createEnvDirectories(envName, envConf); err != nil {
			return err
		}
	} else {
		// create directories from all environment configurations
		for env, config := range conf.Env {
			if err := createEnvDirectories(env, config); err != nil {
				return err
			}
		}
	}
	// create or update .gitignore file
	if err := createOrUpdateGitignore(); err != nil {
		return err
	}
	return nil
}

// createEnvDirectories creates all source and target directories for a given environment
func createEnvDirectories(env string, config types.EnvConfig) error {
	// create all directory paths from the environment config
	resources := map[types.ResourceType][]string{
		types.ResourceTypeBinaries:   {config.BinarySrc, config.BinaryTgt},
		types.ResourceTypeConfigs:    {config.ConfigSrc, config.ConfigTgt},
		types.ResourceTypeImages:     {config.ImageBuildSrc},
		types.ResourceTypeKubernetes: {config.KubernetesSrc, config.KubernetesTgt},
	}
	var targets []string
	for res, dirs := range resources {
		for i, dir := range dirs {
			targets = append(targets, dir)
			switch res {
			case types.ResourceTypeBinaries:
				if i == 1 {
					// !! no need to create resource dirs under config.BinaryTgt
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

// createDirectoryIfNotExists creates a directory if it doesn't already exist
func createDirectoryIfNotExists(env, dirPath string) error {
	if dirPath == "" {
		return nil
	}
	// convert to absolute path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}
	// check if directory already exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return err
		}
		linef("create directories %s for environment %s successfully", dirPath, env)
	}
	return nil
}

// createOrUpdateGitignore creates .gitignore file if it doesn't exist, or updates it with missing entries
func createOrUpdateGitignore() error {
	gitignorePath := ".gitignore"
	requiredEntries := map[string]bool{
		"bin/":       true,
		"dist/":      true,
		"test/":      true,
		"secret.env": false,
	}
	linef("managing .gitignore file")
	// check if .gitignore exists, create if it doesn't
	if _, err := os.Stat(gitignorePath); err != nil {
		// file doesn't exist, create an empty one
		file, err := os.Create(gitignorePath)
		if err != nil {
			return err
		}
		file.Close()
		debugf(".gitignore file created")
	}
	// now parse the .gitignore file (whether it was just created or already existed)
	ignore, err := gitignore.NewGitIgnore(gitignorePath)
	if err != nil {
		return err
	}
	// find missing entries by checking if they would be ignored
	var missingEntries []string
	for requiredEntry, isDir := range requiredEntries {
		// use the Ignore() method to check if this entry would be matched
		// if it's not ignored by existing patterns, we need to add it
		if !ignore.Match(requiredEntry, isDir) {
			missingEntries = append(missingEntries, requiredEntry)
		}
	}
	// add missing entries if any
	if len(missingEntries) == 0 {
		debugf("all required entries are already covered by .gitignore")
		return nil
	}
	file, err := os.OpenFile(gitignorePath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
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
