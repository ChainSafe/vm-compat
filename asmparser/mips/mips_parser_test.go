package mips

import (
	"os"
	"testing"

	"github.com/ChainSafe/vm-compat/asmparser"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tempFile, err := os.CreateTemp("", "sample.asm")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	content := `/sample: file format elf64-tradbigmips

Disassembly of section .text:

0000000000011000 <internal/abi.Kind.String>:
   11000:   dfc10010  ld at,16(s8)
   11004:   003d082b  sltu at,at,sp
   11008:	0000000c 	syscall
   1100c:	0c023676 	jal	8d9d8 <runtime.read>
   11010:   00000000  nop
000000000008d9d8 <runtime.read>:
   8d9d8:	8fa40008 	lw	a0,8(sp)
   8d9dc:	dfa50010 	ld	a1,16(sp)
   8d9e0:	8fa60018 	lw	a2,24(sp)
   8d9e4:	64021388 	daddiu	v0,zero,5000
   8d9e8:	0000000c 	syscall
   8d9ec:	10e00002 	beqz	a3,8d9f8 <runtime.read+0x20>
   8d9f0:	0000000f 	sync
`
	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tempFile.Close()

	parser := NewParser()
	graph, err := parser.Parse(tempFile.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	assert.Equal(t, len(graph.Segments()), 2)

	// Order not preserved in a segment as it is stored in a map
	var segment1, segment2 asmparser.Segment
	for _, seg := range graph.Segments() {
		if seg.Address() == "0x11000" {
			segment1 = seg
		} else if seg.Address() == "0x8d9d8" {
			segment2 = seg
		}
	}

	assert.Equal(t, segment1.Label(), "internal/abi.Kind.String")
	assert.Equal(t, segment1.Address(), "0x11000")

	assert.Equal(t, segment2.Label(), "runtime.read")
	assert.Equal(t, segment2.Address(), "0x8d9d8")

	instrs := segment1.Instructions()
	assert.Equal(t, len(instrs), 5)

	assert.Equal(t, instrs[0].Address(), "0x11000")
	assert.Equal(t, instrs[0].Opcode(), "0x37")
	assert.Equal(t, instrs[0].IsSyscall(), false)
	assert.Equal(t, instrs[0].Funct(), "")
	assert.Equal(t, instrs[0].Type(), asmparser.IType)
	assert.Equal(t, instrs[0].Mnemonic(), "ld")

	assert.Equal(t, instrs[1].Address(), "0x11004")
	assert.Equal(t, instrs[1].Opcode(), "0x0")
	assert.Equal(t, instrs[1].IsSyscall(), false)
	assert.Equal(t, instrs[1].Funct(), "0x2b")
	assert.Equal(t, instrs[1].Type(), asmparser.RType)
	assert.Equal(t, instrs[1].Mnemonic(), "sltu")

	assert.Equal(t, instrs[2].Address(), "0x11008")
	assert.Equal(t, instrs[2].Opcode(), "0x0")
	assert.Equal(t, instrs[2].IsSyscall(), true)
	assert.Equal(t, instrs[2].Funct(), "0xc")
	assert.Equal(t, instrs[2].Type(), asmparser.RType)
	assert.Equal(t, instrs[2].Mnemonic(), "syscall")

	_, err = segment1.RetrieveSyscallNum(instrs[2])
	assert.Error(t, err)

	assert.Equal(t, instrs[3].Address(), "0x1100c")
	assert.Equal(t, instrs[3].Opcode(), "0x3")
	assert.Equal(t, instrs[3].IsSyscall(), false)
	assert.Equal(t, instrs[3].Funct(), "")
	assert.Equal(t, instrs[3].Type(), asmparser.JType)
	assert.Equal(t, instrs[3].Mnemonic(), "jal")

	assert.Equal(t, instrs[4].Address(), "0x11010")
	assert.Equal(t, instrs[4].Opcode(), "0x0")
	assert.Equal(t, instrs[4].IsSyscall(), false)
	assert.Equal(t, instrs[4].Funct(), "0x0")
	assert.Equal(t, instrs[4].Type(), asmparser.RType)
	assert.Equal(t, instrs[4].Mnemonic(), "nop")

	instrs = segment2.Instructions()
	assert.Equal(t, len(instrs), 7)

	// skip firsts 3 as it is similar to already checked instructions
	assert.Equal(t, instrs[3].Address(), "0x8d9e4")
	assert.Equal(t, instrs[3].Opcode(), "0x19")
	assert.Equal(t, instrs[3].IsSyscall(), false)
	assert.Equal(t, instrs[3].Funct(), "")
	assert.Equal(t, instrs[3].Type(), asmparser.IType)
	assert.Equal(t, instrs[3].Mnemonic(), "daddiu")

	syscallNum, err := segment2.RetrieveSyscallNum(instrs[4])
	assert.NoError(t, err)
	assert.Equal(t, syscallNum, 5000)

	assert.Equal(t, instrs[5].Address(), "0x8d9ec")
	assert.Equal(t, instrs[5].Opcode(), "0x4")
	assert.Equal(t, instrs[5].IsSyscall(), false)
	assert.Equal(t, instrs[5].Funct(), "")
	assert.Equal(t, instrs[5].Type(), asmparser.IType)
	assert.Equal(t, instrs[5].Mnemonic(), "beqz")

	assert.Equal(t, instrs[6].Address(), "0x8d9f0")
	assert.Equal(t, instrs[6].Opcode(), "0x0")
	assert.Equal(t, instrs[6].IsSyscall(), false)
	assert.Equal(t, instrs[6].Funct(), "0xf")
	assert.Equal(t, instrs[6].Type(), asmparser.RType)
	assert.Equal(t, instrs[6].Mnemonic(), "sync")
}
