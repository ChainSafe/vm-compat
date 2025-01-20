package mips

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ChainSafe/vm-compat/opcode"
	"github.com/ChainSafe/vm-compat/profile"
)

const (
	mipsAsmRegex = "^\\s*([0-9a-fA-F]+)(:)\\s+([0-9a-fA-F]+)\\s*([a-z]*)\\s*(.*)"
)

type Provider struct {
	Arch    opcode.Arch
	profile *profile.VMProfile
}

func NewProvider(arch opcode.Arch, profile *profile.VMProfile) *Provider {
	return &Provider{Arch: arch, profile: profile}
}

func (p *Provider) ParseOpcode(line string) (*opcode.Instruction, error) {
	instructionDetected, err := parseASMLine(line)
	if err != nil {
		fmt.Printf("Error parsing line: %s: %v\n", line, err)
		return nil, err
	}
	return instructionDetected, nil
}

// IsAllowedOpcode checks if the given function is in the allowed opcodes.
func (p *Provider) IsAllowedOpcode(code uint64) bool {
	for _, op := range p.profile.AllowedOpcodes {
		i, err := strconv.ParseUint(op, 0, 64) // auto detect base
		if err != nil {
			fmt.Printf("Error parsing opcode hex string from vmprofile: %s: %v\n", op, err)
			return false
		}
		if i == code {
			return true
		}
	}
	return false
}

func parseASMLine(line string) (*opcode.Instruction, error) {
	asmLineRe := regexp.MustCompile(mipsAsmRegex)
	line = strings.TrimSpace(line)

	if matches := asmLineRe.FindStringSubmatch(line); len(matches) > 0 {
		hexNumber, err := parseOpcodeHex(matches[3])
		if err != nil {
			return nil, fmt.Errorf("failed to parse opcode hex: %w", err)
		}

		ins := &opcode.Instruction{
			Address:        matches[1],
			InstructionHex: matches[4],
			Opcode:         hexNumber,
		}

		if len(matches) > 5 {
			ins.Args = parseArgs(matches[5])
		}
		return ins, nil
	}

	return nil, nil
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

func parseOpcodeHex(str string) (uint64, error) {
	// parse the hex string to uint64
	i, err := strconv.ParseUint(str, 16, 32)
	if err != nil {
		return 0, err
	}

	fixSixBits := i >> 0x1A
	if fixSixBits == 0 { // R Instruction
		return i & 0x3F, nil // return last 6 bits
	}

	return i >> 0x1A, nil // return first 6 bits
}
