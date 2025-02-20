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
	registerZero = 0  // $zero register index in MIPS
	registerV0   = 2  // $v0 register index in MIPS
	registerSP   = 29 // $sp (Stack Pointer)
)

var (
	// Regular expressions for parsing assembly blocks and instructions.
	// It's only applicable for a file generated with llvm-objdump
	blockStartRegex  = regexp.MustCompile(`^([0-9a-fA-F]+)\s+<([^>]+)>:$`)
	instructionRegex = regexp.MustCompile(`^([0-9a-fA-F]+)(:)\s+((?:[0-9a-fA-F]{2}\s+){4})\s+([a-z]+(?:\.[a-z0-9]*)*)\s*(.*)`)
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
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		instr, err := p.parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("error parsing line: %w", err)
		}
		lineNum++
		if instr == nil { // Ignore comments and empty lines
			continue
		}
		instr.line = lineNum
		if instr.isSegmentStart() {
			currSegment = newSegment(instr.address, instr.label)
			graph.addSegment(currSegment)
		} else {
			if currSegment == nil {
				return nil, fmt.Errorf("invalid assembly: instruction encountered before segment definition")
			}
			currSegment.instructions = append(currSegment.instructions, instr)
			if instr.isJump() {
				//nolint
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
	instr, err := decodeInstruction(strings.ReplaceAll(matches[3], " ", ""))
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
		operands: make([]int64, 0),
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
			//nolint
			int64(rs),
			//nolint
			int64(rt),
			//nolint
			int64(rd),
			//nolint
			int64(shamt),
			//nolint
			int64(funcCode),
		)
	case 0x02, 0x03: // J-Type Instructions (Jump)
		targetAddress := (instr & 0x03FFFFFF) << 2
		decodedInstruction.instType = asmparser.JType
		//nolint
		decodedInstruction.operands = append(decodedInstruction.operands, int64(targetAddress))
	default: // I-Type Instructions (e.g., daddi)
		rs := (instr >> 21) & 0x1F
		rt := (instr >> 16) & 0x1F
		//nolint
		immediate := int16(instr & 0xFFFF)
		decodedInstruction.instType = asmparser.IType
		//nolint
		decodedInstruction.operands = append(decodedInstruction.operands, int64(rs), int64(rt), int64(immediate))
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
	operands     []int64 // RS, RT, RD, Shamt, FunctionCode, Immediate, TargetAddress
	line         int
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
func (i *instruction) jumpTarget() int64 {
	return i.operands[0]
}

func (i *instruction) Line() int {
	return i.line
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

// RetrieveSyscallNum extracts the syscall number by analyzing the preceding instructions.
// Limitation: If syscall number is dynamically generated, it cannot trace that
func (g *callGraph) RetrieveSyscallNum(seg asmparser.Segment, instr asmparser.Instruction) ([]*asmparser.Syscall, error) {
	ins, ok := instr.(*instruction)
	if !ok {
		return nil, fmt.Errorf("invalid instruction type: expected MIPS instruction, got %T", instr)
	}
	s, ok := seg.(*segment)
	if !ok {
		return nil, fmt.Errorf("invalid segment type: expected MIPS segment, got %T", seg)
	}
	var indexOfInstr int
	for i, _instr := range s.instructions {
		if _instr.Address() == ins.Address() {
			indexOfInstr = i
		}
	}
	var resolveRegisterValue func(register, offset int64, instrIdx int, seg, childSeg *segment) ([]*asmparser.Syscall, error)
	seen := make(map[*segment]bool)
	resolveRegisterValue = func(register, offset int64, instrIdx int, seg, childSeg *segment) ([]*asmparser.Syscall, error) {
		result := make([]*asmparser.Syscall, 0)
		// Special case, where we don't know from where to start
		// Need to find out instruction index
		if instrIdx == -2 {
			if seen[seg] {
				return result, nil
			}
			// multiple jump possible
			for i, inst := range seg.instructions {
				if inst.isJump() && uint64(inst.jumpTarget()) == childSeg.address { //nolint:gosec
					res, err := resolveRegisterValue(register, offset, i, seg, childSeg)
					if err != nil {
						return nil, err
					}
					result = append(result, res...)
				}
			}
			return result, nil
		}
		// When all the instruction has finished while processing,
		// Need to track back to it's caller
		if instrIdx == -1 {
			seen[seg] = true
			parents := g.ParentsOf(seg)
			if len(parents) == 0 {
				// Here, we cannot resolve any value for syscall, reasons can be it's being assigned in runtime.
				// Fine to ignore those syscall
				return result, nil
			}
			for _, sg := range parents {
				res, err := resolveRegisterValue(register, offset, -2, sg.(*segment), seg)
				if err != nil {
					return nil, err
				}
				result = append(result, res...)
			}
			return result, nil
		}

		currInstr := seg.instructions[instrIdx]
		switch currInstr.instType {
		case asmparser.RType:
			if len(currInstr.operands) > 2 {
				rd := currInstr.operands[2] // destination register
				// If the destination register is our target register,
				// we need to resolve the value for it
				if rd == register {
					return nil, fmt.Errorf("not handled modification of register in r-type instruction, instruction:%s", currInstr.Address())
				}
			}
		case asmparser.IType:
			if len(currInstr.operands) > 1 {
				rs := currInstr.operands[0]
				rt := currInstr.operands[1]
				if rs == register || rt == register {
					switch currInstr.opcode {
					case 0x23, 0x24, 0x27, 0x37: // load from rs to rt - rt matters
						if register == rt {
							// Load to sp - need to match offset
							if rt == registerSP && offset == currInstr.operands[2] {
								register = rs
							}
							// Load from SP - update offset
							if rs == registerSP {
								offset = currInstr.operands[2]
								register = rs
							}
						}
					case 0x2B, 0x3F, 0x28: // store to rs from rt - rs matters
						if register == rs {
							// Store to SP - need to  match the offset
							if rs == registerSP && offset == currInstr.operands[2] {
								register = rt
							}
							// Store from SP - need to update the offset
							if rt == registerSP {
								offset = currInstr.operands[2]
								register = rt
							}
						}
						return resolveRegisterValue(register, offset, instrIdx-1, seg, childSeg)
					case 0x08, 0x09, 0x18, 0x19: // add operations
						if register == rt {
							// need to check rs carefully
							// case 1- memory shift of sp(daddi sp, sp, -88)
							if rs == registerSP {
								offset += currInstr.operands[2]
								return resolveRegisterValue(register, offset, instrIdx-1, seg, childSeg)
							}
							// case 2- direct assigment to register where rs=registerZero
							if rs == registerZero {
								return []*asmparser.Syscall{{
									Number:      int(currInstr.operands[2]),
									Segment:     seg,
									Instruction: currInstr,
								}}, nil
							}
							return nil, fmt.Errorf("not handled modification of register in i-type instruction, instruction:%s", currInstr.Address())
						}
					default:
						return nil, fmt.Errorf("not handled opcode, instruction:%s", currInstr.Address())
					}
				}
			}
		default:
		}
		return resolveRegisterValue(register, offset, instrIdx-1, seg, childSeg)
	}

	result, err := resolveRegisterValue(registerV0, 0, indexOfInstr-1, s, nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}
