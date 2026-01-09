package main

import (
	"os"

	"github.com/n0roo/pal-kit/cmd/palkit/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
