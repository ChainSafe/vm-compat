package mips

import (
	"os"
	"testing"

	"github.com/ChainSafe/vm-compat/asmparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
   11000:   df c1 00 10  ld at,16(s8)
   11004:   00 3d 08 2b  sltu at,at,sp
   11008:	00 00 00 0c 	syscall
   1100c:	0c 02 36 76 	jal	8d9d8 <runtime.read>
   11010:   00 00 00 00  nop
000000000008d9d8 <runtime.read>:
   8d9d8:	8f a4 00 08 	lw	a0,8(sp)
   8d9dc:	df a5 00 10 	ld	a1,16(sp)
   8d9e0:	8f a6 00 18 	lw	a2,24(sp)
   8d9e4:	64 02 13 88 	daddiu	v0,zero,5000
   8d9e8:	00 00 00 0c 	syscall
   8d9ec:	10 e0 00 02 	beqz	a3,8d9f8 <runtime.read+0x20>
   8d9f0:	00 00 00 0f 	sync
`
	if _, err = tempFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = tempFile.Close()
	}()

	parser := NewParser()
	graph, err := parser.Parse(tempFile.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	assert.Len(t, graph.Segments(), 2)

	// Order not preserved in a segment as it is stored in a map
	var segment1, segment2 asmparser.Segment
	for _, seg := range graph.Segments() {
		if seg.Address() == "0x11000" {
			segment1 = seg
		} else if seg.Address() == "0x8d9d8" {
			segment2 = seg
		}
	}

	assert.Equal(t, "internal/abi.Kind.String", segment1.Label())
	assert.Equal(t, "0x11000", segment1.Address())

	assert.Equal(t, "runtime.read", segment2.Label())
	assert.Equal(t, "0x8d9d8", segment2.Address())

	instrs := segment1.Instructions()
	assert.Len(t, instrs, 5)

	assert.Equal(t, "0x11000", instrs[0].Address())
	assert.Equal(t, "0x37", instrs[0].OpcodeHex())
	assert.False(t, instrs[0].IsSyscall())
	assert.Equal(t, "", instrs[0].Funct())
	assert.Equal(t, asmparser.IType, instrs[0].Type())
	assert.Equal(t, "ld", instrs[0].Mnemonic())

	assert.Equal(t, "0x11004", instrs[1].Address())
	assert.Equal(t, "0x0", instrs[1].OpcodeHex())
	assert.False(t, instrs[1].IsSyscall())
	assert.Equal(t, "0x2b", instrs[1].Funct())
	assert.Equal(t, asmparser.RType, instrs[1].Type())
	assert.Equal(t, "sltu", instrs[1].Mnemonic())

	assert.Equal(t, "0x11008", instrs[2].Address())
	assert.Equal(t, "0x0", instrs[2].OpcodeHex())
	assert.True(t, instrs[2].IsSyscall())
	assert.Equal(t, "0xc", instrs[2].Funct())
	assert.Equal(t, asmparser.RType, instrs[2].Type())
	assert.Equal(t, "syscall", instrs[2].Mnemonic())

	_, err = graph.RetrieveSyscallNum(segment1, instrs[2])
	require.Error(t, err)

	assert.Equal(t, "0x1100c", instrs[3].Address())
	assert.Equal(t, "0x3", instrs[3].OpcodeHex())
	assert.False(t, instrs[3].IsSyscall())
	assert.Equal(t, "", instrs[3].Funct())
	assert.Equal(t, asmparser.JType, instrs[3].Type())
	assert.Equal(t, "jal", instrs[3].Mnemonic())

	assert.Equal(t, "0x11010", instrs[4].Address())
	assert.Equal(t, "0x0", instrs[4].OpcodeHex())
	assert.False(t, instrs[4].IsSyscall())
	assert.Equal(t, "0x0", instrs[4].Funct())
	assert.Equal(t, asmparser.RType, instrs[4].Type())
	assert.Equal(t, "nop", instrs[4].Mnemonic())

	instrs = segment2.Instructions()
	assert.Len(t, instrs, 7)

	// skip firsts 3 as it is similar to already checked instructions
	assert.Equal(t, "0x8d9e4", instrs[3].Address())
	assert.Equal(t, "0x19", instrs[3].OpcodeHex())
	assert.False(t, instrs[3].IsSyscall())
	assert.Equal(t, "", instrs[3].Funct())
	assert.Equal(t, asmparser.IType, instrs[3].Type())
	assert.Equal(t, "daddiu", instrs[3].Mnemonic())

	syscallNums, err := graph.RetrieveSyscallNum(segment2, instrs[4])
	require.NoError(t, err)
	assert.Equal(t, 5000, syscallNums[0].Number)

	assert.Equal(t, "0x8d9ec", instrs[5].Address())
	assert.Equal(t, "0x4", instrs[5].OpcodeHex())
	assert.False(t, instrs[5].IsSyscall())
	assert.Equal(t, "", instrs[5].Funct())
	assert.Equal(t, asmparser.IType, instrs[5].Type())
	assert.Equal(t, "beqz", instrs[5].Mnemonic())

	assert.Equal(t, "0x8d9f0", instrs[6].Address())
	assert.Equal(t, "0x0", instrs[6].OpcodeHex())
	assert.False(t, instrs[6].IsSyscall())
	assert.Equal(t, "0xf", instrs[6].Funct())
	assert.Equal(t, asmparser.RType, instrs[6].Type())
	assert.Equal(t, "sync", instrs[6].Mnemonic())
}

func TestIndirectSyscall(t *testing.T) {
	tempFile, err := os.CreateTemp("", "sample.asm")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	content := `/sample: file format elf64-tradbigmips

Disassembly of section .text:

0000000000011000 <main>:
   937e8:	64 01 00 02 	daddiu	at,zero,2
   937ec:	ff a1 00 08 	sd	at,8(sp)
   937f0:	64 01 00 01 	daddiu	at,zero,1
   937f4:	ff a1 00 10 	sd	at,16(sp)
   937f8:	ff a1 00 18 	sd	at,24(sp)
   937fc:	ff a1 00 20 	sd	at,32(sp)
   93800:	ff a1 00 28 	sd	at,40(sp)
   93804:	ff a1 00 30 	sd	at,48(sp)
   93808:	ff a1 00 38 	sd	at,56(sp)
   9380c:	0c 00 48 e6 	jal	12398 <syscall.RawSyscall6>
0000000000012398 <syscall.RawSyscall6>:
   12398:	ff bf ff a8 	sd	ra,-88(sp)
   1239c:	63 bd ff a8 	daddi	sp,sp,-88
   123a0:	ff bf 00 00 	sd	ra,0(sp)
   123a4:	df a1 00 60 	ld	at,96(sp)
   123a8:	ff a1 00 08 	sd	at,8(sp)
   123ac:	df a1 00 68 	ld	at,104(sp)
   123b0:	ff a1 00 10 	sd	at,16(sp)
   123b4:	df a1 00 70 	ld	at,112(sp)
   123b8:	ff a1 00 18 	sd	at,24(sp)
   123bc:	df a1 00 78 	ld	at,120(sp)
   123c0:	ff a1 00 20 	sd	at,32(sp)
   123c4:	df a1 00 80 	ld	at,128(sp)
   123c8:	ff a1 00 28 	sd	at,40(sp)
   123cc:	df a1 00 88 	ld	at,136(sp)
   123d0:	ff a1 00 30 	sd	at,48(sp)
   123d4:	df a1 00 90 	ld	at,144(sp)
   123d8:	ff a1 00 38 	sd	at,56(sp)
   123dc:	0c 00 49 04 	jal	12410 <runtime/internal/syscall.Syscall6>
0000000000012410 <runtime/internal/syscall.Syscall6>:
   12410:	df a2 00 08 	ld	v0,8(sp)
   12414:	df a4 00 10 	ld	a0,16(sp)
   12418:	df a5 00 18 	ld	a1,24(sp)
   1241c:	df a6 00 20 	ld	a2,32(sp)
   12420:	df a7 00 28 	ld	a3,40(sp)
   12424:	df a8 00 30 	ld	a4,48(sp)
   12428:	df a9 00 38 	ld	a5,56(sp)
   1242c:	00 00 18 25 	move	v1,zero
   12430:	00 00 00 0c 	syscall
`
	if _, err = tempFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = tempFile.Close()
	}()

	parser := NewParser()
	graph, err := parser.Parse(tempFile.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	var syscalls []*asmparser.Syscall
	for _, seg := range graph.Segments() {
		for _, instr := range seg.Instructions() {
			if instr.IsSyscall() {
				res, err := graph.RetrieveSyscallNum(seg, instr)
				assert.NoError(t, err)
				syscalls = append(syscalls, res...)
			}
		}
	}
	assert.Equal(t, 2, syscalls[0].Number)
}
