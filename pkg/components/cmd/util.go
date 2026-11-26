package cmd

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
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

// clearTarget empties dst so generated output reflects only current sources.
//
// A target overlapping a template source is left alone instead. Rendering in
// place is deliberate -- an unset target puts output beside its templates --
// and there the target and the source are the same directory, so clearing it
// would delete the very files about to be read. Stale output from an earlier
// run therefore survives an in-place render; that is the trade the layout
// makes, since the inputs cannot be told apart from the outputs.
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
	return os.RemoveAll(dst)
}

// pathsOverlap reports whether removing either absolute path would affect the
// other, i.e. they are equal or one contains the other.
func pathsOverlap(a, b string) bool {
	if a == b {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(a, b+sep) || strings.HasPrefix(b, a+sep)
}

// injectInfo returns an unquoted "-X", "pkg.Key=value" pair per info var, in
// key order so the command line is reproducible.
func injectInfo() []string {
	keys := slices.Sorted(maps.Keys(info.INJECTION))
	fields := make([]string, 0, 2*len(keys))
	for _, key := range keys {
		fields = append(fields, "-X", fmt.Sprintf("%s.%s=%s", infoPackage, key, *info.INJECTION[key]))
	}
	return fields
}
