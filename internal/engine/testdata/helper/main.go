// Helper program used by engine tests (not part of the FlakeHunter binary).
// Usage: go run ./testdata/helper <ok|fail|sleep>
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	mode := "ok"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	switch mode {
	case "ok":
		fmt.Println("helper-ok")
		os.Exit(0)
	case "fail":
		fmt.Fprintln(os.Stderr, "helper-fail")
		os.Exit(7)
	case "sleep":
		time.Sleep(5 * time.Second)
		fmt.Println("slept")
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q\n", mode)
		os.Exit(2)
	}
}
