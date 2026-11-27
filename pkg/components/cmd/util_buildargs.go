package cmd

import (
	"errors"
	"fmt"
	"strings"
)

const infoPackage = "github.com/xhanio/framingo/pkg/types/info"

// flags whose value is "[pattern=]args" for a tool
var toolFlags = map[string]bool{
	"asmflags":   true,
	"gccgoflags": true,
	"gcflags":    true,
	"ldflags":    true,
}

// Kept in step with "go help build" by TestValueFlagsCoverGoBuild.
var valueFlags = map[string]bool{
	"C": true, "buildmode": true, "compiler": true, "coverpkg": true,
	"covermode": true, "installsuffix": true, "mod": true, "modfile": true,
	"overlay": true, "p": true, "pgo": true, "pkgdir": true, "tags": true,
	"toolexec": true,
}

// assembleBuildArgs follows go build, except: tool flags with the same pattern
// accumulate into one where go build keeps the last; -ldflags takes no pattern
// but "all=" and gets the injected fields appended; and -o, -X on infoPackage
// and positionals are errors, since gopro appends -o and the package itself
// and go build stops parsing flags at a positional.
func assembleBuildArgs(user, injected []string) ([]string, error) {
	type key struct{ name, pattern string }
	type merged struct {
		pos     int // index in out of the flag's value
		pattern string
		values  []string
	}
	var out []string
	tools := map[key]*merged{}
	get := func(name, pattern string) *merged {
		k := key{name, pattern}
		if m, ok := tools[k]; ok {
			return m
		}
		out = append(out, "-"+name, "")
		m := &merged{pos: len(out) - 1, pattern: pattern}
		tools[k] = m
		return m
	}

	for i := 0; i < len(user); i++ {
		start := i
		name, value, hasValue, isFlag := parseFlag(user[i])
		if !isFlag {
			return nil, positionalError(user, i)
		}
		if name == "o" {
			return nil, errors.New("build args may not set -o: gopro names the output, use binary_tgt or --output")
		}
		if !hasValue && (valueFlags[name] || toolFlags[name]) {
			if i+1 == len(user) {
				return nil, fmt.Errorf("build arg -%s needs a value", name)
			}
			i++
			value = user[i]
		}
		if !toolFlags[name] {
			out = append(out, user[start:i+1]...)
			continue
		}
		pattern, args, err := splitPattern(value)
		if err != nil {
			return nil, fmt.Errorf("build arg -%s: %w", name, err)
		}
		if name == "ldflags" {
			if pattern != "" && pattern != "all" {
				return nil, fmt.Errorf("build arg -ldflags: pattern %q is not supported, gopro injects into the main package's link; drop the pattern", pattern)
			}
			pattern = ""
			if err := checkReservedLinkerFlags(args); err != nil {
				return nil, err
			}
		}
		m := get(name, pattern)
		if strings.TrimSpace(args) == "" {
			m.values = nil
		} else {
			m.values = append(m.values, args)
		}
	}

	if len(injected) > 0 {
		joined, err := joinQuoted(injected)
		if err != nil {
			return nil, err
		}
		m := get("ldflags", "")
		m.values = append(m.values, joined)
	}
	for _, m := range tools {
		v := strings.Join(m.values, " ")
		if m.pattern != "" {
			v = m.pattern + "=" + v
		}
		out[m.pos] = v
	}
	return out, nil
}

// After a flag gopro does not know to take a value, the positional is likely
// that flag's value.
func positionalError(user []string, i int) error {
	err := fmt.Errorf("build args take flags only, but %q is positional: gopro supplies the package to build", user[i])
	if i > 0 {
		name, _, hasValue, isFlag := parseFlag(user[i-1])
		if isFlag && !hasValue && !valueFlags[name] && !toolFlags[name] {
			err = fmt.Errorf("%w (if it is the value of -%s, write -%s=%s)", err, name, name, user[i])
		}
	}
	return err
}

func parseFlag(arg string) (name, value string, hasValue, isFlag bool) {
	if len(arg) < 2 || arg[0] != '-' || arg == "--" {
		return "", "", false, false
	}
	name, value, hasValue = strings.Cut(strings.TrimPrefix(arg[1:], "-"), "=")
	return name, value, hasValue, name != ""
}

// The args are checked for quoting here, or merging would run an open quote
// into the next value.
func splitPattern(v string) (pattern, args string, err error) {
	v = strings.TrimSpace(v)
	if v != "" && !strings.HasPrefix(v, "-") {
		p, rest, ok := strings.Cut(v, "=")
		if !ok || strings.TrimSpace(p) == "" {
			return "", "", fmt.Errorf("%q is neither args nor pattern=args", v)
		}
		pattern, v = strings.TrimSpace(p), rest
	}
	if _, err := splitQuoted(v); err != nil {
		return "", "", err
	}
	return pattern, v, nil
}

func checkReservedLinkerFlags(args string) error {
	fields, err := splitQuoted(args)
	if err != nil {
		return err
	}
	for i, f := range fields {
		name, def, hasValue, _ := parseFlag(f)
		if name != "X" {
			continue
		}
		if !hasValue && i+1 < len(fields) {
			def = fields[i+1]
		}
		if strings.HasPrefix(def, infoPackage+".") {
			return fmt.Errorf("build args may not set -X %s: gopro injects %s.*; use project.yaml or the --build-* flags", def, infoPackage)
		}
	}
	return nil
}

// Mirrors cmd/internal/quoted, which has no escapes.
func splitQuoted(s string) ([]string, error) {
	var fields []string
	for {
		s = strings.TrimLeft(s, " \t\n\r")
		if s == "" {
			return fields, nil
		}
		if q := s[0]; q == '"' || q == '\'' {
			i := strings.IndexByte(s[1:], q)
			if i < 0 {
				return nil, fmt.Errorf("unterminated %c string in %q", q, s)
			}
			fields = append(fields, s[1:1+i])
			s = s[2+i:]
			continue
		}
		i := strings.IndexAny(s, " \t\n\r")
		if i < 0 {
			i = len(s)
		}
		fields = append(fields, s[:i])
		s = s[i:]
	}
}

// Inverse of splitQuoted.
func joinQuoted(fields []string) (string, error) {
	parts := make([]string, len(fields))
	for i, f := range fields {
		single, double := strings.ContainsRune(f, '\''), strings.ContainsRune(f, '"')
		switch {
		case f != "" && !single && !double && !strings.ContainsAny(f, " \t\n\r"):
			parts[i] = f
		case !single:
			parts[i] = "'" + f + "'"
		case !double:
			parts[i] = `"` + f + `"`
		default:
			return "", fmt.Errorf("%q holds both single and double quotes and cannot be quoted for the linker", f)
		}
	}
	return strings.Join(parts, " "), nil
}
