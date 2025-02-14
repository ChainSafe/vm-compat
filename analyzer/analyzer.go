// Package analyzer provides an interface for analyzing source code for compatibility issues.
package analyzer

// Analyzer represents the interface for the analyzer.
type Analyzer interface {
	// Analyze analyzes the provided source code and returns any issues found.
	// TODO: better to update the code to take a reader interface instead of path
	Analyze(path string, withTrace bool) ([]*Issue, error)

	// TraceStack generates callstack for a function to debug
	TraceStack(path string, function string) (*IssueSource, error)
}

// IssueSeverity represents the severity level of an issue.
type IssueSeverity string

const (
	IssueSeverityCritical IssueSeverity = "CRITICAL"
	IssueSeverityWarning  IssueSeverity = "WARNING"
)

// Issue represents a single issue found by the analyzer.
type Issue struct {
	Sources  *IssueSource  `json:"sources"`
	Message  string        `json:"message"` // A description of the issue.
	Severity IssueSeverity `json:"severity"`
}

// IssueSource represents a location in the code where the issue originates.
type IssueSource struct {
	File      string       `json:"file"`
	Line      int          `json:"line"`                // The line number where the issue was found.
	Function  string       `json:"function"`            // The function where the issue was found.
	AbsPath   string       `json:"absPath"`             // The absolute file path.
	CallStack *IssueSource `json:"callStack,omitempty"` // The trace of calls leading to this source.
}

// Copy creates a deep copy of the IssueSource.
func (src *IssueSource) Copy() *IssueSource {
	if src == nil {
		return nil
	}
	// Recursively copy the CallStack
	var copiedCallStack *IssueSource
	if src.CallStack != nil {
		copiedCallStack = src.CallStack.Copy()
	}

	return &IssueSource{
		File:      src.File,
		Line:      src.Line,
		Function:  src.Function,
		AbsPath:   src.AbsPath,
		CallStack: copiedCallStack,
	}
}

// AddCallStack add a call stack to the stack et end
func (src *IssueSource) AddCallStack(stack *IssueSource) {
	// Recursively copy the CallStack
	if src.CallStack == nil {
		src.CallStack = stack
		return
	}
	src.CallStack.AddCallStack(stack)
}
