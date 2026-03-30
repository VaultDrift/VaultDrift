package main

import (
	"fmt"
	"os"

	"github.com/vaultdrift/vaultdrift/internal/cli"
)

func main() {
	app, err := cli.NewCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
