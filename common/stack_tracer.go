package common

import (
	"path/filepath"

	"github.com/ChainSafe/vm-compat/analyser"
	"github.com/ChainSafe/vm-compat/asmparser"
)

// TraceAsmCaller correctly tracks function calls in the execution stack.
func TraceAsmCaller(
	filePath string,
	instruction asmparser.Instruction,
	segment asmparser.Segment,
	graph asmparser.CallGraph,
	paths []*analyser.IssueSource,
	depth int) []*analyser.IssueSource {
	if instruction == nil || segment == nil {
		return paths // Prevent nil pointer dereference
	}

	// Create a new IssueSource entry for this function call
	source := &analyser.IssueSource{
		File:     filepath.Base(filePath),
		Line:     instruction.Line(),
		AbsPath:  filePath,
		Function: segment.Label(),
	}
	// If this is the first function call in the trace, initialize the stack
	newPaths := make([]*analyser.IssueSource, 0)
	if len(paths) == 0 {
		newPaths = []*analyser.IssueSource{source}
	} else {
		if len(paths) > 1 {
			panic("multiple paths not possible")
		}
		newPath := paths[0].Copy()
		newPath.AddCallStack(source)
		newPaths = append(newPaths, newPath)
	}

	parents := graph.ParentsOf(segment)
	// Stop recursion at desired depth to prevent infinite loops
	if depth >= 1 || len(parents) == 0 {
		return newPaths
	}

	// Recurse for previous function calls (callers)
	result := make([]*analyser.IssueSource, 0)
	for _, seg := range parents {
		result = append(result, TraceAsmCaller(filePath, seg.Instructions()[0], seg, graph, newPaths, depth+1)...)
	}

	return result
}
