// Package main is the program entry point for the FlakeHunter CLI.
//
// In Go, only a package named "main" with a function named main()
// can become an executable. Keep this file thin: parse nothing complex
// here — call into internal packages instead.
package main

import (
	"os"

	"github.com/edisshort/flakehunter/internal/cli"
)

func main() {
	// os.Args is the slice of command-line arguments.
	// Index 0 is the program name; the rest are user-supplied flags/words.
	code := cli.Run(os.Args[1:])
	if code != 0 {
		os.Exit(code)
	}
}
