package mips

import (
	"bufio"
	"fmt"
	"github.com/ChainSafe/vm-compat/asmparser"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	// TODO: the regex is currently according to objdump tool we are using. This should be updated according to the tool used.
	blockStartRegex  = regexp.MustCompile("^([0-9a-fA-F]+)\\s+<([^>]+)>:$")
	instructionRegex = regexp.MustCompile("^([0-9a-fA-F]+)(:)\\s+([0-9a-fA-F]+)\\s*([a-z]*)\\s*(.*)")
)

type ParserImpl struct {
}

func NewParser() asmparser.Parser {
	return &ParserImpl{}
}

func (p *ParserImpl) Parse(path string) (asmparser.CallGraph, error) {
	// return the absolute path of the given path
	fpath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("error getting the absolute filepath: %s: %w", path, err)
	}

	codefile, err := os.Open(fpath)
	if err != nil {
		return nil, fmt.Errorf("error opening filepath: %s: %w", fpath, err)
	}
	defer func() {
		_ = codefile.Close()
	}()

	var currSegment *segment
	graph := newCallGraph()
	scanner := bufio.NewScanner(codefile)
	for scanner.Scan() {
		line := scanner.Text()
		instr, err := p.parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("error parsing line: %s: %w", line, err)
		}
		if instr == nil { // comments and empty lines
			continue
		}
		if instr.isSegmentStart() {
			currSegment = newSegment(instr.address, instr.label)
			graph.addSegment(currSegment)
		} else {
			if currSegment == nil {
				return nil, fmt.Errorf("assembly code starts without a segment")
			}
			currSegment.instructions = append(currSegment.instructions, instr)
			if instr.isJump() {
				graph.addParent(uint64(instr.jumpTarget()), currSegment.address)
			}
		}
	}
	return graph, nil
}

func (p *ParserImpl) parseLine(line string) (*instruction, error) {
	line = strings.TrimSpace(line)
	switch {
	case blockStartRegex.MatchString(line):
		return parseSegmentStart(line)
	case instructionRegex.MatchString(line):
		return parseInstruction(line)
	default: // comments and others
		return nil, nil
	}
}

func parseSegmentStart(line string) (*instruction, error) {
	if matches := blockStartRegex.FindStringSubmatch(line); len(matches) == 3 {
		pcAddress, err := strconv.ParseUint(matches[1], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PC address for segment start: %w", err)
		}
		return &instruction{
			address: pcAddress,
			label:   matches[2],
		}, nil
	} else {
		return nil, fmt.Errorf("failed to parse segment start: %s", line)
	}
}

func parseInstruction(line string) (*instruction, error) {
	if matches := instructionRegex.FindStringSubmatch(line); len(matches) > 4 {
		instr, err := decodeInstruction(matches[3])
		if err != nil {
			return nil, fmt.Errorf("failed to decode mips instruction from hex: %w", err)
		}
		pcAddress, err := strconv.ParseUint(line, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PC address for instruction: %w", err)
		}
		instr.opcodeString = matches[4]
		instr.address = pcAddress
		return instr, nil
	} else {
		return nil, fmt.Errorf("failed to parse instruction: %s", line)
	}
}

//    6      5     5     5     5      6 bits
// [  op  |  rs |  rt |  rd |shamt| funct]  R-type
// [  op  |  rs |  rt | address/immediate]  I-type
// [  op  |        target address        ]  J-type
// parse the hex string to uint64
// https://en.wikipedia.org/wiki/Machine_code
// https://en.wikibooks.org/wiki/MIPS_Assembly/Instruction_Formats#FI_Instructions
func decodeInstruction(str string) (*instruction, error) {
	_instruction, err := strconv.ParseUint(str, 16, 32)
	if err != nil {
		return nil, err
	}
	// Convert to uint32 for correct bitwise operations
	instr := uint32(_instruction)

	// Extract opcode (first 6 bits)
	opcode := (instr >> 26) & 0x3F

	decodedInstruction := &instruction{
		opcode:   opcode,
		operands: make([]int64, 0),
	}

	switch opcode {
	case 0x00: // R-Type Instructions
		rs := (instr >> 21) & 0x1F
		rt := (instr >> 16) & 0x1F
		rd := (instr >> 11) & 0x1F
		shamt := (instr >> 6) & 0x1F
		funcCode := instr & 0x3F

		decodedInstruction._type = asmparser.InstructionTypeR
		decodedInstruction.operands = append(
			decodedInstruction.operands,
			int64(rs),
			int64(rt),
			int64(rd),
			int64(shamt),
			int64(funcCode))
	case 0x02, 0x03: // J-Type Instructions (Jump)
		targetAddress := (instr & 0x03FFFFFF) << 2

		decodedInstruction._type = asmparser.InstructionTypeJ
		decodedInstruction.operands = append(decodedInstruction.operands, int64(targetAddress))
	default: // I-Type Instructions (e.g., daddi)
		rs := (instr >> 21) & 0x1F
		rt := (instr >> 16) & 0x1F
		immediate := int16(instr & 0xFFFF) // Sign-extend

		decodedInstruction._type = asmparser.InstructionTypeI
		decodedInstruction.operands = append(decodedInstruction.operands, int64(rs), int64(rt), int64(immediate))
	}
	return decodedInstruction, nil
}
