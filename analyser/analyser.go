// Package analyser provides an interface for analyzing source code for compatibility issues.
package analyser

// Analyzer represents the interface for the analyzer.
type Analyzer interface {
	// Analyze analyzes the provided source code and returns any issues found.
	// TODO: better to update the code to take a reader interface instead of path
	Analyze(path string) ([]*Issue, error)
}

// Issue represents a single issue found by the analyzer.
type Issue struct {
	File    string `json:"file"`    // The file where the issue was found.
	Source  string `json:"source"`  // The source(line/pc address, block or function)
	Message string `json:"message"` // A description of the issue.
}
