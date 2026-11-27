package cmd

import (
	"fmt"

	"github.com/xhanio/gopro/pkg/types"
)

var (
	projectPath string
	project     types.Project

	envName string
	env     types.EnvSpec
)

func loadConfig() error {
	project = types.Project{}
	if err := project.Load(projectPath); err != nil {
		return err
	}
	if err := project.Validate(); err != nil {
		return fmt.Errorf("%s is invalid:\n%w", projectPath, err)
	}
	var err error
	env, err = project.GetEnv(envName)
	return err
}
