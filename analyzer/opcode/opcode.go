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
	callGraph, err := op.buildCallGraph(path)
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
				source, err := common.TraceAsmCaller(
					absPath,
					callGraph,
					segment.Label(),
					common.ProgramEntrypoint(op.profile.GOARCH),
				)
				if err != nil { // non-reachable portion ignored
					continue
				}
				if !withTrace {
					source.CallStack = nil
				}

				issue := &analyzer.Issue{
					Severity:  analyzer.IssueSeverityCritical,
					CallStack: source,
					Message: fmt.Sprintf("Potential Incompatible Opcode Detected: Opcode: %s, Funct: %s",
						instruction.OpcodeHex(), instruction.Funct()),
				}
				if common.ShouldIgnoreSource(source, op.profile.IgnoredFunctions) {
					issue.Severity = analyzer.IssueSeverityWarning
				}
				issues = append(issues, issue)
			}
		}
	}
	return issues, nil
}

func (op *opcode) buildCallGraph(path string) (asmparser.CallGraph, error) {
	var (
		err       error
		callGraph asmparser.CallGraph
	)

	// Select the correct parser based on architecture.
	switch op.profile.GOARCH {
	case "mips", "mips64":
		callGraph, err = mips.NewParser().Parse(path)
	default:
		return nil, fmt.Errorf("unsupported GOARCH: %s", op.profile.GOARCH)
	}
	if err != nil {
		return nil, fmt.Errorf("error parsing assembly file: %w", err)
	}
	return callGraph, nil
}

// TraceStack generates callstack for a function to debug
func (op *opcode) TraceStack(path string, function string) (*analyzer.CallStack, error) {
	graph, err := op.buildCallGraph(path)
	if err != nil {
		return nil, err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return common.TraceAsmCaller(absPath, graph, function, common.ProgramEntrypoint(op.profile.GOARCH))
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
