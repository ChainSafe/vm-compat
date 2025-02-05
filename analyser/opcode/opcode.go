package opcode

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ChainSafe/vm-compat/analyser"
	"github.com/ChainSafe/vm-compat/asmparser"
	"github.com/ChainSafe/vm-compat/asmparser/mips"
	"github.com/ChainSafe/vm-compat/profile"
)

var ignoreSegments = []string{
	"runtime.gcenable",
	"runtime.init.5",            // patch out: init() { go forcegchelper() }
	"runtime.main.func1",        // patch out: main.func() { newm(sysmon, ....) }
	"runtime.deductSweepCredit", // uses floating point nums and interacts with gc we disabled
	"runtime.(*gcControllerState).commit",
	"github.com/prometheus/client_golang/prometheus.init",
	"github.com/prometheus/client_golang/prometheus.init.0",
	"github.com/prometheus/procfs.init",
	"github.com/prometheus/common/model.init",
	"github.com/prometheus/client_model/go.init",
	"github.com/prometheus/client_model/go.init.0",
	"github.com/prometheus/client_model/go.init.1",
	"flag.init",
	"runtime.check",
}

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
			if !op.isAllowedOpcode(instruction.OpcodeHex(), instruction.Funct()) {
				issues = append(issues, &analyser.Issue{
					File:   path,
					Source: instruction.Address(), // TODO: add proper source
					Message: fmt.Sprintf("Incompatible Opcode Detected: Opcode: %s, Funct: %s",
						instruction.OpcodeHex(), instruction.Funct()),
				})
			}
		}
	}
	return issues, nil
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

func shouldIgnoreSegment(callGraph asmparser.CallGraph, segment asmparser.Segment) bool {
	parents := callGraph.ParentsOf(segment)
	if len(parents) == 0 {
		return false
	}
	for _, parent := range parents {
		if slices.Contains(ignoreSegments, parent.Label()) {
			return true
		}
		if shouldIgnoreSegment(callGraph, parent) {
			return true
		}
	}
	return false
}
