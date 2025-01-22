package opcode

import (
	"github.com/ChainSafe/vm-compat/opcode/common"
	"github.com/ChainSafe/vm-compat/profile"
)

type Provider interface {
	ParseAssembly(line string) (*common.Instruction, error)
	IsAllowedOpcode(code uint64) bool
}

type Analyzer interface {
	Run(path string) error
}

func AnalyseOpcodes(profile *profile.VMProfile, paths ...string) error {
	analysisProvider := NewAnalyzer(profile)
	for _, path := range paths {
		err := analysisProvider.Run(path)
		if err != nil {
			return err
		}
	}
	return nil
}
