package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xhanio/framingo/pkg/types/info"
)

// matches reports whether the relative path is selected by any pattern. A
// pattern holding a separator is matched against the whole path, so "cert/*"
// stays scoped to cert/. One without is also matched against the base name,
// because filepath.Match's * never crosses a separator: "*.yaml" names a kind
// of file rather than a depth, and matching only the full path would silently
// drop every nested file.
func matches(path string, patterns ...string) (bool, error) {
	if len(patterns) == 0 {
		return true, nil
	}
	for _, pattern := range patterns {
		ok, err := filepath.Match(pattern, path)
		if err != nil {
			return false, err
		}
		if !ok && !strings.ContainsRune(pattern, filepath.Separator) {
			ok, err = filepath.Match(pattern, filepath.Base(path))
			if err != nil {
				return false, err
			}
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
