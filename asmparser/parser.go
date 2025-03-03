// Package asmparser provides interfaces and structures for parsing and analyzing assembly code.
package asmparser

// Parser defines an interface for parsing assembly code from a given file path.
type Parser interface {
	Parse(path string) (CallGraph, error)
}

// InstructionType represents different categories of MIPS instructions.
type InstructionType string

const (
	RType InstructionType = "R-Type"
	IType InstructionType = "I-Type"
	JType InstructionType = "J-Type"
)

// Instruction defines an interface for working with assembly instructions.
type Instruction interface {
	Type() InstructionType // Type returns the instruction type (R, I, or J).
	Address() string       // Address returns the instruction memory address.
	OpcodeHex() string     // OpcodeHex returns the opcode of the instruction in hex string.
	Funct() string         // Funct returns the function code (for R-Type instructions).
	Mnemonic() string      // Mnemonic returns the assembly mnemonic representation.
	IsSyscall() bool       // IsSyscall returns true if the instruction is a syscall.
	Line() int             // Line number of the instruction
}

// Segment defines an interface representing a block of assembly instructions.
type Segment interface {
	// Address returns the segment's starting memory address.
	Address() string
	// Label returns the segment's associated label, if any.
	Label() string
	// Instructions return the list of instructions in the segment.
	Instructions() []Instruction
}

// CallGraph defines an interface representing a call graph of segments.
type CallGraph interface {
	// Segments returns all segments in the call graph.
	Segments() []Segment
	// ParentsOf returns the parent segments of a given segment.
	ParentsOf(segment Segment) []Segment
	// RetrieveSyscallNum returns the number of the syscall from the instr
	RetrieveSyscallNum(segment Segment, instr Instruction) ([]*Syscall, error)
}

// Syscall holds syscall origin related details
type Syscall struct {
	Number      int
	Segment     Segment
	Instruction Instruction
}
