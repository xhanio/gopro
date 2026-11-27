package types

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Validate reports every component a project names wrongly: a name that is
// not a single path element, one defined twice, a list entry or $base that
// names nothing defined, and a list entry given twice. Nothing reported here
// is ever acted on as written, and a name like ".." would make generation
// clear the parent of its target.
func (p *Project) Validate() error {
	var errs []error
	defined := map[string][]string{
		"build.binaries":      definitions(&errs, "build.binaries", p.Build.Binaries, func(b BinarySpec) string { return b.Name }),
		"build.images":        definitions(&errs, "build.images", p.Build.Images, func(i ImageSpec) string { return i.Name }),
		"generate.configs":    definitions(&errs, "generate.configs", p.Generate.Configs, func(c ConfigSpec) string { return c.Name }),
		"generate.kubernetes": definitions(&errs, "generate.kubernetes", p.Generate.Kubernetes, func(k KubernetesSpec) string { return k.Name }),
	}
	for i, image := range p.Build.Images {
		if base, ok := strings.CutPrefix(image.Base, "$"); ok && !slices.Contains(defined["build.images"], base) {
			errs = append(errs, fmt.Errorf("build.images[%d].base %q: %q is not defined in build.images", i, image.Base, base))
		}
	}

	check := func(section string, spec EnvSpec) {
		references(&errs, section+".binaries", spec.Binaries, "build.binaries", defined)
		references(&errs, section+".images", spec.Images, "build.images", defined)
		references(&errs, section+".configs", spec.Configs, "generate.configs", defined)
		references(&errs, section+".kubernetes_templates", spec.KubernetesTemplates, "generate.kubernetes", defined)
	}
	check("default", p.Default)
	// An env is checked only for the lists it sets; the rest are default's.
	for _, name := range slices.Sorted(maps.Keys(p.Env)) {
		check("env."+name, p.Env[name])
	}
	return errors.Join(errs...)
}

// definitions checks the names in a definition list and returns them.
func definitions[T any](errs *[]error, list string, specs []T, name func(T) string) []string {
	names := make([]string, 0, len(specs))
	for i, spec := range specs {
		n := name(spec)
		if !isPathElement(n) {
			*errs = append(*errs, fmt.Errorf("%s[%d].name %q: must be a single path element, since it names files and directories", list, i, n))
		}
		if slices.Contains(names, n) {
			*errs = append(*errs, fmt.Errorf("%s: %q is defined twice", list, n))
		}
		names = append(names, n)
	}
	return names
}

// references checks that every entry of an env-level list names a defined
// component, once.
func references(errs *[]error, list string, entries []string, definedIn string, defined map[string][]string) {
	for i, n := range entries {
		if !slices.Contains(defined[definedIn], n) {
			*errs = append(*errs, fmt.Errorf("%s: %q is not defined in %s", list, n, definedIn))
		}
		if slices.Contains(entries[:i], n) {
			*errs = append(*errs, fmt.Errorf("%s: %q is listed twice", list, n))
		}
	}
}

func isPathElement(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`)
}
