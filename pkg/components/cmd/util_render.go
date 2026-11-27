package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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

// never written to output; FromSecretEnv reads it from the source
const secretEnvFile = "secret.env"

func FromSecretEnv(name, key string) string {
	b, err := os.ReadFile(filepath.Join(env.ConfigSrc, name, secretEnvFile))
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

// output is a file generation writes, by its path relative to the target.
type output struct {
	src, rel string
	data     []byte
	rendered bool
}

// renderLayer adds the files srcDir renders to outs, replacing a file an
// earlier layer gave the same path.
func renderLayer(outs []output, name, srcDir, prefix string, patterns []string) ([]output, error) {
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, er := filepath.Rel(srcDir, path)
		if er != nil {
			return er
		}
		// The prefix names the file, so only the base name loses it and a template
		// keeps its subdirectory.
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
		if filepath.Base(outRel) == secretEnvFile {
			// even when files matches it
			ok = false
		}
		if !ok || d.IsDir() {
			// A file creates its own parents, so an excluded subtree leaves nothing.
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
		out := output{src: path, rel: outRel, data: b, rendered: templated}
		if i := slices.IndexFunc(outs, func(o output) bool { return o.rel == outRel }); i >= 0 {
			outs[i] = out
		} else {
			outs = append(outs, out)
		}
		return nil
	})
	return outs, err
}

func writeOutputs(dstDir string, outs []output) error {
	for _, o := range outs {
		if err := writeOutput(o.src, filepath.Join(dstDir, o.rel), o.data, o.rendered); err != nil {
			return err
		}
	}
	return nil
}
