package main

import (
	"fmt"
	"os"

	"github.com/roth-andreas/gosungrow-home-assistant/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, cmd.FailureClassLine(err))
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
