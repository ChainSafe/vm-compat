package asmparser

// Parser holds interface for parsing assembly code
type Parser interface {
	Parse(path string) (CallGraph, error)
}

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
	Funct() string
	Mnemonic() string
	IsSyscall() bool
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
