package cmd

import (
	"errors"
	"fmt"
	"strings"
)

// infoPackage is the package whose vars gopro sets with -X.
const infoPackage = "github.com/xhanio/framingo/pkg/types/info"

// toolFlags take a "[pattern=]args" string that go build hands to a tool.
var toolFlags = map[string]bool{
	"asmflags":   true,
	"gccgoflags": true,
	"gcflags":    true,
	"ldflags":    true,
}

// valueFlags are the other go build flags that take a separate value.
// TestValueFlagsCoverGoBuild keeps the list in step with "go help build"; an
// unlisted flag still works as -name=value.
var valueFlags = map[string]bool{
	"C": true, "buildmode": true, "compiler": true, "coverpkg": true,
	"covermode": true, "installsuffix": true, "mod": true, "modfile": true,
	"overlay": true, "p": true, "pgo": true, "pkgdir": true, "tags": true,
	"toolexec": true,
}

// assembleBuildArgs merges the user's go build args with the injected linker
// fields. It follows go build, except:
//
//   - Tool flags accumulate: occurrences with the same pattern join into one
//     flag at the first one's position, where go build keeps only the last.
//     An empty value still resets the earlier ones.
//   - -ldflags takes no pattern or "all=" (both reach the main package's
//     link) and gets the injected fields appended, quoted.
//   - -o, -X on infoPackage, and positionals are errors: gopro appends -o
//     and the package itself, and go build stops parsing flags at the first
//     positional.
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

// positionalError reports user[i] as positional. When it follows a bare flag
// gopro does not know to take a value, it is probably that flag's value, so
// the error suggests the -name=value form.
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

// parseFlag splits "-name", "--name" and "-name=value". isFlag is false for
// anything else, "--" included.
func parseFlag(arg string) (name, value string, hasValue, isFlag bool) {
	if len(arg) < 2 || arg[0] != '-' || arg == "--" {
		return "", "", false, false
	}
	name, value, hasValue = strings.Cut(strings.TrimPrefix(arg[1:], "-"), "=")
	return name, value, hasValue, name != ""
}

// splitPattern splits a tool flag value into pattern and args as go build
// does: a value not starting with "-" is "pattern=args". The args must be
// well quoted, or merging would run an open quote into the next value.
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

// checkReservedLinkerFlags rejects a -X on infoPackage.
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

// splitQuoted splits tool flags as the go command does (cmd/internal/quoted):
// on whitespace, where a field opening with ' or " runs to the matching
// quote. There are no escapes.
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

// joinQuoted inverts splitQuoted, wrapping a field that holds whitespace or a
// quote in the quote it lacks. A field holding both quotes is an error.
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
