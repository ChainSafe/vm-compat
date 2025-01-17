package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Instruction struct {
	Address   string
	Opcode    string
	OpcodeHex string
	Args      []string
}

func main() {
	mode := flag.String("mode", "source", "Mode to generate opcodes: 'source' or 'binary'")
	output := flag.String("output", "", "Output file for generated assembly (optional)")
	target := flag.String("target", "", "Path to Go file or package (required)")
	goos := flag.String("goos", "linux", "GOOS for the target binary. default: linux")
	arch := flag.String("arch", "mips", "GOARCH for the target binary. default: mips")

	flag.Parse()

	if *target == "" {
		fmt.Println("Error: Target file or package is required.")
		flag.Usage()
		os.Exit(1)
	}
	var assembly string
	var err error
	switch *mode {
	case "source":
		assembly, err = generateSourceAssembly(*target, *goos, *arch)
	case "binary":
		assembly, err = generateBinaryDisassembly(*target, *goos, *arch)
	default:
		fmt.Println("Error: Invalid mode. Use 'source' or 'binary'.")
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Error generating assembly: %v\n", err)
		os.Exit(1)
	}

	if *output != "" {
		err = os.WriteFile(*output, []byte(assembly), 0644)
		if err != nil {
			fmt.Printf("Error writing to output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Assembly written to %s\n", *output)
	} else {
		fmt.Println(assembly)
	}

	instructions, err := parseAsmOutput(assembly)
	if err != nil {
		fmt.Printf("Error parsing assembly: %v\n", err)
		os.Exit(1)
	}

	// Print parsed instructions
	for _, inst := range instructions {
		fmt.Printf("Address: %-10s Opcode: %-8s OpcodeHex: %s Args: %-30s",
			inst.Address,
			inst.Opcode,
			inst.OpcodeHex,
			strings.Join(inst.Args, ", "))

		fmt.Println()
	}
}

func generateSourceAssembly(target string, goos, arch string) (string, error) {
	// Build the binary
	tempFile := filepath.Join(os.TempDir(), "temp_binary")
	defer os.Remove(tempFile)

	fmt.Println("tempfle--", tempFile)
	buildCmd := exec.Command("go", "build", "-o", tempFile, target)
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

func parseAsmOutput(input string) ([]Instruction, error) {
	var instructions []Instruction
	scanner := bufio.NewScanner(strings.NewReader(input))

	asmLineRe := regexp.MustCompile(`^\s*([0-9a-fA-F]+)(:)\s+([0-9a-fA-F]+)\s+([a-z]+)\s+(.*)`)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if matches := asmLineRe.FindStringSubmatch(line); len(matches) > 0 {
			hexNumber, err := parseOpcodeHex(matches[3])
			if err != nil {
				fmt.Println("line: ", line)
				fmt.Println("matches: ", matches)
				return nil, err
			}

			instruction := Instruction{
				Address:   matches[1],
				Opcode:    matches[4],
				OpcodeHex: hexNumber,
				Args:      parseArgs(matches[5]),
			}
			instructions = append(instructions, instruction)
		}
	}

	return instructions, nil
}

func parseArgs(argsStr string) []string {
	args := []string{}
	current := ""
	inParens := false

	for _, char := range argsStr {
		switch char {
		case '(':
			inParens = true
			current += string(char)
		case ')':
			inParens = false
			current += string(char)
		case ',':
			if !inParens {
				if current != "" {
					args = append(args, strings.TrimSpace(current))
					current = ""
				}
			} else {
				current += string(char)
			}
		default:
			current += string(char)
		}
	}

	if current != "" {
		args = append(args, strings.TrimSpace(current))
	}

	return args
}

func parseOpcodeHex(str string) (string, error) {
	// parse the hex string to uint64
	i, err := strconv.ParseUint(str, 16, 32)
	if err != nil {
		return "", err
	}

	fixSixBits := i >> 0x1A

	if fixSixBits == 0 { // R Instruction
		return fmt.Sprintf("0x%x", i&0x3F), nil // return last 6 bits
	}

	return fmt.Sprintf("0x%x", i>>0x1A), nil // return first 6 bits
}
