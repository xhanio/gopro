package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/tidwall/gjson"

	"github.com/xhanio/framingo/pkg/types/info"
	"github.com/xhanio/framingo/pkg/utils/envutil"
	"github.com/xhanio/gopro/pkg/types"
)

func funcMap() template.FuncMap {
	fm := sprig.TxtFuncMap()
	fm["GetEnvKey"] = GetEnvKey
	fm["GetConfigDir"] = GetConfigDir
	fm["GetImageName"] = GetImageName
	fm["GetImageNameWithTag"] = GetImageNameWithTag
	fm["FromFile"] = FromFile
	fm["FromConfigFile"] = FromConfigFile
	fm["FromConfigJSON"] = FromConfigJSON
	fm["FromSecretEnv"] = FromSecretEnv
	return fm
}

func GetEnvKey(key string) string {
	prefix := envutil.EnvPrefix(info.ProductName)
	if prefix == "" {
		return key
	}
	return fmt.Sprintf("%s_%s", prefix, key)
}

func GetConfigDir(name string) string {
	for _, binary := range project.Build.Binaries {
		if binary.Name == name {
			return binary.ConfigDir
		}
	}
	return ""
}

func GetImageName(name string) string {
	for _, image := range project.Build.Images {
		if image.Name == name {
			return image.GetImageName(env)
		}
	}
	return ""
}

func GetImageNameWithTag(name, tag string) string {
	for _, image := range project.Build.Images {
		if image.Name == name {
			return image.GetImageNameWithTag(env, tag)
		}
	}
	return ""
}

func FromFile(name string) string {
	b, err := os.ReadFile(name)
	if err != nil {
		panic(fmt.Errorf("failed to render from file %s: %s", name, err.Error()))
	}
	return string(b)
}

func FromConfigFile(name, filename string) string {
	b, err := os.ReadFile(filepath.Join(env.ConfigTgt, name, filename))
	if err != nil {
		panic(fmt.Errorf("failed to render from file %s: %s", name, err.Error()))
	}
	return string(b)
}

func FromConfigJSON(name, filename, jsonpath string) string {
	b, err := os.ReadFile(filepath.Join(env.ConfigTgt, name, filename))
	if err != nil {
		panic(fmt.Errorf("failed to render from file %s: %s", name, err.Error()))
	}
	result := gjson.GetBytes(b, jsonpath)
	return result.String()
}

func FromSecretEnv(name, key string) string {
	b, err := os.ReadFile(filepath.Join(env.ConfigSrc, name, "secret.env"))
	if err != nil {
		panic(fmt.Errorf("failed to render from %s secret.env: %s", name, err.Error()))
	}
	scanner := bufio.NewScanner(bytes.NewReader(b))
	kv := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			kv[key] = value
		}
	}
	if val, ok := kv[key]; ok {
		return val
	}
	panic(fmt.Errorf("failed to render from %s secret.env: key %s not found", name, key))
}

type renderContext struct {
	Name    string
	Project types.Project
	EnvName string
	Env     types.EnvSpec
}

func render(name, srcDir, dstDir, prefix string, patterns []string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, er := filepath.Rel(srcDir, path)
		if er != nil {
			return er
		}
		// The template prefix names the file, not the directory holding it, so
		// only the base name is stripped and the rest of the relative path
		// carries through. One output path drives both the pattern match and
		// the write, so a template keeps the subdirectory it was authored in
		// instead of landing in the output root on top of its siblings.
		outRel, templated := rel, false
		if !d.IsDir() {
			if after, ok := strings.CutPrefix(d.Name(), prefix); ok {
				outRel = filepath.Join(filepath.Dir(rel), after)
				templated = true
			}
		}
		ok, er := matches(outRel, patterns...)
		if er != nil {
			return er
		}
		if !ok || d.IsDir() {
			// Directories are not created up front; a file creates its own
			// parents below, so an excluded subtree leaves nothing behind.
			return nil
		}
		b, er := os.ReadFile(path)
		if er != nil {
			return er
		}
		if templated {
			linef("render %s from %s", outRel, path)
			t, er := template.New(d.Name()).Delims("[[", "]]").Funcs(funcMap()).Parse(string(b))
			if er != nil {
				return er
			}
			var buffer bytes.Buffer
			er = t.Execute(&buffer, &renderContext{
				Name:    name,
				Project: project,
				EnvName: envName,
				Env:     env,
			})
			if er != nil {
				return er
			}
			b = buffer.Bytes()
		} else {
			linef("copy %s from %s", outRel, path)
		}
		dstFile := filepath.Join(dstDir, outRel)
		if er := os.MkdirAll(filepath.Dir(dstFile), 0755); er != nil {
			return er
		}
		return os.WriteFile(dstFile, b, 0644)
	})
}
