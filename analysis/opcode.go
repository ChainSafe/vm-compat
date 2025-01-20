package analysis

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChainSafe/vm-compat/opcode"
	"github.com/ChainSafe/vm-compat/opcode/mips"
	"github.com/ChainSafe/vm-compat/profile"
)

func AnalyseOpcodes(profile *profile.VMProfile, paths ...string) error {
	for _, path := range paths {
		err := analyzeOpcodes(profile, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func analyzeOpcodes(profile *profile.VMProfile, path string) error {
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

	opcodeAnalyzerProvider, err := newProvider(profile.Arch, profile)
	if err != nil {
		fmt.Printf("Error getting provider: %v\n", err)
		return err
	}

	scanner := bufio.NewScanner(codefile)
	for scanner.Scan() {
		line := scanner.Text()
		instructionDetected, err := opcodeAnalyzerProvider.ParseOpcode(line)
		if err != nil {
			fmt.Printf("Error parsing line: %s: %v\n", line, err)
			return err
		}

		if instructionDetected == nil {
			continue
		}

		if !opcodeAnalyzerProvider.IsAllowedOpcode(instructionDetected.Opcode) {
			fmt.Println("Incompatible Opcode Detected: ", fmt.Sprintf("0x%x", instructionDetected.Opcode))
		}
	}
	return nil
}

func newProvider(arch string, prof *profile.VMProfile) (opcode.Provider, error) {
	switch arch {
	case "mips32":
		return mips.NewProvider(opcode.ArchMIPS32Bit, prof), nil
	case "mips64":
		return mips.NewProvider(opcode.ArchMIPS64Bit, prof), nil
	default:
		return nil, fmt.Errorf("unsupported architecture: %d", arch)
	}
}
