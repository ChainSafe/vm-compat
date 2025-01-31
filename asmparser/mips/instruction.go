package mips

import (
	"fmt"
	"github.com/ChainSafe/vm-compat/asmparser"
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

func (i instruction) isSegmentStart() bool {
	return len(i.label) > 0
}

func (i instruction) Type() asmparser.InstructionType {
	return i._type
}

func (i instruction) Opcode() string {
	return fmt.Sprintf("%x", i.opcode)
}

func (i instruction) Mnemonic() string {
	return i.opcodeString
}

func (i instruction) Address() string {
	return fmt.Sprintf("%x", i.address)
}

func (i instruction) isJump() bool {
	return i.opcode == 0x02 || i.opcode == 0x03
}

func (i instruction) jumpTarget() int64 {
	return i.operands[0]
}

type segment struct {
	address      uint64
	label        string
	instructions []*instruction
	parents      map[uint64]bool // Map of parent segments addresses. The Map is used to remove duplication.
	registers    []uint64        // store register values based on their index
}

func newSegment(address uint64, label string) *segment {
	return &segment{
		address:      address,
		label:        label,
		instructions: make([]*instruction, 0),
		parents:      make(map[uint64]bool),
		registers:    make([]uint64, 0),
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
			registers:    make([]uint64, 0),
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
