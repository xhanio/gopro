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
	if conf.Project == "" {
		mb, err := os.ReadFile(filepath.Join(filepath.Dir(confPath), "go.mod"))
		if err != nil {
			return err
		}
		conf.Project = modfile.ModulePath(mb)
	}
	envConf = conf.GetEnv(envName)
	return nil
}
