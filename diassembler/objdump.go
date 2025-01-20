package diassembler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func GenerateBinaryDisassembly(target string, outputPath, goos, arch string) (string, error) {
	disassembly, err := generateBinaryDisassembly(target, goos, arch)
	if err != nil {
		return "", err
	}

	if outputPath != "" {
		absOutputPath, err := filepath.Abs(outputPath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path of output file: %w", err)
		}
		err = os.WriteFile(absOutputPath, []byte(disassembly), 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write to output file: %w", err)
		}
		return fmt.Sprintf("disassembly written to %s", outputPath), nil
	}
	return disassembly, nil
}

func GenerateSourceAssembly(target string, outputPath string, goos, arch string) (string, error) {
	assembly, err := generateSourceAssembly(target, goos, arch)
	if err != nil {
		return "", err
	}

	if outputPath != "" {
		absOutputPath, err := filepath.Abs(outputPath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path of output file: %w", err)
		}
		err = os.WriteFile(absOutputPath, []byte(assembly), 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write to output file: %w", err)
		}
		return fmt.Sprintf("assembly written to %s", outputPath), nil
	}
	return assembly, nil
}

func generateSourceAssembly(target string, goos, arch string) (string, error) {
	// Build the binary
	tempFile := filepath.Join(os.TempDir(), "temp_binary")
	defer os.Remove(tempFile)

	absPath, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	buildCmd := exec.Command("go", "build", "-o", tempFile, absPath)
	buildCmd.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", goos),
		fmt.Sprintf("GOARCH=%s", arch),
	)
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build binary: %w\nOutput:\n%s", err, string(output))
	}

	cmd := exec.Command("objdump", "-d", tempFile)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", goos),
		fmt.Sprintf("GOARCH=%s", arch),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to generate source assembly: %w\nOutput:\n%s", err, string(output))
	}
	return string(output), nil
}

func generateBinaryDisassembly(target string, goos, arch string) (string, error) {
	// Run objdump on the binary
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
