package main

import "syscall"

const (
	SYS_CLOCK_GETTIME = 228
)

func main() {
	_, _, err := syscall.Syscall(SYS_CLOCK_GETTIME, 0, 0, 0)
	if err != 0 {
		panic(err)
	}
}
