package cmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/moby/patternmatcher"
	"github.com/moby/patternmatcher/ignorefile"
)

// The build context is the project root. Only secret.env is excluded:
// Dockerfiles copy from bin/ and dist/.
const dockerignoreSecretRule = "**/" + secretEnvFile

func createOrUpdateDockerignore() error {
	const path = ".dockerignore"
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	patterns, err := ignorefile.ReadAll(strings.NewReader(string(existing)))
	if err != nil {
		return err
	}
	exposed, err := unexcluded([]string{secretEnvFile, "env/default/config/x/" + secretEnvFile}, patterns)
	if err != nil {
		return err
	}
	if len(exposed) == 0 {
		debugf("secret.env is already excluded by .dockerignore")
		return nil
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	add := dockerignoreSecretRule + "\n"
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		add = "\n" + add
	}
	if _, err := file.WriteString(add); err != nil {
		return err
	}
	linef("added to .dockerignore: %s", dockerignoreSecretRule)
	return nil
}

func findSecretEnvFiles(root string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.IsDir() && d.Name() == secretEnvFile {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			found = append(found, filepath.ToSlash(rel))
		}
		return nil
	})
	slices.Sort(found)
	return found, err
}

func unexcluded(files, patterns []string) ([]string, error) {
	pm, err := patternmatcher.New(patterns)
	if err != nil {
		return nil, err
	}
	var kept []string
	for _, f := range files {
		excluded, err := pm.MatchesOrParentMatches(f)
		if err != nil {
			return nil, err
		}
		if !excluded {
			kept = append(kept, f)
		}
	}
	return kept, nil
}

// BuildKit reads <dockerfile>.dockerignore in place of the root's.
func dockerignoreFor(root, dockerfile string) ([]string, error) {
	for _, path := range []string{dockerfile + ".dockerignore", filepath.Join(root, ".dockerignore")} {
		f, err := os.Open(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return ignorefile.ReadAll(f)
	}
	return nil, nil
}

// Warns without failing: the Dockerfile may copy none of them.
func warnSecretsInContext(secrets []string, root, dockerfile string) error {
	patterns, err := dockerignoreFor(root, dockerfile)
	if err != nil {
		return err
	}
	exposed, err := unexcluded(secrets, patterns)
	if err != nil {
		return err
	}
	if len(exposed) > 0 {
		warnf("the build context includes %s, which .dockerignore does not exclude; add %s to it (gopro init does)",
			strings.Join(exposed, ", "), dockerignoreSecretRule)
	}
	return nil
}
