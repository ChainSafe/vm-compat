package main

import (
	"syscall"
)

<<<<<<< HEAD
func main() {
	lvl31(0)
	lvl32(0)
}

func lvl31(v0 int) {
	lvl21(v0)
	lvl22(v0)
}
func lvl32(v0 int) {
	lvl21(v0)
	lvl22(v0)
}
func lvl21(v0 int) {
	lvl1(v0)
}

func lvl22(v0 int) {
	lvl1(v0)
}

func lvl1(v0 int) {
	syscall.RawSyscall6(1, 0, 0, 0, 0, 0, 0)
=======
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
>>>>>>> 05d41c22f14455c84c6444320b683cca7e767f69
}
func getTrap2() uintptr {
	return syscall.SYS_WRITE
}
