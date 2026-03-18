package cmd

import (
	"os"
	"path/filepath"

	"go.uber.org/config"
	"golang.org/x/mod/modfile"

	"github.com/xhanio/gopro/pkg/types"
)

var (
	projectPath string
	project     types.Project

	envName string
	env     types.EnvSpec
)

func loadConfig() error {
	p, err := config.NewYAML(config.File(projectPath))
	if err != nil {
		return err
	}
	err = p.Get(config.Root).Populate(&project)
	if err != nil {
		return err
	}
	env = project.GetEnv(envName)
	goModPath := filepath.Join(filepath.Dir(projectPath), "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil
	}
	if project.Module == "" {
		// load module path from go.mod as project.Module
		mb, err := os.ReadFile(goModPath)
		if err != nil {
			return err
		}
		project.Module = modfile.ModulePath(mb)
	}
	return nil
}
