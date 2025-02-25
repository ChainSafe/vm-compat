// Package syscall implements analyser. Analyze for detecting syscalls
package syscall

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ChainSafe/vm-compat/analyzer"
	"github.com/ChainSafe/vm-compat/asmparser"
	"github.com/ChainSafe/vm-compat/asmparser/mips"
	"github.com/ChainSafe/vm-compat/common"
	"github.com/ChainSafe/vm-compat/profile"
)

const (
	analyzerWorkingPrincipalURL = "https://github.com/ChainSafe/vm-compat?tab=readme-ov-file#how-it-works"
	potentialImpactMsg          = `This syscall is present in the program, but its execution depends on the actual runtime behavior. 
             If the execution path does not reach this syscall, it may not affect execution.`
)

// asmSyscallAnalyser analyzes system calls in assembly files.
type asmSyscallAnalyser struct {
	profile *profile.VMProfile
}

// NewAssemblySyscallAnalyser initializes an analyser for assembly syscalls.
func NewAssemblySyscallAnalyser(profile *profile.VMProfile) analyzer.Analyzer {
	return &asmSyscallAnalyser{profile: profile}
}

// Analyze scans an assembly file for syscalls and detects compatibility issues.
//
//nolint:cyclop
func (a *asmSyscallAnalyser) Analyze(path string, withTrace bool) ([]*analyzer.Issue, error) {
	callGraph, err := a.buildCallGraph(path)
	if err != nil {
		return nil, err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	issues := make([]*analyzer.Issue, 0)
	// Iterate through segments and check for syscall.
	for _, segment := range callGraph.Segments() {
		for _, instruction := range segment.Instructions() {
			if !instruction.IsSyscall() {
				continue
			}
			syscalls, err := callGraph.RetrieveSyscallNum(segment, instruction)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve syscall number: %w", err)
			}
			for _, syscall := range syscalls {
				// Categorize syscall
				if slices.Contains(a.profile.AllowedSycalls, syscall.Number) {
					continue
				}
				source, err := common.TraceAsmCaller(absPath, callGraph, syscall.Segment.Label(), endCondition)
				if err != nil { // non-reachable portion ignored
					continue
				}
				if !withTrace {
					source.CallStack = nil
				}

				severity := analyzer.IssueSeverityCritical
				message := fmt.Sprintf("Potential Incompatible Syscall Detected: %d", syscall.Number)
				if slices.Contains(a.profile.NOOPSyscalls, syscall.Number) {
					message = fmt.Sprintf("Potential NOOP Syscall Detected: %d", syscall.Number)
					severity = analyzer.IssueSeverityWarning
				}

				issues = append(issues, &analyzer.Issue{
					Severity:  severity,
					Message:   message,
					CallStack: source,
					Impact:    potentialImpactMsg,
					Reference: analyzerWorkingPrincipalURL,
				})
			}
		}
	}
	return issues, nil
}

func (a *asmSyscallAnalyser) buildCallGraph(path string) (asmparser.CallGraph, error) {
	var (
		err       error
		callGraph asmparser.CallGraph
	)

	// Select the correct parser based on architecture.
	switch a.profile.GOARCH {
	case "mips", "mips64":
		callGraph, err = mips.NewParser().Parse(path)
	default:
		return nil, fmt.Errorf("unsupported GOARCH: %s", a.profile.GOARCH)
	}
	if err != nil {
		return nil, fmt.Errorf("error parsing assembly file: %w", err)
	}
	return callGraph, nil
}

// TraceStack generates callstack for a function to debug
func (a *asmSyscallAnalyser) TraceStack(path string, function string) (*analyzer.CallStack, error) {
	graph, err := a.buildCallGraph(path)
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return common.TraceAsmCaller(absPath, graph, function, endCondition)
}

func endCondition(function string) bool {
	return function == "runtime.rt0_go" || // start point of a go program
		function == "main.main" || // main
		strings.Contains(function, ".init.") || // all init functions
		strings.HasSuffix(function, ".init") // vars
}
