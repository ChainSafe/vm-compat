package syscall

import (
	"fmt"
	"os"
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
	"runtime/internal/syscall.Syscall6",
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
	syscalls := make([]int, 0)
	err = callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		callee := edge.Callee.Func
		if callee != nil && callee.Pkg != nil && callee.Pkg.Pkg != nil {
			packagePath := callee.Pkg.Pkg.Path()
			if packagePath == "syscall" && callee.Name() == "RawSyscall6" {
				calls := traceSyscalls(nil, edge)
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
	for _, syscallNum := range syscalls {
		// Categorize syscall
		if slices.Contains(a.profile.AllowedSycalls, syscallNum) {
			continue
		}

		message := fmt.Sprintf("Incompatible Syscall Detected: 0x%x", syscallNum)
		if slices.Contains(a.profile.NOOPSyscalls, syscallNum) {
			message = fmt.Sprintf("NOOP Syscall Detected: 0x%x", syscallNum)
		}

		issues = append(issues, &analyser.Issue{
			File:    path,
			Message: message,
		})
	}

	return issues, nil
}

func traceSyscalls(value ssa.Value, edge *callgraph.Edge) []int {
	if edge != nil {
		args := edge.Site.Common().Args
		if len(args) > 0 {
			value = args[0]
		}
	}
	result := make([]int, 0)
	switch v := value.(type) {
	case *ssa.Const:
		valInt, err := strconv.Atoi(v.Value.String())
		if err == nil {
			return []int{valInt}
		}
	case *ssa.Global:
		result = append(result, traceInit(v, v.Pkg.Members)...)
	case *ssa.Parameter:
		prev := edge.Caller.In
		for _, p := range prev {
			result = append(result, traceSyscalls(nil, p)...)
		}
	case *ssa.Phi:
		for _, val := range v.Edges {
			result = append(result, traceSyscalls(val, nil)...)
		}
	case *ssa.Call:
		// Trace nested calls
		fn := v.Call.StaticCallee()
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				// Look for return instructions
				if ret, ok := instr.(*ssa.Return); ok {
					for _, val := range ret.Results {
						result = append(result, traceSyscalls(val, nil)...)
					}
				}
			}
		}
	case *ssa.UnOp:
		result = append(result, traceSyscalls(v.X, nil)...)
	case *ssa.Convert:
		result = append(result, traceSyscalls(v.X, nil)...)
	default:
		fmt.Printf("Unhandled value type: %T\n", v)
		panic("not handled")
	}
	return result
}

func traceInit(v *ssa.Global, members map[string]ssa.Member) (result []int) {
	// Iterate through instructions in the Init function
	// Iterate through all functions in the package to find the initialization
	for _, member := range members {
		if fn, ok := member.(*ssa.Function); ok {
			for _, block := range fn.Blocks {
				for _, instr := range block.Instrs {
					// Look for Store instructions
					if store, ok := instr.(*ssa.Store); ok {
						if store.Addr == v {
							result = append(result, traceSyscalls(store.Val, nil)...)
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
