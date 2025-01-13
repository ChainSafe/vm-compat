package analysis

import (
	"fmt"
	"strings"

	"github.com/ChainSafe/vm-compat/profile"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func AnalyseSyscalls(profile *profile.VMProfile, paths ...string) error {
	cfg := &packages.Config{
		Mode:       packages.LoadAllSyntax,
		BuildFlags: []string{},
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

	// -- call graph construction ------------------------------------------

	cg := static.CallGraph(prog)
	cg.DeleteSyntheticNodes()

	err = callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		funcName := edge.Callee.Func.String()
		if isSyscall(funcName) {
			if isRestrictedSyscall(funcName, profile.RestrictedSyscalls) {
				fmt.Printf("Restricted syscall detected: %s\n", funcName)
			}
		}
		return nil
	})
	return err
}

// isRestrictedSyscall checks if the function name matches a restricted syscall.
func isRestrictedSyscall(funcName string, restricted []string) bool {
	for _, syscall := range restricted {
		if strings.EqualFold(funcName, syscall) {
			return true
		}
	}
	return false
}

// isSyscall checks if the function call is a syscall.
func isSyscall(funcName string) bool {
	// TODO: check other system call packages
	return strings.Contains(funcName, "syscall.")
}
