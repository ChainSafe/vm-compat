package opcode

type Arch int64

const (
	ArchMIPS32Bit Arch = iota + 1
	ArchMIPS64Bit
)

type Instruction struct {
	Address        string
	InstructionHex string
	Opcode         uint64
	Args           []string
}

type Provider interface {
	ParseOpcode(line string) (*Instruction, error)
	IsAllowedOpcode(code uint64) bool
}
