package asmparser

// Parser holds interface for parsing assembly code
type Parser interface {
	Parse(path string) (CallGraph, error)
}

// LineType represents the type of parsed line
type LineType int

const (
	LineTypeSegmentStart LineType = iota // Represents a function/block start
	LineTypeInstruction                  // Represents an assembly instruction
)

// InstructionType defines MIPS instruction categories
type InstructionType string

const (
	InstructionTypeR InstructionType = "R-Type"
	InstructionTypeI InstructionType = "I-Type"
	InstructionTypeJ InstructionType = "J-Type"
)

// Instruction holds required methods definition for implementing an instruction
type Instruction interface {
	Type() InstructionType
	Address() string
	Opcode() string
	Mnemonic() string
}

type Segment interface {
	Address() string
	Label() string
	Instructions() []Instruction
}

type CallGraph interface {
	Segments() []Segment
	ParentsOf(segment Segment) []Segment
}
