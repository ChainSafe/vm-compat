package common

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindGoModuleRoot finds the Go module root directory by searching for `go.mod`
func FindGoModuleRoot(target string) (string, error) {
	dir := filepath.Dir(target)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil // Found go.mod, return this directory as module root
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached the root directory
		}
		dir = parent
	}
	return "", fmt.Errorf("go module root not found for target: %s", target)
}
