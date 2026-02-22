package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/wallgen"
)

func main() {
	repoRoot, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result, err := wallgen.Generate(repoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Synced snippet markers in %s\n", result.ReadmePath)
}
