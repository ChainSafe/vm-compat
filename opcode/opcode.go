package opcode

import (
	"github.com/ChainSafe/vm-compat/analyser"
	"github.com/ChainSafe/vm-compat/opcode/common"
	"github.com/ChainSafe/vm-compat/profile"
)

type Provider interface {
	ParseAssembly(line string) (*common.Instruction, error)
	IsAllowedOpcode(code uint64) bool
}

func AnalyseOpcodes(profile *profile.VMProfile, path string) ([]analyser.Issue, error) {
	analysisProvider := NewAnalyzer(profile)
	return analysisProvider.Analyze(path)
}
