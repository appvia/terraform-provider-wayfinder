//go:build wfmanifestgen

// Placeholder generator: proves the harness compiles and runs inside the
// Wayfinder checkout. Harness flags/args are accepted and ignored.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	f, err := os.Open("go.mod")
	if err != nil {
		fmt.Fprintln(os.Stderr, "manifestgen:", err)
		os.Exit(1)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "go "); ok {
			fmt.Println("wayfinder go version:", v)
			return
		}
	}
	fmt.Fprintln(os.Stderr, "manifestgen: no go directive in go.mod")
	os.Exit(1)
}
