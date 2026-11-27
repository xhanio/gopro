package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

// Progress goes quiet under --dry-run, whose stdout is the operations alone;
// warnings still show, on stderr.

func linef(format string, args ...any) {
	printf(color.FgHiWhite, false, false, format, args...)
}

func titlef(format string, args ...any) {
	printf(color.FgHiGreen, true, true, format, args...)
}

func debugf(format string, args ...any) {
	printf(color.FgHiBlue, true, false, format, args...)
}

func warnf(format string, args ...any) {
	w := os.Stdout
	if dryRun {
		w = os.Stderr
	}
	fprintf(w, color.FgHiYellow, true, true, format, args...)
}

func printf(c color.Attribute, env bool, bold bool, format string, args ...any) {
	if dryRun {
		return
	}
	fprintf(os.Stdout, c, env, bold, format, args...)
}

func fprintf(w io.Writer, c color.Attribute, env bool, bold bool, format string, args ...any) {
	ec := color.New(color.FgHiCyan)
	if bold {
		ec.Add(color.Bold)
	}
	ic := color.New(c)
	if bold {
		ic.Add(color.Bold)
	}
	info := ic.Sprintf(format, args...)
	if env && envName != "" {
		info = ec.Sprintf("[ %s ] ", envName) + info
	}
	fmt.Fprintln(w, info)
}
