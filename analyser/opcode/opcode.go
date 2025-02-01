package opcode

import (
	"fmt"
	"slices"

	"github.com/ChainSafe/vm-compat/analyser"
	"github.com/ChainSafe/vm-compat/asmparser"
	"github.com/ChainSafe/vm-compat/asmparser/mips"
	"github.com/ChainSafe/vm-compat/profile"
)

type opcode struct {
	profile *profile.VMProfile
}

func NewAnalyser(profile *profile.VMProfile) analyser.Analyzer {
	return &opcode{profile: profile}
}

func (op *opcode) Analyze(path string) ([]*analyser.Issue, error) {
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
	issues := make([]*analyser.Issue, 0)
	for _, segment := range callGraph.Segments() {
		for _, instruction := range segment.Instructions() {
			if !op.isAllowedOpcode(instruction.Opcode(), instruction.Funct()) {
				// TODO: add funct in issue
				issues = append(issues, &analyser.Issue{
					File:    path,
					Segment: segment.Label(),
					Message: fmt.Sprintf("Incompatible Opcode Detected: 0x%x", instruction.Opcode()),
				})
			}
		}
	}
	return issues, nil
}

func (op *opcode) isAllowedOpcode(opcode, funct string) bool {
	return slices.ContainsFunc(op.profile.AllowedOpcodes, func(instr profile.OpcodeInstruction) bool {
		if instr.Opcode != opcode {
			return false
		}
		if len(instr.Funct) == 0 {
			return funct == ""
		}
		return slices.Contains(instr.Funct, funct)
	})
}
