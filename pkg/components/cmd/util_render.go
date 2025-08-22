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

	"github.com/xhanio/gopro/pkg/types"
	"github.com/xhanio/gopro/pkg/types/info"
)

func funcMap() template.FuncMap {
	fm := sprig.TxtFuncMap()
	fm["GetEnvKey"] = GetEnvKey
	fm["GetConfigDir"] = GetConfigDir
	fm["GetImageName"] = GetImageName
	fm["FromFile"] = FromFile
	fm["FromConfigFile"] = FromConfigFile
	fm["FromConfigJSON"] = FromConfigJSON
	fm["FromSecretEnv"] = FromSecretEnv
	return fm
}

func GetEnvKey(key string) string {
	prefix := info.EnvPrefix(info.ProductName)
	if prefix == "" {
		return key
	}
	return fmt.Sprintf("%s_%s", prefix, key)
}

func GetConfigDir(name string) string {
	for _, binary := range conf.Build.Binaries {
		if binary.Name == name {
			return binary.ConfigDir
		}
	}
	return ""
}

func GetImageName(name string) string {
	for _, image := range conf.Build.Images {
		if image.Name == name {
			return image.GetImageName(envConf)
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
	b, err := os.ReadFile(filepath.Join(envConf.ConfigTgt, name, filename))
	if err != nil {
		panic(fmt.Errorf("failed to render from file %s: %s", name, err.Error()))
	}
	return string(b)
}

func FromConfigJSON(name, filename, jsonpath string) string {
	b, err := os.ReadFile(filepath.Join(envConf.ConfigTgt, name, filename))
	if err != nil {
		panic(fmt.Errorf("failed to render from file %s: %s", name, err.Error()))
	}
	result := gjson.GetBytes(b, jsonpath)
	return result.String()
}

func FromSecretEnv(name, key string) string {
	b, err := os.ReadFile(filepath.Join(envConf.ConfigTgt, name, "secret.env"))
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
	Name   string
	Config types.Config
	Env    types.EnvConfig
}

func render(name, srcDir, dstDir, prefix string, patterns []string) error {
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		srcFile, er := filepath.Rel(srcDir, path)
		if er != nil {
			return er
		}
		dstFile := filepath.Join(dstDir, srcFile)
		if after, ok := strings.CutPrefix(d.Name(), prefix); ok {
			srcFile = filepath.Join(filepath.Dir(srcFile), after)
		}
		ok, er := matches(srcFile, patterns...)
		if er != nil {
			return er
		}
		if !ok {
			return nil
		}
		er = os.MkdirAll(filepath.Dir(dstFile), 0755)
		if er != nil {
			return er
		}
		if d.IsDir() {
			return nil
		}
		b, er := os.ReadFile(path)
		if er != nil {
			return er
		}
		if after, ok := strings.CutPrefix(d.Name(), prefix); ok {
			dstFile = filepath.Join(dstDir, after)
			linef("render %s from %s", after, path)
			t, er := template.New(d.Name()).Delims("[[", "]]").Funcs(funcMap()).Parse(string(b))
			if er != nil {
				return er
			}
			var buffer bytes.Buffer
			er = t.Execute(&buffer, &renderContext{
				Name:   name,
				Config: conf,
				Env:    envConf,
			})
			if er != nil {
				return er
			}
			b = buffer.Bytes()
		} else {
			linef("copy %s from %s", d.Name(), path)
		}
		er = os.WriteFile(dstFile, b, 0644)
		if er != nil {
			return er
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
