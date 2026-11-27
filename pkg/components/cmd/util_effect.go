package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Every side effect of build and generate goes through perform.
func perform(env []string, name string, args []string, op func() error) error {
	if dryRun {
		fmt.Println(shellLine(env, name, args))
		return nil
	}
	return op()
}

// Queries, needed in a dry run too, call execute directly.
func run(cmd string, args, env []string) error {
	return perform(env, cmd, args, func() error {
		_, err := execute(cmd, args, env, true)
		return err
	})
}

func removeAll(path string) error {
	return perform(nil, "rm", []string{"-rf", path}, func() error {
		return os.RemoveAll(path)
	})
}

func mkdirAll(path string) error {
	return perform(nil, "mkdir", []string{"-p", path}, func() error {
		return os.MkdirAll(path, 0o755)
	})
}

func writeOutput(src, dst string, data []byte, rendered bool) error {
	verb := "cp"
	if rendered {
		verb = "render"
	}
	return perform(nil, verb, []string{src, dst}, func() error {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
}

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

const shellSafe = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_@%+=:,./-"

func shellQuote(s string) string {
	unsafe := func(r rune) bool { return !strings.ContainsRune(shellSafe, r) }
	if s != "" && !strings.ContainsFunc(s, unsafe) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
