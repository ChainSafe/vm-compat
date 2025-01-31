package common

type Instruction struct {
	Address        string
	InstructionHex string
	Opcode         uint64
	Funct          uint64
	Args           []string
}
