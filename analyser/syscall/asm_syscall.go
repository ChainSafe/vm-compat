// Package syscall implements analyser. Analyze for detecting syscalls
package syscall

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/ChainSafe/vm-compat/analyser"
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
func NewAssemblySyscallAnalyser(profile *profile.VMProfile) analyser.Analyzer {
	return &asmSyscallAnalyser{profile: profile}
}

// Analyze scans an assembly file for syscalls and detects compatibility issues.
//
//nolint:cyclop
func (a *asmSyscallAnalyser) Analyze(path string) ([]*analyser.Issue, error) {
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

	issues := make([]*analyser.Issue, 0)

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

			severity := analyser.IssueSeverityCritical
			message := fmt.Sprintf("Incompatible Syscall Detected: %d", syscallNum)
			if slices.Contains(a.profile.NOOPSyscalls, syscallNum) {
				message = fmt.Sprintf("NOOP Syscall Detected: %d", syscallNum)
				severity = analyser.IssueSeverityWarning
			}

			issues = append(issues, &analyser.Issue{
				Severity: severity,
				Message:  message,
				Sources:  common.TraceAsmCaller(absPath, instruction, segment, callGraph, make([]*analyser.IssueSource, 0), 0),
			})
		}
	}
	return issues, nil
}
