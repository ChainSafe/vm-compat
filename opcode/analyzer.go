package opcode

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChainSafe/vm-compat/analyser"
	"github.com/ChainSafe/vm-compat/opcode/common"
	"github.com/ChainSafe/vm-compat/opcode/mips"
	"github.com/ChainSafe/vm-compat/profile"
)

type opcode struct {
	Profile *profile.VMProfile
}

func NewAnalyzer(profile *profile.VMProfile) analyser.Analyser {
	return &opcode{Profile: profile}
}

func (a *opcode) Analyze(path string) ([]analyser.Issue, error) {
	// return the absolute path of the given path
	fpath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("Error getting the absolute filepath: %s: %v\n", path, err)
		return nil, err
	}

	codefile, err := os.Open(fpath)
	if err != nil {
		fmt.Printf("Error opening filepath: %s: %v\n", fpath, err)
		return nil, err
	}
	defer codefile.Close()

	opcodeAnalyzerProvider, err := newProvider(a.Profile.GOARCH, a.Profile)
	if err != nil {
		fmt.Printf("Error getting provider: %v\n", err)
		return nil, err
	}

	scanner := bufio.NewScanner(codefile)
	issues := []analyser.Issue{}
	for scanner.Scan() {
		line := scanner.Text()
		instructionDetected, err := opcodeAnalyzerProvider.ParseAssembly(line)
		if err != nil {
			fmt.Printf("Error parsing line: %s: %v\n", line, err)
			return nil, err
		}

		if instructionDetected == nil {
			continue
		}

		if !opcodeAnalyzerProvider.IsAllowedOpcode(instructionDetected.Opcode) {
			issues = append(issues, analyser.Issue{
				Message: fmt.Sprintf("Incompatible Opcode Detected: 0x%x", instructionDetected.Opcode),
			})
		}
	}
	return issues, nil
}

func newProvider(arch string, prof *profile.VMProfile) (Provider, error) {
	switch arch {
	case "mips":
		return mips.NewProvider(common.ArchMIPS32Bit, prof), nil
	case "mips64":
		return mips.NewProvider(common.ArchMIPS64Bit, prof), nil
	default:
		return nil, fmt.Errorf("unsupported architecture: %s", arch)
	}
}
