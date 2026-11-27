package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xhanio/framingo/pkg/types/info"
	"github.com/xhanio/framingo/pkg/utils/envutil"

	"github.com/xhanio/gopro/pkg/types"
)

func executeBuildImage(name, src, image, base string, cache bool) error {
	if verbose {
		debugf("building image %s %s from base %s", src, image, base)
	}
	args, err := dockerBuildArgs(name, src, image, base, cache, env.ImageBuildArgs)
	if err != nil {
		return fmt.Errorf("image %s: %w", name, err)
	}
	if verbose {
		debugf("args: %s", strings.Join(args, " "))
	}
	return run("docker", args, env.ImageBuildEnv)
}

var injectedBuildArgs = []string{"NAME", "BASE", "CONFIG_TGT", "CONFIG_DIR"}

// -f and the context stay last; a stray positional is left to docker, which
// rejects a second context.
func dockerBuildArgs(name, src, image, base string, cache bool, user []string) ([]string, error) {
	if err := checkReservedDockerArgs(user); err != nil {
		return nil, err
	}
	var args []string
	args = append(args, "build")
	args = append(args, "-t", image)
	if !cache {
		args = append(args, "--no-cache")
	}
	args = append(args, "--build-arg", fmt.Sprintf("NAME=%s", name))
	args = append(args, "--build-arg", fmt.Sprintf("BASE=%s", base))
	args = append(args, "--build-arg", fmt.Sprintf("CONFIG_TGT=%s", env.ConfigTgt))
	args = append(args, "--build-arg", fmt.Sprintf("CONFIG_DIR=%s", GetConfigDir(name)))
	args = append(args, user...)
	args = append(args, "-f", filepath.Join(info.ProjectRoot, src, "Dockerfile"))
	args = append(args, info.ProjectRoot)
	return args, nil
}

// A second -f would replace the Dockerfile, a second -t add a tag --push never
// pushes, and a repeated injected build arg override gopro's.
func checkReservedDockerArgs(user []string) error {
	for i := 0; i < len(user); i++ {
		arg := user[i]
		switch {
		case arg == "--tag" || strings.HasPrefix(arg, "--tag=") || isShortFlag(arg, 't'):
			return errors.New("image build args may not set -t: gopro tags the image, use the image's repo, prefix and tag")
		case arg == "--file" || strings.HasPrefix(arg, "--file=") || isShortFlag(arg, 'f'):
			return errors.New("image build args may not set -f: gopro uses {build_src}/Dockerfile")
		case arg == "--build-arg" || strings.HasPrefix(arg, "--build-arg="):
			def, ok := strings.CutPrefix(arg, "--build-arg=")
			if !ok {
				if i+1 >= len(user) {
					return errors.New("image build arg --build-arg needs a value")
				}
				i++
				def = user[i]
			}
			key, _, _ := strings.Cut(def, "=")
			if slices.Contains(injectedBuildArgs, key) {
				return fmt.Errorf("image build args may not set --build-arg %s: gopro injects %s", key, strings.Join(injectedBuildArgs, ", "))
			}
		}
	}
	return nil
}

func isShortFlag(arg string, c byte) bool {
	return len(arg) >= 2 && arg[0] == '-' && arg[1] == c
}

func executePullImage(image string) error {
	linef("pull image %s", image)
	var args []string
	args = append(args, "pull")
	args = append(args, image)
	return run("docker", args, nil)
}

func executeTagImage(src, tgt string) error {
	linef("tag image from %s to %s", src, tgt)
	var args []string
	args = append(args, "tag")
	args = append(args, src)
	args = append(args, tgt)
	return run("docker", args, env.ImageBuildEnv)
}

func executePushImage(image string) error {
	var args []string
	args = append(args, "push")
	args = append(args, image)
	return run("docker", args, nil)
}

// -v prints env, so pass only additions, never os.Environ(). No stdin, so a
// command that would prompt fails instead of blocking.
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

// Args replace rather than merge: go build flags are positional and
// repeatable. Only an unset (nil) level inherits, so [] means no args.
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

// A zero PlatformSpec builds for the host.
func executeBuildBinary(binary types.BinarySpec, platform types.PlatformSpec, src, dst string) error {
	name := binary.Name
	// Each level overrides only the variables it names.
	envs := envutil.Merge(env.BinaryBuildEnv, binary.BuildEnv, platform.Env)
	if platform.Name != "" {
		parts := strings.Split(platform.Name, "/")
		if len(parts) != 2 {
			return errors.New("unknown platform " + platform.Name)
		}
		name = fmt.Sprintf("%s_%s_%s", name, parts[0], parts[1])
		envs = envutil.Merge(envs, []string{"GOOS=" + parts[0], "GOARCH=" + parts[1]})
	}
	buildArgs, err := assembleBuildArgs(buildArgsFor(env, binary, platform), injectInfo())
	if err != nil {
		return fmt.Errorf("binary %s: %w", binary.Name, err)
	}
	var args []string
	args = append(args, "build")
	args = append(args, buildArgs...)
	args = append(args, "-o", filepath.Join(dst, name))
	args = append(args, filepath.Join(info.ProjectRoot, src))
	return run("go", args, envs)
}
