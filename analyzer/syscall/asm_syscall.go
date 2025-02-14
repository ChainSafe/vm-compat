// Package syscall implements analyser. Analyze for detecting syscalls
package syscall

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/ChainSafe/vm-compat/analyzer"
	"github.com/ChainSafe/vm-compat/asmparser"
	"github.com/ChainSafe/vm-compat/asmparser/mips"
	"github.com/ChainSafe/vm-compat/common"
	"github.com/ChainSafe/vm-compat/profile"
)

var syscallAPISForAsm = append(syscallAPIs, "runtime/internal/syscall.Syscall6")

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
	var (
		err       error
		callGraph asmparser.CallGraph
	)

	// Select the correct parser based on architecture.
	switch a.profile.GOARCH {
	case "mips32", "mips64":
		callGraph, err = mips.NewParser().Parse(path)
	default:
		return nil, fmt.Errorf("unsupported GOARCH: %s", a.profile.GOARCH)
	}
	if err != nil {
		return nil, fmt.Errorf("error parsing assembly file: %w", err)
	}

	issues := make([]*analyzer.Issue, 0)

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// Iterate through segments and check for syscall.
	for _, segment := range callGraph.Segments() {
		segmentLabel := segment.Label()
		for _, instruction := range segment.Instructions() {
			if !instruction.IsSyscall() {
				continue
			}
			// Ignore indirect syscall calling from syscall apis
			if slices.Contains(syscallAPISForAsm, segmentLabel) {
				continue
			}
			syscallNum, err := segment.RetrieveSyscallNum(instruction)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve syscall number: %w", err)
			}

			// Categorize syscall
			if slices.Contains(a.profile.AllowedSycalls, syscallNum) {
				continue
			}
			// Better to develop a new algo to check all segments at once like go_syscall
			source, err := common.TraceAsmCaller(absPath, callGraph, segment.Label())
			if err != nil { // non-reachable portion ignored
				continue
			}
			if !withTrace {
				source.CallStack = nil
			}

			severity := analyzer.IssueSeverityCritical
			message := fmt.Sprintf("Potential Incompatible Syscall Detected: %d", syscallNum)
			if slices.Contains(a.profile.NOOPSyscalls, syscallNum) {
				message = fmt.Sprintf("Potential NOOP Syscall Detected: %d", syscallNum)
				severity = analyzer.IssueSeverityWarning
			}

			issues = append(issues, &analyzer.Issue{
				Severity: severity,
				Message:  message,
				Sources:  source,
			})
		}
	}
	return issues, nil
}

// TraceStack generates callstack for a function to debug
func (a *asmSyscallAnalyser) TraceStack(path string, function string) (*analyzer.IssueSource, error) {
	return nil, fmt.Errorf("stack trace is not supported for assembly code")
}
