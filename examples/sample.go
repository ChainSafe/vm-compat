package main

import (
	"syscall"
)

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
}
