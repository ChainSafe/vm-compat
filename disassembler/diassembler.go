package disassembler

import (
	"errors"

	"github.com/ChainSafe/vm-compat/disassembler/objdump"
)

type Source int64

const (
	SourceBinary Source = iota + 1
	SourceFile
)

type Disassembler interface {
	Disassemble(mode Source, target string, outputPath string) (string, error)
}

type Type int64

const (
	TypeObjdump Type = iota + 1
)

func NewDisassembler(typ Type, os, arch string) (Disassembler, error) {
	switch typ {
	case TypeObjdump:
		return objdump.New(os, arch), nil
	default:
		return nil, errors.New("disassembler not supported")
	}
}
