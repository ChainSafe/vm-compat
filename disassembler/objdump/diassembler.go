package objdump

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ChainSafe/vm-compat/disassembler"
)

type Objdump struct {
	Arch string
	GOOS string
}

func New(goos, arch string) *Objdump {
	return &Objdump{
		Arch: arch,
		GOOS: goos,
	}
}

func (o *Objdump) Disassemble(mode disassembler.Source, target string, outputPath string) (string, error) {
	var disassembly string
	var err error

	switch mode {
	case disassembler.SourceBinary:
		disassembly, err = generateBinaryDisassembly(target, o.GOOS, o.Arch)
		if err != nil {
			return "", err
		}
	case disassembler.SourceFile:
		disassembly, err = generateSourceAssembly(target, o.GOOS, o.Arch)
		if err != nil {
			return "", err
		}
	}

	if outputPath != "" {
		absOutputPath, err := filepath.Abs(outputPath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path of output file: %w", err)
		}
		err = os.WriteFile(absOutputPath, []byte(disassembly), 0600)
		if err != nil {
			return "", fmt.Errorf("failed to write to output file: %w", err)
		}
		return fmt.Sprintf("disassembly written to %s", outputPath), nil
	}
	return disassembly, nil
}

func generateSourceAssembly(target string, goos, arch string) (string, error) {
	// Build the binary
	tempFile := filepath.Join(os.TempDir(), "temp_binary")
	defer func() {
		_ = os.Remove(tempFile)
	}()

	absPath, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	//nolint:gosec
	buildCmd := exec.Command("go", "build", "-o", tempFile, absPath)
	buildCmd.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", goos),
		fmt.Sprintf("GOARCH=%s", arch),
	)
	var output []byte
	if output, err = buildCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build binary: %w\nOutput:\n%s", err, string(output))
	}

	//nolint:gosec
	cmd := exec.Command("objdump", "-d", tempFile)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", goos),
		fmt.Sprintf("GOARCH=%s", arch),
	)

	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to generate source assembly: %w\nOutput:\n%s", err, string(output))
	}
	return string(output), nil
}

func generateBinaryDisassembly(target string, goos, arch string) (string, error) {
	// Run objdump on the binary
	//nolint:gosec
	objdumpCmd := exec.Command("objdump", "-d", target)
	objdumpCmd.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", goos),
		fmt.Sprintf("GOARCH=%s", arch),
	)
	output, err := objdumpCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to generate binary disassembly: %w\nOutput:\n%s", err, string(output))
	}

	return string(output), nil
}
