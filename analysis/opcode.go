package analysis

import (
	"github.com/ChainSafe/vm-compat/opcode/analyzer"
	"github.com/ChainSafe/vm-compat/profile"
)

func AnalyseOpcodes(profile *profile.VMProfile, paths ...string) error {
	analysisProvider := analyzer.NewAnalyzer(profile)
	for _, path := range paths {
		err := analysisProvider.AnalyzeOpcodes(path)
		if err != nil {
			return err
		}
	}
	return nil
}
