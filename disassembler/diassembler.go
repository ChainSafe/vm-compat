package disassembler

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
