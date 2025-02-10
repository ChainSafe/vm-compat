package syscall

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/ChainSafe/vm-compat/analyser"
	"github.com/ChainSafe/vm-compat/common"
	"github.com/ChainSafe/vm-compat/profile"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var syscallAPIs = []string{
	"syscall.RawSyscall6",
	"syscall.rawSyscallNoError",
	"syscall.rawVforkSyscall",
	"syscall.runtime_doAllThreadsSyscall",
}

// goSyscallAnalyser analyzes system calls in Go binaries.
type goSyscallAnalyser struct {
	profile *profile.VMProfile
}

// NewGOSyscallAnalyser initializes an analyser for Go syscalls.
func NewGOSyscallAnalyser(profile *profile.VMProfile) analyser.Analyzer {
	return &goSyscallAnalyser{profile: profile}
}

// Analyze scans a Go binary for syscalls and detects compatibility issues.
//
//nolint:cyclop
func (a *goSyscallAnalyser) Analyze(path string) ([]*analyser.Issue, error) {
	// Find the Go module root for correct context
	modRoot, err := common.FindGoModuleRoot(path)
	if err != nil {
		return nil, fmt.Errorf("failed to find Go module root: %w", err)
	}
	cfg := &packages.Config{
		Mode:       packages.LoadAllSyntax,
		BuildFlags: []string{},
		Dir:        modRoot,
		Env: append(
			os.Environ(),
			fmt.Sprintf("GOOS=%s", a.profile.GOOS),
			fmt.Sprintf("GOARCH=%s", a.profile.GOARCH),
		),
	}

	initial, err := packages.Load(cfg, path)
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(initial) > 0 {
		return nil, fmt.Errorf("packages contain errors")
	}

	// Create and build SSA-form program representation.
	mode := ssa.InstantiateGenerics
	prog, _ := ssautil.AllPackages(initial, mode)
	prog.Build()

	// Construct call graph using RTA analysis.
	mains, err := mainPackages(prog.AllPackages())
	if err != nil {
		return nil, err
	}
	roots := make([]*ssa.Function, 0)
	for _, main := range mains {
		roots = append(roots, main.Func("main"))
	}
	roots = append(roots, initFuncs(prog.AllPackages())...)

	cg := rta.Analyze(roots, true).CallGraph
	cg.DeleteSyntheticNodes()

	// Analyze call graph for syscalls.
	fset := initial[0].Fset
	syscalls := make([]syscallSource, 0)
	err = callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		callee := edge.Callee.Func
		if callee != nil && callee.Pkg != nil && callee.Pkg.Pkg != nil {
			if slices.Contains(syscallAPIs, callee.String()) {
				calls := traceSyscalls(edge.Site.Common().Args[0], edge, fset)
				syscalls = append(syscalls, calls...)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Check against allowed syscalls.
	issues := []*analyser.Issue{}
	for i := range syscalls {
		syscll := syscalls[i]
		if slices.Contains(a.profile.AllowedSycalls, syscll.num) {
			continue
		}

		severity := analyser.IssueSeverityCritical
		message := fmt.Sprintf("Incompatible Syscall Detected: %d", syscll.num)
		if slices.Contains(a.profile.NOOPSyscalls, syscll.num) {
			severity = analyser.IssueSeverityWarning
			message = fmt.Sprintf("NOOP Syscall Detected: %d", syscll.num)
		}

		issues = append(issues, &analyser.Issue{
			Severity: severity,
			Sources:  syscll.source,
			Message:  message,
		})
	}

	return issues, nil
}

type syscallSource struct {
	num    int
	source []*analyser.IssueSource
}

func traceSyscalls(value ssa.Value, edge *callgraph.Edge, fset *token.FileSet) []syscallSource {
	result := make([]syscallSource, 0)
	switch v := value.(type) {
	case *ssa.Const:
		valInt, err := strconv.Atoi(v.Value.String())
		if err == nil {
			return []syscallSource{{num: valInt, source: traceCaller(edge, make([]*analyser.IssueSource, 0), 0, fset)}}
		}
	case *ssa.Global:
		result = append(result, traceInit(v, v.Pkg.Members, edge, fset)...)
	case *ssa.Parameter:
		prev := edge.Caller.In
		for _, p := range prev {
			result = append(result, traceSyscalls(p.Site.Common().Args[0], p, fset)...)
		}
	case *ssa.Phi:
		for _, val := range v.Edges {
			result = append(result, traceSyscalls(val, edge, fset)...)
		}
	case *ssa.Call:
		// Trace nested calls
		fn := v.Call.StaticCallee()
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				// Look for return instructions
				if ret, ok := instr.(*ssa.Return); ok {
					for _, val := range ret.Results {
						result = append(result, traceSyscalls(val, edge, fset)...)
					}
				}
			}
		}
	case *ssa.UnOp:
		result = append(result, traceSyscalls(v.X, edge, fset)...)
	case *ssa.Convert:
		result = append(result, traceSyscalls(v.X, edge, fset)...)
	case *ssa.FieldAddr:
		// check all instructions to get the latest value store for this field address
		var val ssa.Value
		for _, instr := range v.Block().Instrs {
			if st, ok := instr.(*ssa.Store); ok {
				if fe, ok := st.Addr.(*ssa.FieldAddr); ok {
					if fe.X == v.X {
						val = st.Val
					}
				}
			}
		}
		result = append(result, traceSyscalls(val, edge, fset)...)
	default:
		fmt.Printf("Unhandled value type: %T\n", v)
		panic("not handled")
	}
	return result
}

