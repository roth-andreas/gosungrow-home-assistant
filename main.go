package main

import (
	"fmt"
	"github.com/roth-andreas/gosungrow-home-assistant/cmd"
	"os"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
