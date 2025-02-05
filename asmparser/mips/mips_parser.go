// Package mips provides the implementation of the asmparser interfaces for MIPS architecture.
package mips

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ChainSafe/vm-compat/asmparser"
)

// Constants defining MIPS register indexes.
const (
	registerZero = 0 // $zero register index in MIPS
	registerV0   = 2 // $v0 register index in MIPS
)

var (
	// Regular expressions for parsing assembly blocks and instructions.
	blockStartRegex  = regexp.MustCompile(`^([0-9a-fA-F]+)\s+<([^>]+)>:$`)
	instructionRegex = regexp.MustCompile(`^([0-9a-fA-F]+)(:)\s+([0-9a-fA-F]+)\s+([a-z]+)\s*(.*)`)
)

// parserImpl implements the asmparser.Parser interface.
type parserImpl struct{}

// NewParser returns a new instance of a MIPS assembly parser.
func NewParser() asmparser.Parser {
	return &parserImpl{}
}

// Parse reads and parses a MIPS assembly file into a CallGraph.
func (p *parserImpl) Parse(path string) (asmparser.CallGraph, error) {
	fpath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("error resolving absolute filepath: %w", err)
	}

	codefile, err := os.Open(fpath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
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
			return nil, fmt.Errorf("error parsing line: %w", err)
		}
		if instr == nil { // Ignore comments and empty lines
			continue
		}
		if instr.isSegmentStart() {
			currSegment = newSegment(instr.address, instr.label)
			graph.addSegment(currSegment)
		} else {
			if currSegment == nil {
				return nil, fmt.Errorf("invalid assembly: instruction encountered before segment definition")
			}
			currSegment.instructions = append(currSegment.instructions, instr)
			if instr.isJump() {
				graph.addParent(uint64(instr.jumpTarget()), currSegment.address)
			}
		}
	}
	return graph, nil
}

// parseLine attempts to parse a line of MIPS assembly.
func (p *parserImpl) parseLine(line string) (*instruction, error) {
	line = strings.TrimSpace(line)
	switch {
	case blockStartRegex.MatchString(line):
		return parseSegmentStart(line)
	case instructionRegex.MatchString(line):
		return parseInstruction(line)
	default:
		return nil, nil // Ignore comments and unrecognized lines
	}
}

// parseSegmentStart extracts segment information from a line.
func parseSegmentStart(line string) (*instruction, error) {
	matches := blockStartRegex.FindStringSubmatch(line)
	if len(matches) != 3 {
		return nil, fmt.Errorf("failed to parse segment start: %s", line)
	}
	pcAddress, err := strconv.ParseUint(matches[1], 16, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid segment address: %w", err)
	}
	return &instruction{address: pcAddress, label: matches[2]}, nil
}

// parseInstruction extracts instruction information from a line.
func parseInstruction(line string) (*instruction, error) {
	matches := instructionRegex.FindStringSubmatch(line)
	if len(matches) <= 4 {
		return nil, fmt.Errorf("failed to parse instruction: %s", line)
	}
	instr, err := decodeInstruction(matches[3])
	if err != nil {
		return nil, fmt.Errorf("invalid MIPS instruction format: %w", err)
	}
	pcAddress, err := strconv.ParseUint(matches[1], 16, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid instruction address: %w", err)
	}
	instr.opcodeString = matches[4]
	instr.address = pcAddress
	return instr, nil
}

// decodeInstruction decodes a hexadecimal MIPS instruction.
// https://en.wikibooks.org/wiki/MIPS_Assembly/Instruction_Formats#FI_Instructions
func decodeInstruction(str string) (*instruction, error) {
	_instruction, err := strconv.ParseUint(str, 16, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hex instruction: %w", err)
	}
	instr := uint32(_instruction)
	opcode := (instr >> 26) & 0x3F

	decodedInstruction := &instruction{
		opcode:   opcode,
		operands: make([]uint32, 0),
	}

	switch opcode {
	case 0x00, 0x1c: // R-Type Instructions (0x1c SPECIAL2 Format)
		rs := (instr >> 21) & 0x1F
		rt := (instr >> 16) & 0x1F
		rd := (instr >> 11) & 0x1F
		shamt := (instr >> 6) & 0x1F
		funcCode := instr & 0x3F
		decodedInstruction.instType = asmparser.RType
		decodedInstruction.operands = append(decodedInstruction.operands,
			rs,
			rt,
			rd,
			shamt,
			funcCode,
		)
	case 0x02, 0x03: // J-Type Instructions (Jump)
		targetAddress := (instr & 0x03FFFFFF) << 2
		decodedInstruction.instType = asmparser.JType
		decodedInstruction.operands = append(decodedInstruction.operands, targetAddress)
	default: // I-Type Instructions (e.g., daddi)
		rs := (instr >> 21) & 0x1F
		rt := (instr >> 16) & 0x1F
		immediate := instr & 0xFFFF
		decodedInstruction.instType = asmparser.IType
		decodedInstruction.operands = append(decodedInstruction.operands, rs, rt, immediate)
	}
	return decodedInstruction, nil
}

