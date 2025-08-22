package info

import "strings"

func EnvPrefix(name string) string {
	pn := name
	pn = strings.ReplaceAll(pn, " ", "_")
	pn = strings.ReplaceAll(pn, "-", "_")
	pn = strings.ToUpper(pn)
	return pn
}
