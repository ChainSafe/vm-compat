package mips

import (
	"testing"

	"github.com/ChainSafe/vm-compat/opcode/common"
	"github.com/ChainSafe/vm-compat/profile"
)

func TestMips(t *testing.T) {
	provider := NewProvider(common.ArchMIPS32Bit, &profile.VMProfile{
		VMName:         "canon",
		AllowedOpcodes: []string{"0X20", "0X21", "0X2A", "0X0A"},
	})

	instruction, err := provider.ParseAssembly("11000:\t8fc10008 \tlw\tat,8(s8)")
	if err != nil {
		t.Fatalf("failed to analyze opcodes: %v", err)
	}

	if instruction == nil {
		t.Fatalf("instruction is nil")
	}

	instruction, err = provider.ParseAssembly("1100c:\t00000000 \tnop")
	if err != nil {
		t.Fatalf("failed to analyze opcodes: %v", err)
	}

	if instruction == nil {
		t.Fatalf("instruction is nil")
	}
}