// instruction represents a MIPS instruction implementing the asmparser.Instruction interface.
type instruction struct {
	instType     asmparser.InstructionType
	opcodeString string
	address      uint64
	label        string // Used if this instruction marks the start of a segment
	opcode       uint32
	operands     []uint32 // RS, RT, RD, Shamt, FunctionCode, Immediate, TargetAddress
}

// isSegmentStart checks if the instruction marks the beginning of a segment.
func (i *instruction) isSegmentStart() bool {
	return len(i.label) > 0
}

func (i *instruction) Type() asmparser.InstructionType {
	return i.instType
}

func (i *instruction) OpcodeHex() string {
	return fmt.Sprintf("0x%x", i.opcode)
}

func (i *instruction) Funct() string {
	if i.instType == asmparser.RType && len(i.operands) > 4 {
		return fmt.Sprintf("0x%x", i.operands[4])
	}
	return ""
}

func (i *instruction) Mnemonic() string {
	return i.opcodeString
}

func (i *instruction) Address() string {
	return fmt.Sprintf("0x%x", i.address)
}

func (i *instruction) IsSyscall() bool {
	return strings.EqualFold(i.opcodeString, "syscall")
}

// isJump checks if the instruction is a jump instruction.
func (i *instruction) isJump() bool {
	return i.opcode == 0x02 || i.opcode == 0x03
}

// jumpTarget returns the jump target address of a jump instruction.
func (i *instruction) jumpTarget() uint32 {
	return i.operands[0]
}

// segment represents a block of assembly instructions implementing the asmparser.Segment interface.
type segment struct {
	address      uint64
	label        string
	instructions []*instruction
	parents      map[uint64]bool // Map of parent segment addresses to prevent duplicates.
}

// newSegment initializes a new segment with the given address and label.
func newSegment(address uint64, label string) *segment {
	return &segment{
		address:      address,
		label:        label,
		instructions: make([]*instruction, 0),
		parents:      make(map[uint64]bool),
	}
}

func (s *segment) Address() string {
	return fmt.Sprintf("0x%x", s.address)
}

func (s *segment) Label() string {
	return s.label
}

func (s *segment) Instructions() []asmparser.Instruction {
	instrs := make([]asmparser.Instruction, len(s.instructions))
	for i, ins := range s.instructions {
		instrs[i] = ins
	}
	return instrs
}

// RetrieveSyscallNum extracts the syscall number by analyzing the preceding instructions.
// Limitations:
// - Only supports `daddui` and `addui` instructions for loading syscall numbers.
// - Assumes that `v0` is set by an immediate operation and does not track register dependencies.
// - Does not handle indirect loading methods or data-dependent values.
func (s *segment) RetrieveSyscallNum(instr asmparser.Instruction) (int, error) {
	ins, ok := instr.(*instruction)
	if !ok {
		return 0, fmt.Errorf("invalid instruction type: expected MIPS instruction, got %T", instr)
	}
	offset := ins.address - s.address
	indexOfInstr := offset / uint64(4)

	for i := indexOfInstr - 1; i >= 0; i-- {
		currInstr := s.instructions[i]
		if currInstr.instType == asmparser.RType && len(currInstr.operands) > 2 && currInstr.operands[2] == registerV0 {
			return 0, fmt.Errorf("unsupported operation: register v0 modified before syscall assignment at %s",
				currInstr.Address())
		}
		if currInstr.instType == asmparser.IType && len(currInstr.operands) > 2 && currInstr.operands[1] == registerV0 {
			if currInstr.opcode == 0x19 || currInstr.opcode == 0x09 { // daddui or addui
				if currInstr.operands[0] != registerZero {
					return 0, fmt.Errorf("unsupported operation: syscall number must be loaded from $zero at address %s",
						currInstr.Address())
				}
				return int(currInstr.operands[2]), nil
			}
		}
	}

	return 0, fmt.Errorf("failed to retrieve syscall number: no valid assignment to register $v0 found in segment")
}

// callGraph represents a graph structure implementing asmparser.CallGraph.
type callGraph struct {
	segments map[uint64]*segment
}

// newCallGraph initializes an empty call graph.
func newCallGraph() *callGraph {
	return &callGraph{segments: make(map[uint64]*segment)}
}

func (g *callGraph) Segments() []asmparser.Segment {
	segments := make([]asmparser.Segment, 0, len(g.segments))
	for _, seg := range g.segments {
		segments = append(segments, seg)
	}
	return segments
}

func (g *callGraph) ParentsOf(seg asmparser.Segment) []asmparser.Segment {
	if segObj, ok := seg.(*segment); ok {
		parents := make([]asmparser.Segment, 0, len(segObj.parents))
		for addr := range segObj.parents {
			parents = append(parents, g.segments[addr])
		}
		return parents
	}
	return nil
}

func (g *callGraph) addParent(segmentAddr uint64, parentAddr uint64) {
	seg, exists := g.segments[segmentAddr]
	if !exists {
		seg = &segment{address: segmentAddr, instructions: make([]*instruction, 0), parents: make(map[uint64]bool)}
	}
	seg.parents[parentAddr] = true
	g.segments[segmentAddr] = seg
}

func (g *callGraph) addSegment(seg *segment) {
	if existingSeg, exists := g.segments[seg.address]; exists {
		seg.parents = existingSeg.parents
	}
	g.segments[seg.address] = seg
}
