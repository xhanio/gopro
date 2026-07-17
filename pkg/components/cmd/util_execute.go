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

	"github.com/xhanio/framingo/pkg/types/info"
	"github.com/xhanio/framingo/pkg/utils/envutil"

	"github.com/xhanio/gopro/pkg/types"
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
	args = append(args, "--build-arg", fmt.Sprintf("CONFIG_TGT=%s", env.ConfigTgt))
	args = append(args, "--build-arg", fmt.Sprintf("CONFIG_DIR=%s", GetConfigDir(name)))
	args = append(args, "-f", filepath.Join(info.ProjectRoot, src, "Dockerfile"))
	args = append(args, info.ProjectRoot)
	if verbose {
		debugf("args: %s", strings.Join(args, " "))
	}
	_, err := execute("docker", args, env.ImageBuildEnv, true)
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
	_, err := execute("docker", args, env.ImageBuildEnv, true)
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
	if len(env) > 0 && verbose {
		debugf("env: \n%s", strings.Join(env, "\n"))
	}
	p.Stdin = os.Stdin
	buffer := bytes.NewBuffer([]byte{})
	var ow, ew io.Writer
	if print {
		ow = io.MultiWriter(os.Stdout, buffer)
		ew = os.Stderr
	} else {
		ow = buffer
		ew = io.Discard
	}
	p.Stdout = ow
	p.Stderr = ew
	err := p.Run()
	return buffer.String(), err
}

// buildArgsFor returns the most specific build args declared. Unlike build env
// these replace rather than merge: go build flags are positional and
// repeatable, so a key-wise merge cannot tell an override from an
// accumulation. Only an unset level inherits, so an empty list is a way to
// build with no arguments at all -- which is why this tests nil rather than
// length, and why sliceutil.Last can't stand in ([]string isn't comparable).
func buildArgsFor(e types.EnvSpec, binary types.BinarySpec, platform types.PlatformSpec) []string {
	args := e.BinaryBuildArgs
	if binary.BuildArgs != nil {
		args = binary.BuildArgs
	}
	if platform.Args != nil {
		args = platform.Args
	}
	return args
}

// executeBuildBinary builds one binary for one platform. A zero PlatformSpec
// builds for the host, inheriting everything and pinning no GOOS/GOARCH.
func executeBuildBinary(binary types.BinarySpec, platform types.PlatformSpec, src, dst string) error {
	name := binary.Name
	// Each level overrides only the variables it names, inheriting the rest.
	envs := envutil.Merge(env.BinaryBuildEnv, binary.BuildEnv, platform.Env)
	if platform.Name != "" {
		parts := strings.Split(platform.Name, "/")
		if len(parts) != 2 {
			return errors.New("unknown platform " + platform.Name)
		}
		name = fmt.Sprintf("%s_%s_%s", name, parts[0], parts[1])
		// The platform being built for outranks any GOOS/GOARCH in build_env.
		envs = envutil.Merge(envs, []string{"GOOS=" + parts[0], "GOARCH=" + parts[1]})
	}
	var args []string
	args = append(args, "build")
	args = append(args, buildArgsFor(env, binary, platform)...)
	args = append(args, injectInfo()...)
	args = append(args, "-o", filepath.Join(dst, name))
	args = append(args, filepath.Join(info.ProjectRoot, src))
	_, err := execute("go", args, envs, true)
	return err
}
