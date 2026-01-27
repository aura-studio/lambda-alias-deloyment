package main

import (
	"os"

	"github.com/aura-studio/lambda-alias-deployment/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