func traceInit(v *ssa.Global, members map[string]ssa.Member, edge *callgraph.Edge, fset *token.FileSet) (result []syscallSource) {
	// Iterate through instructions in the Init function
	// Iterate through all functions in the package to find the initialization
	for _, member := range members {
		if fn, ok := member.(*ssa.Function); ok {
			for _, block := range fn.Blocks {
				for _, instr := range block.Instrs {
					// Look for Store instructions
					if store, ok := instr.(*ssa.Store); ok {
						if store.Addr == v {
							result = append(result, traceSyscalls(store.Val, edge, fset)...)
						}
					}
				}
			}
		}
	}
	return result
}

// mainPackages returns the main packages to analyze.
// Each resulting package is named "main" and has a main function.
func mainPackages(pkgs []*ssa.Package) ([]*ssa.Package, error) {
	var mains []*ssa.Package
	for _, p := range pkgs {
		if p != nil && p.Pkg.Name() == "main" && p.Func("main") != nil {
			mains = append(mains, p)
		}
	}
	if len(mains) == 0 {
		return nil, fmt.Errorf("no main packages")
	}
	return mains, nil
}

// initFuncs returns all package init functions.
func initFuncs(pkgs []*ssa.Package) []*ssa.Function {
	var inits []*ssa.Function
	for _, p := range pkgs {
		if p == nil {
			continue
		}
		for name, member := range p.Members {
			fun, ok := member.(*ssa.Function)
			if !ok {
				continue
			}
			if name == "init" || strings.HasPrefix(name, "init#") {
				inits = append(inits, fun)
			}
		}
	}
	return inits
}

// traceCaller correctly tracks function calls in the execution stack.
func traceCaller(edge *callgraph.Edge, paths []*analyser.IssueSource, depth int, fset *token.FileSet) []*analyser.IssueSource {
	if edge == nil || edge.Caller == nil {
		return paths // Prevent nil pointer dereference
	}

	// Create a new IssueSource entry for this function call
	source := &analyser.IssueSource{
		File:     "undefined",
		Line:     0,
		AbsPath:  "undefined",
		Function: edge.Caller.Func.String(),
	}

	// Get file name, absolute path, and line number safely
	if edge.Site != nil {
		pos := edge.Site.Pos()
		position := fset.Position(pos)
		source.File = filepath.Base(position.Filename)
		source.Line = position.Line
		if position.Filename != "" {
			source.AbsPath = filepath.Clean(position.Filename)
		}
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

	// Stop recursion at desired depth to prevent infinite loops
	if depth >= 1 || len(edge.Caller.In) == 0 {
		return newPaths
	}

	// Recurse for previous function calls (callers)
	result := make([]*analyser.IssueSource, 0)
	for _, e := range edge.Caller.In {
		result = append(result, traceCaller(e, newPaths, depth+1, fset)...)
	}

	return result
}

//nolint:unused
func traceCallerToRoot(edge *callgraph.Edge, paths []*analyser.IssueSource, depth int, fset *token.FileSet) []*analyser.IssueSource {
	if edge == nil || edge.Caller == nil {
		return paths // Prevent nil pointer dereference
	}
	// Create a new IssueSource entry for this function call
	source := &analyser.IssueSource{
		File:     "undefined",
		Line:     0,
		AbsPath:  "undefined",
		Function: edge.Caller.Func.String(),
	}
	// Get file name, absolute path, and line number safely
	if edge.Site != nil {
		pos := edge.Site.Pos()
		position := fset.Position(pos)
		source.File = filepath.Base(position.Filename)
		source.Line = position.Line
		if position.Filename != "" {
			source.AbsPath = filepath.Clean(position.Filename)
		}
	}

	if len(edge.Caller.In) == 0 && (source.Function == "command-line-arguments.main" ||
		source.Function == "command-line-arguments.init") {
		return []*analyser.IssueSource{source}
	}
	if len(edge.Caller.In) == 0 || depth == 10 {
		return nil
	}
	results := make([]*analyser.IssueSource, 0)
	for _, e := range edge.Caller.In {
		res := traceCaller(e, paths, depth+1, fset)
		if len(res) > 0 {
			results = append(results, res...)
			break
		}
	}
	finalResults := make([]*analyser.IssueSource, 0)
	for _, result := range results {
		src := source.Copy()
		src.CallStack = result
		finalResults = append(finalResults, src)
	}
	return finalResults
}
