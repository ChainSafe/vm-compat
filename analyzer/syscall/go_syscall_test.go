package syscall_test

import (
	"github.com/ChainSafe/vm-compat/analyzer/syscall"
	"github.com/ChainSafe/vm-compat/profile"
	"testing"
)

func TestTrackStack(t *testing.T) {
	path := "/home/sadiq/chainsafe/vm-compat/analyzer/syscall/test/ignoringEINTR.go"
	analyzer := syscall.NewGOSyscallAnalyser(&profile.VMProfile{
		VMName: "test",
		GOOS:   "linux",
		GOARCH: "mips64",
	})

	_, err := analyzer.Analyze(path, true)
	if err != nil {
		t.Fatal(err)
	}
}
