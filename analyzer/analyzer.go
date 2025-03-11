// Package analyzer provides an interface for analyzing source code for compatibility issues.
package analyzer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Analyzer represents the interface for the analyzer.
type Analyzer interface {
	// Analyze analyzes the provided source code and returns any issues found.
	// TODO: better to update the code to take a reader interface instead of path
	Analyze(path string, withTrace bool, skipWarnings bool) ([]*Issue, error)

	// TraceStack generates callstack for a function to debug
	TraceStack(path string, function string) (*CallStack, error)
}

// IssueSeverity represents the severity level of an issue.
type IssueSeverity string

const (
	IssueSeverityCritical IssueSeverity = "CRITICAL"
	IssueSeverityWarning  IssueSeverity = "WARNING"
)

// Issue represents a single issue found by the analyzer.
type Issue struct {
	CallStack *CallStack    `json:"callStack"`
	Message   string        `json:"message"` // A description of the issue.
	Severity  IssueSeverity `json:"severity"`
	Impact    string        `json:"impact,omitempty"`
	Reference string        `json:"reference,omitempty"`
	Hash      string        `json:"hash"`
}

// Opt is a functional option for configuring an Issue.
type Opt func(*Issue)

// WithImpact sets the impact of the issue.
func WithImpact(impact string) Opt {
	return func(i *Issue) {
		i.Impact = impact
	}
}

// WithReference sets the reference for the issue.
func WithReference(reference string) Opt {
	return func(i *Issue) {
		i.Reference = reference
	}
}

// WithCallStack sets the call stack for the issue.
func WithCallStack(callStack *CallStack) Opt {
	return func(i *Issue) {
		i.CallStack = callStack
	}
}

// WithSeverity sets the severity for the issue.
func WithSeverity(severity IssueSeverity) Opt {
	return func(i *Issue) {
		i.Severity = severity
	}
}

// WithMessage sets the message for the issue.
func WithMessage(message string) Opt {
	return func(i *Issue) {
		i.Message = message
	}
}

// NewIssue creates a new issue with the provided severity, message, and source.
func NewIssue(opts ...Opt) *Issue {
	issue := new(Issue)
	for _, opt := range opts {
		opt(issue)
	}
	issue.PopulateHash()
	return issue
}

// PopulateHash generates a hash for the issue and populates it into the issue
func (i *Issue) PopulateHash() {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s:%s", i.Message, i.CallStack.Trace())
	i.Hash = hex.EncodeToString(h.Sum(nil))
}

// CallStack represents a location in the code where the issue originates.
type CallStack struct {
	File      string     `json:"file"`
	Line      int        `json:"line"`                // The line number where the issue was found.
	Function  string     `json:"function"`            // The function where the issue was found.
	AbsPath   string     `json:"absPath"`             // The absolute file path.
	CallStack *CallStack `json:"callStack,omitempty"` // The trace of calls leading to this source.
}

func (src *CallStack) Trace() string {
	sub := src.Function
	if src.CallStack != nil {
		sub = fmt.Sprintf("%s:%s", sub, src.CallStack.Trace())
	}
	return sub
}

// Copy creates a deep copy of the CallStack.
func (src *CallStack) Copy() *CallStack {
	if src == nil {
		return nil
	}
	// Recursively copy the CallStack
	var copiedCallStack *CallStack
	if src.CallStack != nil {
		copiedCallStack = src.CallStack.Copy()
	}

	return &CallStack{
		File:      src.File,
		Line:      src.Line,
		Function:  src.Function,
		AbsPath:   src.AbsPath,
		CallStack: copiedCallStack,
	}
}

// AddCallStack add a call stack to the stack et end
func (src *CallStack) AddCallStack(stack *CallStack) {
	// Recursively copy the CallStack
	if src.CallStack == nil {
		src.CallStack = stack
		return
	}
	src.CallStack.AddCallStack(stack)
}

// SortIssues sorts the issues by severity and hash.
func SortIssues(issues []*Issue) []*Issue {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Severity != issues[j].Severity {
			return issues[i].Severity < issues[j].Severity
		}
		return strings.Compare(issues[i].Hash, issues[j].Hash) < 0
	})
	return issues
}
