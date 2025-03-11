package common

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ChainSafe/vm-compat/analyzer"
	"github.com/ChainSafe/vm-compat/asmparser"
	"github.com/ChainSafe/vm-compat/common/lifo"
)

var (
	stdPkgs, _ = getStandardPackages()
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
		if len(sources) >= 20 { // protect for infinite searching
			return
		}
		if seen[segment] {
			return
		}
		seen[segment] = true
		currentStack.Push(segment)

		parents := graph.ParentsOf(segment)

		if endCond(segment.Label()) {
			sources = append(sources, currentStack.Copy())
		} else {
			// sort the parents for consistent output
			slices.SortFunc(parents, func(a, b asmparser.Segment) int {
				if a.Address() > b.Address() {
					return 1
				} else if a.Address() < b.Address() {
					return -1
				}
				return 0
			})

			for _, seg := range parents {
				visit(seg)
			}
		}

		currentStack.Pop()

		// We don't want to revisit the sdk packages as the number of paths can be up to billions.
		pkg := strings.Split(segment.Label(), ".")[0]
		if !stdPkgs[pkg] {
			seen[segment] = false
		}
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

// getStandardPackages fetches all standard library packages and stores them in a map for fast lookup.
func getStandardPackages() (map[string]bool, error) {
	cmd := exec.Command("go", "list", "std")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	stdPackages := make(map[string]bool)
	for _, pkg := range strings.Split(string(output), "\n") {
		if pkg != "" {
			stdPackages[pkg] = true
		}
	}
	return stdPackages, nil
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
