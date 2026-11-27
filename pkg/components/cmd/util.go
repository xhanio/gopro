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

// A target overlapping a template source is an in-place render: clearing it
// would delete the templates, so stale output survives there.
func clearTarget(dst string, srcs ...string) error {
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	for _, src := range srcs {
		if src == "" {
			continue
		}
		absSrc, err := filepath.Abs(src)
		if err != nil {
			return err
		}
		if pathsOverlap(absDst, absSrc) {
			if verbose {
				debugf("rendering in place into %s; leaving existing files alone", dst)
			}
			return nil
		}
	}
	return removeAll(dst)
}

func pathsOverlap(a, b string) bool {
	if a == b {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(a, b+sep) || strings.HasPrefix(b, a+sep)
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
