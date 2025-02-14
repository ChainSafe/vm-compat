package syscall

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/ChainSafe/vm-compat/analyzer"
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
func NewGOSyscallAnalyser(profile *profile.VMProfile) analyzer.Analyzer {
	return &goSyscallAnalyser{profile: profile}
}

// Analyze scans a Go binary for syscalls and detects compatibility issues.
//
//nolint:cyclop
func (a *goSyscallAnalyser) Analyze(path string, withTrace bool) ([]*analyzer.Issue, error) {
	cg, fset, err := a.buildCallGraph(path)
	if err != nil {
		return nil, err
	}
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
	issues := make([]*analyzer.Issue, 0)
	functions := make([]string, 0)
	for _, syscall := range syscalls {
		functions = append(functions, syscall.source.Function)
	}
	tracker := a.reachableFunctions(cg, functions)
	stackTraceMap := make(map[string]*analyzer.IssueSource)
	for i := range syscalls {
		syscll := syscalls[i]
		if slices.Contains(a.profile.AllowedSycalls, syscll.num) || !tracker[syscll.source.Function] {
			continue
		}
		stackTrace := syscll.source
		if withTrace {
			stackTrace = stackTraceMap[syscll.source.Function]
			if stackTrace == nil {
				stackTrace, _ = a.trackStack(cg, fset, syscll.source.Function)
				stackTraceMap[syscll.source.Function] = stackTrace
			}
		}

		severity := analyzer.IssueSeverityCritical
		message := fmt.Sprintf("Potential Incompatible Syscall Detected: %d", syscll.num)
		if slices.Contains(a.profile.NOOPSyscalls, syscll.num) {
			severity = analyzer.IssueSeverityWarning
			message = fmt.Sprintf("Potential NOOP Syscall Detected: %d", syscll.num)
		}

		issues = append(issues, &analyzer.Issue{
			Severity: severity,
			Sources:  stackTrace,
			Message:  message,
		})
	}

	return issues, nil
}

func (a *goSyscallAnalyser) TraceStack(path string, function string) (*analyzer.IssueSource, error) {
	cg, fset, err := a.buildCallGraph(path)
	if err != nil {
		return nil, err
	}
	return a.trackStack(cg, fset, function)
}

func (a *goSyscallAnalyser) trackStack(cg *callgraph.Graph, fset *token.FileSet, function string) (*analyzer.IssueSource, error) {
	seen := make(map[*callgraph.Node]bool)
	var visit func(n *callgraph.Node, edge *callgraph.Edge) *analyzer.IssueSource
	visit = func(n *callgraph.Node, edge *callgraph.Edge) *analyzer.IssueSource {
		var src *analyzer.IssueSource
		if edge != nil && edge.Caller != nil && edge.Site != nil {
			position := fset.Position(edge.Site.Pos())
			fn := edge.Caller.Func.String()
			src = &analyzer.IssueSource{
				File:     position.Filename,
				Line:     position.Line,
				Function: fn,
				AbsPath:  filepath.Clean(position.Filename),
			}
			if fn == function {
				return src
			}
		}
		// as we are checking edge.Caller we need to get 1 step deeper everytime, that requires to re-visit the node
		if seen[n] {
			return nil
		}
		seen[n] = true

		for _, e := range n.Out {
			ch := visit(e.Callee, e)
			if ch != nil {
				if src != nil {
					ch.AddCallStack(src)
				}
				return ch
			}
		}
		return nil
	}

	for _, n := range cg.Nodes {
		if n.Func.String() == "command-line-arguments.main" || n.Func.String() == "command-line-arguments.init" {
			if source := visit(n, nil); source != nil {
				return source, nil
			}
		}
	}
	return nil, fmt.Errorf("no trace found to root for the given function")
}

func (a *goSyscallAnalyser) buildCallGraph(path string) (*callgraph.Graph, *token.FileSet, error) {
	// Find the Go module root for correct context
	modRoot, err := common.FindGoModuleRoot(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find Go module root: %w", err)
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
		return nil, nil, err
	}
	if packages.PrintErrors(initial) > 0 {
		return nil, nil, fmt.Errorf("packages contain errors")
	}

	// Create and build SSA-form program representation.
	mode := ssa.InstantiateGenerics
	prog, _ := ssautil.AllPackages(initial, mode)
	prog.Build()

	// Construct call graph using RTA analysis.
	mains, err := mainPackages(prog.AllPackages())
	if err != nil {
		return nil, nil, err
	}
	roots := make([]*ssa.Function, 0)
	for _, main := range mains {
		roots = append(roots, main.Func("main"))
	}
	roots = append(roots, initFuncs(prog.AllPackages())...)

	cg := rta.Analyze(roots, true).CallGraph
	cg.DeleteSyntheticNodes()

	return cg, initial[0].Fset, nil
}

func (a *goSyscallAnalyser) reachableFunctions(cg *callgraph.Graph, functions []string) map[string]bool {
	seen := make(map[*callgraph.Node]bool)
	tracker := make(map[string]bool)

	var visit func(n *callgraph.Node)
	visit = func(n *callgraph.Node) {
		if seen[n] {
			return
		}
		seen[n] = true

		if slices.Contains(functions, n.Func.String()) {
			tracker[n.Func.String()] = true
		}

		for _, e := range n.Out {
			visit(e.Callee)
		}
	}

	for _, n := range cg.Nodes {
		if n.Func.String() == "command-line-arguments.main" || n.Func.String() == "command-line-arguments.init" {
			visit(n)
		}
	}
	return tracker
}

type syscallSource struct {
	num    int
	source *analyzer.IssueSource
}

func traceSyscalls(value ssa.Value, edge *callgraph.Edge, fset *token.FileSet) []syscallSource {
	result := make([]syscallSource, 0)
	switch v := value.(type) {
	case *ssa.Const:
		valInt, err := strconv.Atoi(v.Value.String())
		if err == nil {
			position := fset.Position(edge.Site.Pos())
			return []syscallSource{{num: valInt,
				source: &analyzer.IssueSource{
					File:     position.Filename,
					Line:     position.Line,
					Function: edge.Caller.Func.String(),
					AbsPath:  filepath.Clean(position.Filename),
				},
			}}
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
