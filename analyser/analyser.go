package analyser

// Analyzer represents the interface for the analyser.
type Analyzer interface {
	// Analyze analyzes the provided source code and returns any issues found.
	// TODO: better to update the code to take a reader interface instead of path
	Analyze(path string) ([]*Issue, error)
}

// Issue represents a single issue found by the analyser.
type Issue struct {
	File    string // The file where the issue was found.
	Line    int    // The line number of the issue.
	Segment string // The block or function where the issue found
	Message string // A description of the issue.
}
