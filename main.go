package main

import (
	"os"

	"github.com/abhinavcdev/gflow/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
