package manager

import (
	"errors"

	"github.com/ChainSafe/vm-compat/disassembler"
	"github.com/ChainSafe/vm-compat/disassembler/objdump"
)

func NewDisassembler(typ disassembler.Type, os, arch string) (disassembler.Disassembler, error) {
	switch typ {
	case disassembler.TypeObjdump:
		return objdump.New(os, arch), nil
	default:
		return nil, errors.New("disassembler not supported")
	}
}
