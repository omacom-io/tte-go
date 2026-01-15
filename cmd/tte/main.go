package main

import (
	"os"

	"tte-go/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		cli.PrintError(err)
		os.Exit(1)
	}
}
