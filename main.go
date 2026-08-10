package main

import (
	"os"

	"github.com/brekke/brekke-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}