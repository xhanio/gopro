package cmd

import (
	"os"
	"path/filepath"

	"go.uber.org/config"
	"golang.org/x/mod/modfile"

	"github.com/xhanio/gopro/pkg/types"
)

var (
	confPath string
	conf     types.Config

	envName string
	envConf types.EnvConfig
)

func loadConfig() error {
	p, err := config.NewYAML(config.File(confPath))
	if err != nil {
		return err
	}
	err = p.Get(config.Root).Populate(&conf)
	if err != nil {
		return err
	}
	envConf = conf.GetEnv(envName)
	goModPath := filepath.Join(filepath.Dir(confPath), "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil
	}
	if conf.Project == "" {
		// load module path from go.mod as conf.Project
		mb, err := os.ReadFile(goModPath)
		if err != nil {
			return err
		}
		conf.Project = modfile.ModulePath(mb)
	}
	return nil
}
