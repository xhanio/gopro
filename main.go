package main

import (
	"os"

	"github.com/xhanio/gopro/pkg/components/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(-1)
	}
}
