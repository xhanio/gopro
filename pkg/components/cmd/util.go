package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xhanio/framingo/pkg/types/info"
)

func matches(path string, patterns ...string) (bool, error) {
	if len(patterns) == 0 {
		return true, nil
	}
	for _, pattern := range patterns {
		ok, err := filepath.Match(pattern, path)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func infoString(key string, val any) string {
	return fmt.Sprintf("-X github.com/xhanio/framingo/pkg/types/info.%s=%v", key, val)
}

func injectInfo() []string {
	var infos []string
	for key, val := range info.INJECTION {
		infos = append(infos, infoString(key, *val))
	}
	return []string{
		"-ldflags",
		strings.Join(infos, " "),
	}
}
