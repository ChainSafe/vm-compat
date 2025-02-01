package syscall

import (
	"fmt"
	"github.com/ChainSafe/vm-compat/analyser"
	"github.com/ChainSafe/vm-compat/asmparser"
	"github.com/ChainSafe/vm-compat/asmparser/mips"
	"github.com/ChainSafe/vm-compat/profile"
)

type asmSyscallAnalyser struct {
	profile *profile.VMProfile
}

func NewAssemblySyscallAnalyser(profile *profile.VMProfile) analyser.Analyser {
	return &asmSyscallAnalyser{profile: profile}
}

func (a *asmSyscallAnalyser) Analyse(path string) ([]*analyser.Issue, error) {
	var err error
	var callGraph asmparser.CallGraph

	switch a.profile.GOARCH {
	case "mips32", "mips64":
		callGraph, err = mips.NewParser().Parse(path)
	default:
		return nil, fmt.Errorf("unsupported GOARCH %s", a.profile.GOARCH)

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
