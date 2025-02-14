// Package opcode implements analyzer.Analyzer for detecting incompatible opcodes.
package opcode

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

type opcode struct {
	profile *profile.VMProfile
}

func NewAnalyser(profile *profile.VMProfile) analyzer.Analyzer {
	return &opcode{profile: profile}
}

func (op *opcode) Analyze(path string, withTrace bool) ([]*analyzer.Issue, error) {
	var err error
	var callGraph asmparser.CallGraph

	switch op.profile.GOARCH {
	case "mips32", "mips64":
		callGraph, err = mips.NewParser().Parse(path)
	default:
		return nil, fmt.Errorf("unsupported GOARCH %s", op.profile.GOARCH)
	}
	if err != nil {
		return nil, err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	issues := make([]*analyzer.Issue, 0)
	for _, segment := range callGraph.Segments() {
		for _, instruction := range segment.Instructions() {
			if !op.isAllowedOpcode(instruction.OpcodeHex(), instruction.Funct()) {
				source, err := common.TraceAsmCaller(absPath, callGraph, segment.Label())
				if err != nil { // non-reachable portion ignored
					continue
				}
				if !withTrace {
					source.CallStack = nil
				}
				issues = append(issues, &analyzer.Issue{
					Severity: analyzer.IssueSeverityCritical,
					Sources:  source,
					Message: fmt.Sprintf("Incompatible Opcode Detected: Opcode: %s, Funct: %s",
						instruction.OpcodeHex(), instruction.Funct()),
				})
			}
		}
	}
	return issues, nil
}

// TraceStack generates callstack for a function to debug
func (op *opcode) TraceStack(path string, function string) (*analyzer.IssueSource, error) {
	return nil, fmt.Errorf("stack trace is not supported for assembly code")
}

func (op *opcode) isAllowedOpcode(opcode, funct string) bool {
	return slices.ContainsFunc(op.profile.AllowedOpcodes, func(instr profile.OpcodeInstruction) bool {
		if !strings.EqualFold(instr.Opcode, opcode) {
			return false
		}
		if len(instr.Funct) == 0 {
			return funct == ""
		}
		return slices.ContainsFunc(instr.Funct, func(s string) bool {
			return strings.EqualFold(s, funct)
		})
	})
}
