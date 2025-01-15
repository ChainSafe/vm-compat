package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Instruction struct {
	Address string
	Opcode  string
	Args    []string
}

func main() {
	mode := flag.String("mode", "source", "Mode to generate opcodes: 'source' or 'binary'")
	output := flag.String("output", "", "Output file for generated assembly (optional)")
	target := flag.String("target", "", "Path to Go file or package (required)")

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
		assembly, err = generateSourceAssembly(*target)
	case "binary":
		assembly, err = generateBinaryDisassembly(*target)
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

	instructions := parseAsmOutput(assembly)

	// Print parsed instructions
	for _, inst := range instructions {
		fmt.Printf("Address: %-10s Opcode: %-8s Args: %-30s",
			inst.Address,
			inst.Opcode,
			strings.Join(inst.Args, ", "))

		fmt.Println()
	}
}

func generateSourceAssembly(target string) (string, error) {
	cmd := exec.Command("go", "build", "-gcflags=-S", target)
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=mips",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to generate source assembly: %w\nOutput:\n%s", err, string(output))
	}
	return string(output), nil
}

func generateBinaryDisassembly(target string) (string, error) {
	// Build the binary
	tempFile := filepath.Join(os.TempDir(), "temp_binary")
	defer os.Remove(tempFile)

	buildCmd := exec.Command("go", "build", "-o", tempFile, target)
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build binary: %w\nOutput:\n%s", err, string(output))
	}

	// Run objdump on the binary
	objdumpCmd := exec.Command("go", "tool", "objdump", tempFile)
	output, err := objdumpCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to generate binary disassembly: %w\nOutput:\n%s", err, string(output))
	}

	return string(output), nil
}

func parseAsmOutput(input string) []Instruction {
	var instructions []Instruction
	scanner := bufio.NewScanner(strings.NewReader(input))

	asmLineRe := regexp.MustCompile(`^\s*(0x[0-9a-fA-F]+)\s+([0-9]+)\s+\((.+):(\d+)\)\s+(\S+)\s+(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" || strings.Contains(line, "TEXT") || strings.Contains(line, "PCDATA") || strings.Contains(line, "FUNCDATA") ||
			strings.Contains(line, "CALL") || strings.Contains(line, "gclocals") {
			continue
		}

		if matches := asmLineRe.FindStringSubmatch(line); len(matches) > 0 {
			instruction := Instruction{
				Address: matches[1],
				Opcode:  matches[5],
				Args:    parseArgs(matches[6]),
			}
			instructions = append(instructions, instruction)
		}
	}

	return instructions
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
