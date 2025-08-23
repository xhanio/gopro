package cmd

import (
	"os"
	"path/filepath"

	"github.com/denormal/go-gitignore"
	"github.com/spf13/cobra"

	"github.com/xhanio/framingo/pkg/utils/errors"
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
	titlef("Initializing project directories")
	// Create directories from default configuration
	if err := createEnvDirectories("default", conf.Default); err != nil {
		return err
	}
	// Create directories from all environment configurations
	for envName, envConfig := range conf.Env {
		if err := createEnvDirectories(envName, envConfig); err != nil {
			return err
		}
	}
	// Create or update .gitignore file
	if err := createOrUpdateGitignore(); err != nil {
		return err
	}
	titlef("Project initialization completed")
	return nil
}

// createEnvDirectories creates all source and target directories for a given environment
func createEnvDirectories(envName string, env types.EnvConfig) error {
	linef("Creating directories for environment: %s", envName)
	// Create all directory paths from the environment config
	dirs := []string{
		env.ConfigSrc,
		env.ConfigTgt,
		env.BinarySrc,
		env.BinaryTgt,
		env.ImageBuildSrc,
		env.KubernetesSrc,
		env.KubernetesTgt,
	}
	// Filter out empty directory paths and create them
	for _, dir := range dirs {
		if dir != "" {
			if err := createDirectoryIfNotExists(dir); err != nil {
				return err
			}
		}
	}
	return nil
}

// createDirectoryIfNotExists creates a directory if it doesn't already exist
func createDirectoryIfNotExists(dirPath string) error {
	// Convert to absolute path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}
	// Check if directory already exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		debugf("Creating directory: %s", absPath)
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return err
		}
		linef("Created directory: %s", dirPath)
	}
	return nil
}

// createOrUpdateGitignore creates .gitignore file if it doesn't exist, or updates it with missing entries
func createOrUpdateGitignore() error {
	gitignorePath := ".gitignore"
	requiredEntries := []string{
		"bin/",
		"dist/",
		"test/",
		"secret.env",
	}
	linef("Managing .gitignore file")
	// Check if .gitignore exists, create if it doesn't
	if _, err := os.Stat(gitignorePath); err != nil {
		// File doesn't exist, create an empty one
		file, err := os.Create(gitignorePath)
		if err != nil {
			return err
		}
		file.Close()
		debugf(".gitignore file created")
	}
	// Now parse the .gitignore file (whether it was just created or already existed)
	ignore, err := gitignore.NewFromFile(gitignorePath)
	if err != nil {
		return err
	}
	// Find missing entries by checking if they would be ignored
	var missingEntries []string
	for _, required := range requiredEntries {
		// Use the Ignore() method to check if this entry would be matched
		// If it's not ignored by existing patterns, we need to add it
		if !ignore.Ignore(required) {
			missingEntries = append(missingEntries, required)
		}
	}
	// Add missing entries if any
	if len(missingEntries) == 0 {
		debugf("All required entries are already covered by .gitignore")
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
			linef("Added to .gitignore: %s", entry)
		}
	}
	return errors.Combine(errs...)
}
