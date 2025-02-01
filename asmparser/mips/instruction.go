package mips

import (
	"fmt"
	"strings"

	"github.com/ChainSafe/vm-compat/asmparser"
)

var (
	registerZero = 0 // $zero register index in MIPS
	registerV0   = 2 // $v0 register index in MIPS
)

// instruction is struct, specific to mips instruction that implements asmparser.Instruction
type instruction struct {
	_type        asmparser.InstructionType
	opcodeString string
	address      uint64
	label        string // in case of segment start
	opcode       uint32
	operands     []int64 // RS, RT, RD, Shamt, FunctionCode, Immediate, TargetAddress
}

func (i *instruction) isSegmentStart() bool {
	return len(i.label) > 0
}

func (i *instruction) Type() asmparser.InstructionType {
	return i._type
}

func (i *instruction) Opcode() string {
	return fmt.Sprintf("%x", i.opcode)
}

func (i *instruction) Funct() string {
	if i._type == asmparser.InstructionTypeR {
		if len(i.operands) > 4 {
			return fmt.Sprintf("%x", i.operands[4])
		}
	}
	return ""
}

func (i *instruction) Mnemonic() string {
	return i.opcodeString
}

func (i *instruction) Address() string {
	return fmt.Sprintf("%x", i.address)
}

func (i *instruction) IsSyscall() bool {
	return strings.EqualFold(i.opcodeString, "syscall")
}

func (i *instruction) isJump() bool {
	return i.opcode == 0x02 || i.opcode == 0x03
}

func (i *instruction) jumpTarget() int64 {
	return i.operands[0]
}

type segment struct {
	address      uint64
	label        string
	instructions []*instruction
	parents      map[uint64]bool // Map of parent segments addresses. The Map is used to remove duplication.
}

func newSegment(address uint64, label string) *segment {
	return &segment{
		address:      address,
		label:        label,
		instructions: make([]*instruction, 0),
		parents:      make(map[uint64]bool),
	}
}

func (s *segment) Address() string {
	return fmt.Sprintf("%x", s.address)
}

func (s *segment) Label() string {
	return s.label
}

func (s *segment) Instructions() []asmparser.Instruction {
	instructions := make([]asmparser.Instruction, len(s.instructions))
	for i, ins := range s.instructions {
		instructions[i] = ins
	}
	return instructions
}

func (s *segment) RetrieveSyscallNum(_instr asmparser.Instruction) (int, error) {
	if instr, ok := _instr.(*instruction); ok {
		offset := instr.address - instr.address
		indexOfInstr := offset / 4
		// Need to check for register v0 value for syscall.

		// This is an incomplete function. which only supports daddui and addui with zero in rs register and an immediate value.
		for i := indexOfInstr - 1; i >= 0; i-- {
			currInstr := s.instructions[i]
			switch currInstr._type {
			case asmparser.InstructionTypeI: // check if target register(rt) is v0, and if it's the rs register is $zero
				if len(currInstr.operands) > 2 && currInstr.operands[1] == int64(registerV0) {
					if currInstr.opcode == 0x19 || currInstr.opcode == 0x09 { // daddui or, addui
						if currInstr.operands[0] != int64(registerZero) {
							return 0, fmt.Errorf("unsupported operation to register v0. loading with zero register not supported %s", currInstr.Address())
						}
						return int(currInstr.operands[2]), nil
					}
					return 0, fmt.Errorf("unsupported operation to register v0. opcode not supported %s", currInstr.Address())
				}

			case asmparser.InstructionTypeR: // check if destination register(rd) is v0, which is currently not supported
				if len(currInstr.operands) > 2 && currInstr.operands[2] == int64(registerV0) {
					return 0, fmt.Errorf("updating v0 register not supported yet to retrieve syscall %s", currInstr.Address())
				}
			default: // J type, not expected
				return 0, fmt.Errorf("invalid instruction type while retrieving syscall %s", currInstr.Address())
			}
		}
		return 0, fmt.Errorf("error tracking syscall value in the segment, value not found")
	}
	return 0, fmt.Errorf("invalid instruction type while retrieving syscall %s", _instr.Address())
}

type callGraph struct {
	segments map[uint64]*segment
}

func newCallGraph() *callGraph {
	return &callGraph{segments: make(map[uint64]*segment)}
}

func (g *callGraph) Segments() []asmparser.Segment {
	segments := make([]asmparser.Segment, 0)
	for _, seg := range g.segments {
		segments = append(segments, seg)
	}
	return segments
}

func (g *callGraph) ParentsOf(seg asmparser.Segment) []asmparser.Segment {
	segments := make([]asmparser.Segment, 0)
	if segObj, ok := seg.(*segment); ok {
		for address := range segObj.parents {
			segments = append(segments, g.segments[address])
		}
	}
	return segments
}

func (g *callGraph) addParent(segmentAddress uint64, parentAddress uint64) {
	seg, exist := g.segments[segmentAddress]
	if !exist {
		seg = &segment{
			address:      segmentAddress,
			instructions: make([]*instruction, 0),
			parents:      make(map[uint64]bool),
		}
	}
	seg.parents[parentAddress] = true
	g.segments[segmentAddress] = seg
}

func (g *callGraph) addSegment(seg *segment) {
	_seg, exist := g.segments[seg.address]
	if exist {
		seg.parents = _seg.parents
	}
	g.segments[seg.address] = seg
}
