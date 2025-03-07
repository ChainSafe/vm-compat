package common

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/ChainSafe/vm-compat/analyzer"
	"github.com/ChainSafe/vm-compat/asmparser"
	"github.com/ChainSafe/vm-compat/common/lifo"
)

// TraceAsmCaller correctly tracks function calls in the execution stack.
func TraceAsmCaller(
	filePath string,
	graph asmparser.CallGraph,
	function string,
	endCond func(string) bool,
) (*analyzer.CallStack, error) {
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
	var visit func(graph asmparser.CallGraph, segment asmparser.Segment) *analyzer.CallStack

	visit = func(graph asmparser.CallGraph, segment asmparser.Segment) *analyzer.CallStack {
		if seen[segment] {
			return nil
		}
		seen[segment] = true

		source := &analyzer.CallStack{
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

// TraceAllAsmCaller correctly tracks all possible function calls in the execution stack.
func TraceAllAsmCaller(
	filePath string,
	graph asmparser.CallGraph,
	function string,
	endCond func(string) bool,
) ([]*analyzer.CallStack, error) {
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
	sources := make([]*lifo.Stack[asmparser.Segment], 0)
	currentStack := lifo.Stack[asmparser.Segment]{}
	seen := make(map[asmparser.Segment]bool)

	var visit func(segment asmparser.Segment)

	visit = func(segment asmparser.Segment) {
		if seen[segment] {
			return
		}
		seen[segment] = true
		currentStack.Push(segment)

		if len(sources) >= 10 {
			return
		}

		if endCond(segment.Label()) {
			sources = append(sources, currentStack.Copy())
		} else {
			for _, seg := range graph.ParentsOf(segment) {
				visit(seg)
			}
		}

		currentStack.Pop()
		seen[segment] = false
	}

	visit(segment)

	if len(sources) == 0 {
		return nil, fmt.Errorf("no trace found to root for the given function")
	}
	// build call stacks from the sources
	traces := make([]*analyzer.CallStack, len(sources))
	for i, source := range sources {
		var callStack *analyzer.CallStack
		for !source.IsEmpty() {
			if seg, ok := source.Pop(); ok {
				stack := &analyzer.CallStack{
					File:     filepath.Base(filePath),
					Line:     seg.Instructions()[0].Line() - 1, // function start line
					AbsPath:  filePath,
					Function: seg.Label(),
				}
				if callStack == nil {
					callStack = stack
				} else {
					stack.CallStack = callStack
					callStack = stack
				}
			}
		}
		traces[i] = callStack
	}

	return traces, nil
}

func ShouldIgnoreSource(callStack *analyzer.CallStack, functions []string) bool {
	if callStack != nil {
		if slices.Contains(functions, callStack.Function) {
			return true
		}
		return ShouldIgnoreSource(callStack.CallStack, functions)
	}
	return false
}
