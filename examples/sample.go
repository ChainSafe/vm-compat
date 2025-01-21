package main

import (
	"syscall"
)

var ptr = syscall.SYS_OPENAT

func RawSyscall(trap uintptr) {
	syscall.RawSyscall6(trap, 0, 0, 0, 0, 0, 0)
}

func Syscall2(t uintptr) {
	RawSyscall(t)
}

func main() {
	var trap uintptr = syscall.SYS_READ
	if true {
		trap = getTrap()
	}
	Syscall2(trap)
}

func getTrap() uintptr {
	if true {
		return getTrap2()
	} else {
		return uintptr(ptr)
	}
}
func getTrap2() uintptr {
	return syscall.SYS_WRITE
}
