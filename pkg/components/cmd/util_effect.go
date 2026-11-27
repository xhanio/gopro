package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// dryRun, set by --dry-run on build and generate, prints each operation with a
// side effect, one per line, instead of performing it. Everything else runs as
// usual: git queries, reading and rendering templates, and every check, so a
// dry run fails where the real run would. Only the functions below touch the
// outside world on build and generate's behalf.
var dryRun bool

const dryRunUsage = "print each command and file operation, one per line, instead of performing it"

// run runs a command that changes something, such as a build or a docker
// call, or under --dry-run prints it as a shell line. A query, whose output is
// needed even in a dry run, uses execute directly.
func run(cmd string, args, env []string) error {
	if dryRun {
		fmt.Println(shellLine(env, cmd, args))
		return nil
	}
	_, err := execute(cmd, args, env, true)
	return err
}

func removeAll(path string) error {
	if dryRun {
		fmt.Println(shellLine(nil, "rm", []string{"-rf", path}))
		return nil
	}
	return os.RemoveAll(path)
}

func mkdirAll(path string) error {
	if dryRun {
		fmt.Println(shellLine(nil, "mkdir", []string{"-p", path}))
		return nil
	}
	return os.MkdirAll(path, 0o755)
}

// writeOutput writes a generated file, creating its directory; under
// --dry-run it prints "render <src> <dst>" for a template, "cp <src> <dst>"
// for a file copied as is.
func writeOutput(src, dst string, data []byte, rendered bool) error {
	if dryRun {
		verb := "cp"
		if rendered {
			verb = "render"
		}
		fmt.Println(shellLine(nil, verb, []string{src, dst}))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// shellLine renders a command, with the environment gopro adds to it, as a
// line a POSIX shell runs the same way.
func shellLine(env []string, cmd string, args []string) string {
	words := make([]string, 0, len(env)+1+len(args))
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		words = append(words, k+"="+shellQuote(v))
	}
	words = append(words, shellQuote(cmd))
	for _, a := range args {
		words = append(words, shellQuote(a))
	}
	return strings.Join(words, " ")
}

func shellQuote(s string) string {
	safe := s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_@%+=:,./-", r))
	}) < 0
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
