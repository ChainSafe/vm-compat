package analysis

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/ChainSafe/vm-compat/profile"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func AnalyseSyscalls(profile *profile.VMProfile, paths ...string) error {
	cfg := &packages.Config{
		Mode:       packages.LoadAllSyntax,
		BuildFlags: []string{},
		Env: append(
			os.Environ(),
			fmt.Sprintf("GOOS=%s", profile.GOOS),
			fmt.Sprintf("GOARCH=%s", profile.GOARCH),
		),
	}

	initial, err := packages.Load(cfg, paths...)
	if err != nil {
		return err
	}
	if packages.PrintErrors(initial) > 0 {
		return fmt.Errorf("packages contain errors")
	}

	// Create and build SSA-form program representation.
	mode := ssa.InstantiateGenerics // instantiate generics by default for soundness
	prog, _ := ssautil.AllPackages(initial, mode)
	prog.Build()

	// -- rta call graph construction ------------------------------------------

	mains, err := mainPackages(prog.AllPackages())
	if err != nil {
		return err
	}
	roots := make([]*ssa.Function, 0)
	for _, main := range mains {
		roots = append(roots, main.Func("main"))
	}
	roots = append(roots, initFuncs(prog.AllPackages())...)

	cg := rta.Analyze(roots, true).CallGraph
	cg.DeleteSyntheticNodes()

	// Analyze the call graph for syscalls
	syscalls := make([]int, 0)
	err = callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		callee := edge.Callee.Func
		if callee != nil && callee.Pkg != nil && callee.Pkg.Pkg != nil {
			packagePath := callee.Pkg.Pkg.Path()
			switch packagePath {
			case "syscall":
				if callee.Name() == "RawSyscall6" {
					fmt.Println("---------------------------------")
					fmt.Printf("From: %s\n", edge.Caller.Func)
					calls := traceSyscalls(nil, edge)
					fmt.Printf("SYSCODES: %v \n", calls)
					syscalls = append(syscalls, calls...)
				}
			case "unix":

			default:
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Println("---------------")
	for _, sycall := range syscalls {
		if !slices.Contains(profile.AllowedSycalls, sycall) {
			fmt.Println("Restricted syscall detected:", sycall)
		}
	}

	return nil
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
		// Iterate through instructions in the Init function
		// Iterate through all functions in the package to find the initialization
		for _, member := range v.Pkg.Members {
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
