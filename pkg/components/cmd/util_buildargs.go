package cmd

import (
	"errors"
	"fmt"
	"strings"
)

// infoPackage is the import path whose vars gopro stamps through -X.
const infoPackage = "github.com/xhanio/framingo/pkg/types/info"

// toolFlags are the go build flags whose value is an argument string for a
// tool, optionally scoped by a package pattern ("pattern=flags").
var toolFlags = map[string]bool{
	"asmflags":   true,
	"gccgoflags": true,
	"gcflags":    true,
	"ldflags":    true,
}

// valueFlags are the other go build flags taking a separate value, so that
// the value is skipped over instead of read as a flag of its own.
var valueFlags = map[string]bool{
	"C": true, "buildmode": true, "compiler": true, "coverpkg": true,
	"covermode": true, "installsuffix": true, "mod": true, "modfile": true,
	"o": true, "overlay": true, "p": true, "pgo": true, "pkgdir": true,
	"tags": true, "toolexec": true,
}

// toolFlag is one merged tool flag: every occurrence sharing a name and
// pattern, concatenated in the order given.
type toolFlag struct {
	name, pattern string
	values        []string
}

func (f *toolFlag) arg() string {
	v := strings.Join(f.values, " ")
	if f.pattern != "" {
		return f.pattern + "=" + v
	}
	return v
}

// assembleBuildArgs turns the build args the user declared into the flags
// passed to go build, folding in the linker fields gopro injects. It follows
// go build except where that would silently discard what the user wrote:
//
//   - Tool flags (-ldflags, -gcflags, -asmflags, -gccgoflags) accumulate: the
//     occurrences sharing a package pattern join into one flag, placed where
//     the first appeared. go build keeps only the last, so writing -ldflags
//     twice, or gopro adding its own, would drop the earlier. An empty value
//     still resets the ones before it, as in go build.
//   - For -ldflags, "all=" is treated like no pattern, since both reach the
//     link of the main package; any other pattern is refused, because the
//     injection could not take effect alongside it.
//   - What gopro sets itself is reserved: -o, and -X on the info package.
//     Supplying either is an error rather than a silent override.
//   - Everything else passes through untouched, positional rules included.
//
// The injected fields join the -ldflags after the user's, quoted so a value
// holding spaces stays one field.
func assembleBuildArgs(user, injected []string) ([]string, error) {
	var out []string
	merged := map[string]*toolFlag{}
	// position in out of each merged flag's value, filled in at the end
	var order []*toolFlag
	at := map[*toolFlag]int{}

	for i := 0; i < len(user); i++ {
		arg := user[i]
		name, value, hasValue, isFlag := parseFlag(arg)
		if !isFlag {
			out = append(out, arg)
			continue
		}
		if name == "o" {
			return nil, errors.New("build args may not set -o: gopro names the output, use binary_tgt or --output")
		}
		if !toolFlags[name] {
			out = append(out, arg)
			if valueFlags[name] && !hasValue {
				if i+1 >= len(user) {
					return nil, fmt.Errorf("build arg -%s needs a value", name)
				}
				i++
				out = append(out, user[i])
			}
			continue
		}
		if !hasValue {
			if i+1 >= len(user) {
				return nil, fmt.Errorf("build arg -%s needs a value", name)
			}
			i++
			value = user[i]
		}
		pattern, flags, err := splitPattern(value)
		if err != nil {
			return nil, fmt.Errorf("build arg -%s: %w", name, err)
		}
		if name == "ldflags" {
			if pattern != "" && pattern != "all" {
				return nil, fmt.Errorf("build arg -ldflags: pattern %q is not supported, gopro injects into the main package's link; drop the pattern", pattern)
			}
			pattern = ""
			if err := checkReservedLinkerFlags(flags); err != nil {
				return nil, err
			}
		}
		key := name + "\x00" + pattern
		f, ok := merged[key]
		if !ok {
			f = &toolFlag{name: name, pattern: pattern}
			merged[key] = f
			order = append(order, f)
			out = append(out, "-"+name, "")
			at[f] = len(out) - 1
		}
		if strings.TrimSpace(flags) == "" {
			f.values = nil
		} else {
			f.values = append(f.values, flags)
		}
	}

	if len(injected) > 0 {
		joined, err := joinQuoted(injected)
		if err != nil {
			return nil, err
		}
		key := "ldflags\x00"
		f, ok := merged[key]
		if !ok {
			f = &toolFlag{name: "ldflags"}
			order = append(order, f)
			out = append(out, "-ldflags", "")
			at[f] = len(out) - 1
		}
		f.values = append(f.values, joined)
	}
	for _, f := range order {
		out[at[f]] = f.arg()
	}
	return out, nil
}

// parseFlag splits a "-name", "--name" or "-name=value" argument. isFlag is
// false for anything else, which passes through as is.
func parseFlag(arg string) (name, value string, hasValue, isFlag bool) {
	if len(arg) < 2 || arg[0] != '-' || arg == "--" {
		return "", "", false, false
	}
	s := strings.TrimPrefix(arg[1:], "-")
	name, value, hasValue = strings.Cut(s, "=")
	return name, value, hasValue, name != ""
}

// splitPattern separates a tool flag value into its package pattern and the
// flags, mirroring go build: a value not starting with "-" is
// "pattern=flags". The flags are validated as quoting goes, since an
// unbalanced quote would otherwise run into the next value once merged.
func splitPattern(v string) (pattern, flags string, err error) {
	v = strings.TrimSpace(v)
	if v != "" && !strings.HasPrefix(v, "-") {
		p, rest, ok := strings.Cut(v, "=")
		if !ok || strings.TrimSpace(p) == "" {
			return "", "", fmt.Errorf("%q is neither flags nor pattern=flags", v)
		}
		pattern, v = strings.TrimSpace(p), rest
	}
	if _, err := splitQuoted(v); err != nil {
		return "", "", err
	}
	return pattern, v, nil
}

// checkReservedLinkerFlags refuses a -X the user aimed at the info package.
func checkReservedLinkerFlags(flags string) error {
	fields, err := splitQuoted(flags)
	if err != nil {
		return err
	}
	for i, f := range fields {
		var def string
		switch {
		case f == "-X" || f == "--X":
			if i+1 < len(fields) {
				def = fields[i+1]
			}
		case strings.HasPrefix(f, "-X="):
			def = f[len("-X="):]
		case strings.HasPrefix(f, "--X="):
			def = f[len("--X="):]
		default:
			continue
		}
		if strings.HasPrefix(def, infoPackage+".") {
			return fmt.Errorf("build args may not set -X %s: gopro injects %s.*; use project.yaml or the --build-* flags", def, infoPackage)
		}
	}
	return nil
}

// splitQuoted splits a tool flag string the way the go command does
// (cmd/internal/quoted): on whitespace, with a field that starts with a
// single or double quote running to the matching one. There are no escapes.
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

// joinQuoted is the inverse of splitQuoted: a field holding whitespace or a
// quote is wrapped in whichever quote it does not contain. One holding both
// cannot be expressed and is an error.
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
