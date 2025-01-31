package opcode

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ChainSafe/vm-compat/opcode/common"
	"github.com/ChainSafe/vm-compat/opcode/mips"
	"github.com/ChainSafe/vm-compat/profile"
)

type opcode struct {
	Profile *profile.VMProfile
}

func NewAnalyzer(profile *profile.VMProfile) Analyzer {
	return &opcode{Profile: profile}
}

func (a *opcode) Run(path string) error {
	// return the absolute path of the given path
	fpath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("Error getting the absolute filepath: %s: %v\n", path, err)
		return err
	}

	codefile, err := os.Open(fpath)
	if err != nil {
		fmt.Printf("Error opening filepath: %s: %v\n", fpath, err)
		return err
	}
	defer codefile.Close()

	opcodeAnalyzerProvider, err := newProvider(a.Profile.GOARCH, a.Profile)
	if err != nil {
		fmt.Printf("Error getting provider: %v\n", err)
		return err
	}

	scanner := bufio.NewScanner(codefile)
	invalidOpcodeDetected := make(map[uint64]bool)
	for scanner.Scan() {
		line := scanner.Text()
		instructionDetected, err := opcodeAnalyzerProvider.ParseAssembly(line)
		if err != nil {
			fmt.Printf("Error parsing line: %s: %v\n", line, err)
			return err
		}

		if instructionDetected == nil {
			continue
		}

		if !opcodeAnalyzerProvider.IsAllowedOpcode(instructionDetected.Opcode, instructionDetected.Funct) {
			if !invalidOpcodeDetected[instructionDetected.Opcode] {
				fmt.Printf("Incompatible Opcode Detected. Opcode: %s, fun: %s \n", fmt.Sprintf("0x%s", strconv.FormatInt(int64(instructionDetected.Opcode), 16)), fmt.Sprintf("0x%s", strconv.FormatInt(int64(instructionDetected.Funct), 16)))
			}
			invalidOpcodeDetected[instructionDetected.Opcode] = true
		}
	}
	return nil
}

func newProvider(arch string, prof *profile.VMProfile) (Provider, error) {
	switch arch {
	case "mips32":
		return mips.NewProvider(common.ArchMIPS32Bit, prof), nil
	case "mips64":
		return mips.NewProvider(common.ArchMIPS64Bit, prof), nil
	default:
		return nil, fmt.Errorf("unsupported architecture: %s", arch)
	}
}
