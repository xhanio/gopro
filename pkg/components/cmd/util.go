package cmd

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xhanio/framingo/pkg/types/info"
)

// A pattern without a separator also matches the base name: filepath.Match's
// * never crosses a separator, so "*.yaml" would otherwise miss nested files.
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

// overlaps reports whether removing either path would affect the other.
func overlaps(a, b string) (bool, error) {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return within(absA, absB) || within(absB, absA), nil
}

// within reports whether the absolute path is dir or lies under it.
func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && filepath.IsLocal(rel)
}

// Sorted, since info.INJECTION is a map; assembleBuildArgs quotes the values.
func injectInfo() []string {
	keys := slices.Sorted(maps.Keys(info.INJECTION))
	fields := make([]string, 0, 2*len(keys))
	for _, key := range keys {
		fields = append(fields, "-X", fmt.Sprintf("%s.%s=%s", infoPackage, key, *info.INJECTION[key]))
	}
	return fields
}
