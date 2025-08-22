package cmd

import (
	"fmt"

	"github.com/fatih/color"
)

func linef(format string, args ...any) {
	printf(color.FgHiWhite, false, false, format, args...)
}

func titlef(format string, args ...any) {
	printf(color.FgHiGreen, true, true, format, args...)
}

func debugf(format string, args ...any) {
	printf(color.FgHiBlue, true, false, format, args...)
}

func printf(c color.Attribute, env bool, bold bool, format string, args ...any) {
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
	fmt.Println(info)
}
