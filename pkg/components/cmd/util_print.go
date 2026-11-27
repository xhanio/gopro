package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

func linef(format string, args ...any) {
	printf(messageOutput(false), color.FgHiWhite, false, false, format, args...)
}

func titlef(format string, args ...any) {
	printf(messageOutput(false), color.FgHiGreen, true, true, format, args...)
}

func debugf(format string, args ...any) {
	printf(messageOutput(false), color.FgHiBlue, true, false, format, args...)
}

func warnf(format string, args ...any) {
	printf(messageOutput(true), color.FgHiYellow, true, true, format, args...)
}

// Under --dry-run stdout is the operations alone.
func messageOutput(warning bool) io.Writer {
	switch {
	case !dryRun:
		return os.Stdout
	case warning:
		return os.Stderr
	default:
		return io.Discard
	}
}

func printf(w io.Writer, c color.Attribute, withEnv, bold bool, format string, args ...any) {
	ec := color.New(color.FgHiCyan)
	ic := color.New(c)
	if bold {
		ec.Add(color.Bold)
		ic.Add(color.Bold)
	}
	line := ic.Sprintf(format, args...)
	if withEnv && envName != "" {
		line = ec.Sprintf("[ %s ] ", envName) + line
	}
	fmt.Fprintln(w, line)
}
