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

type InstructionSet string

const (
	R_INSTRUCTION  InstructionSet = "R"
	I_INSTRUCTION  InstructionSet = "I"
	J_INSTRUCTION  InstructionSet = "J"
	FR_INSTRUCTION InstructionSet = "FR"
	FI_INSTRUCTION InstructionSet = "FI"
)

var (
	// ref: https://en.wikibooks.org/wiki/MIPS_Assembly/Instruction_Formats#FI_Instructions
	memnomicToInstructionSet = map[string]InstructionSet{
		"add":   R_INSTRUCTION,
		"addi":  I_INSTRUCTION,
		"addiu": I_INSTRUCTION,
		"addu":  R_INSTRUCTION,
		"and":   R_INSTRUCTION,
		"andi":  I_INSTRUCTION,
		"beq":   I_INSTRUCTION,
		"blez":  I_INSTRUCTION,
		"bne":   I_INSTRUCTION,
		"bgtz":  I_INSTRUCTION,
		"div":   R_INSTRUCTION,
		"divu":  R_INSTRUCTION,
		"j":     J_INSTRUCTION,
		"jal":   J_INSTRUCTION,
		"jalr":  R_INSTRUCTION,
		"jr":    R_INSTRUCTION,
		"lb":    I_INSTRUCTION,
		"lbu":   I_INSTRUCTION,
		"lhu":   I_INSTRUCTION,
		"lui":   I_INSTRUCTION,
		"lw":    I_INSTRUCTION,
		"mfhi":  R_INSTRUCTION,
		"mthi":  R_INSTRUCTION,
		"mflo":  R_INSTRUCTION,
		"mtlo":  R_INSTRUCTION,
		"mfc0":  R_INSTRUCTION,
		"mult":  R_INSTRUCTION,
		"multu": R_INSTRUCTION,
		"nor":   R_INSTRUCTION,
		"xor":   R_INSTRUCTION,
		"or":    R_INSTRUCTION,
		"ori":   I_INSTRUCTION,
		"sb":    I_INSTRUCTION,
		"sh":    I_INSTRUCTION,
		"slt":   R_INSTRUCTION,
		"slti":  I_INSTRUCTION,
		"sltiu": I_INSTRUCTION,
		"sltu":  R_INSTRUCTION,
		"sll":   R_INSTRUCTION,
		"srl":   R_INSTRUCTION,
		"sra":   R_INSTRUCTION,
		"sub":   R_INSTRUCTION,
		"subu":  R_INSTRUCTION,
		"sw":    I_INSTRUCTION,
		"bnez":  I_INSTRUCTION,
	}
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
			iset, ok := memnomicToInstructionSet[matches[4]]
			if !ok {
				continue
			}
			hexNumber, err := parseOpcodeHex(matches[3], iset)
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

func parseOpcodeHex(str string, instructionSet InstructionSet) (string, error) {
	// parse the hex string to uint64
	i, err := strconv.ParseUint(str, 16, 32)
	if err != nil {
		return "", err
	}

	switch instructionSet {
	case R_INSTRUCTION, FR_INSTRUCTION:
		// R-type instructions have last 6 bits for the opcode, funct part
		return fmt.Sprintf("0x%x", i&0x3F), nil
	case I_INSTRUCTION, J_INSTRUCTION, FI_INSTRUCTION:
		// I-type instructions have first 6 bits for the opcode
		return fmt.Sprintf("0x%x", i>>0x1A), nil
	default:
		return "", fmt.Errorf("unknown instruction set: %s", instructionSet)
	}
}
