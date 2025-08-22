package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xhanio/gopro/pkg/types/info"
)

func executeBuildImage(name, src, image, base string) error {
	if verbose {
		debugf("building image %s %s from base %s", src, image, base)
	}
	var args []string
	args = append(args, "build")
	args = append(args, "-t", image)
	args = append(args, "--no-cache")
	args = append(args, "--build-arg", fmt.Sprintf("NAME=%s", name))
	args = append(args, "--build-arg", fmt.Sprintf("BASE=%s", base))
	args = append(args, "--build-arg", fmt.Sprintf("CONFIG_TGT=%s", envConf.ConfigTgt))
	args = append(args, "--build-arg", fmt.Sprintf("CONFIG_DIR=%s", GetConfigDir(name)))
	args = append(args, "-f", filepath.Join(info.ProjectRoot, src, "Dockerfile"))
	args = append(args, info.ProjectRoot)
	if verbose {
		debugf("args: %s", strings.Join(args, " "))
	}
	_, err := execute("docker", args, envConf.ImageBuildEnv, true)
	return err
}

func executePullImage(image string) error {
	linef("pull image %s", image)
	var args []string
	args = append(args, "pull")
	args = append(args, image)
	_, err := execute("docker", args, nil, true)
	return err
}

func executeTagImage(src, tgt string) error {
	linef("tag image from %s to %s", src, tgt)
	var args []string
	args = append(args, "tag")
	args = append(args, src)
	args = append(args, tgt)
	_, err := execute("docker", args, envConf.ImageBuildEnv, true)
	return err
}

func executePushImage(image string) error {
	var args []string
	args = append(args, "push")
	args = append(args, image)
	_, err := execute("docker", args, nil, true)
	return err
}

func execute(cmd string, args []string, env []string, print bool) (string, error) {
	if verbose {
		debugf("executing %s %s", cmd, strings.Join(args, "\n"))
	}
	p := exec.Command(cmd, args...)
	p.Env = os.Environ()
	p.Env = append(p.Env, env...)
	if verbose {
		debugf("env: \n%s", strings.Join(p.Env, "\n"))
	}
	p.Stdin = os.Stdin
	buffer := bytes.NewBuffer([]byte{})
	var w io.Writer
	if print {
		w = io.MultiWriter(os.Stdout, buffer)
	} else {
		w = buffer
	}
	p.Stdout = w
	p.Stderr = os.Stderr
	err := p.Run()
	return buffer.String(), err
}

func executeBuildBinary(name, platform, src, dst string) error {
	var envs []string
	envs = append(envs, envConf.BinaryBuildEnv...)
	if platform != "" {
		parts := strings.Split(platform, "/")
		if len(parts) != 2 {
			return errors.New("unknown platform " + platform)
		}
		name = fmt.Sprintf("%s_%s_%s", name, parts[0], parts[1])
		envs = append(envs, "GOOS="+parts[0])
		envs = append(envs, "GOARCH="+parts[1])
	}
	var args []string
	args = append(args, "build")
	args = append(args, envConf.BinaryBuildArgs...)
	args = append(args, injectInfo()...)
	args = append(args, "-o", filepath.Join(dst, name))
	args = append(args, filepath.Join(info.ProjectRoot, src))
	_, err := execute("go", args, envs, true)
	return err
}
