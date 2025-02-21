package common

import (
	"fmt"
	"path/filepath"

	"github.com/ChainSafe/vm-compat/analyzer"
	"github.com/ChainSafe/vm-compat/asmparser"
)

// TraceAsmCaller correctly tracks function calls in the execution stack.
func TraceAsmCaller(
	filePath string,
	graph asmparser.CallGraph,
	function string,
	endCond func(string) bool,
) (*analyzer.IssueSource, error) {
	var segment asmparser.Segment
	for _, seg := range graph.Segments() {
		if seg.Label() == function {
			segment = seg
			break
		}
	}
	if segment == nil {
		return nil, fmt.Errorf("could not find %s in %s", function, filePath)
	}
	seen := make(map[asmparser.Segment]bool)
	var visit func(graph asmparser.CallGraph, segment asmparser.Segment) *analyzer.IssueSource

	visit = func(graph asmparser.CallGraph, segment asmparser.Segment) *analyzer.IssueSource {
		if seen[segment] {
			return nil
		}
		seen[segment] = true

		source := &analyzer.IssueSource{
			File:     filepath.Base(filePath),
			Line:     segment.Instructions()[0].Line() - 1, // function start line
			AbsPath:  filePath,
			Function: segment.Label(),
		}
		if endCond(source.Function) {
			return source
		}
		for _, seg := range graph.ParentsOf(segment) {
			ch := visit(graph, seg)
			if ch != nil {
				source.AddCallStack(ch)
				return source
			}
		}
		return nil
	}
	src := visit(graph, segment)
	if src == nil {
		return nil, fmt.Errorf("no trace found to root for the given function")
	}
	return src, nil
}
