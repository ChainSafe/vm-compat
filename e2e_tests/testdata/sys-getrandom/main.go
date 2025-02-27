package main

import "syscall"

const (
	SYS_GETRANDOM = 318
)

func main() {
	_, _, err := syscall.Syscall(SYS_GETRANDOM, 0, 0, 0)
	if err != 0 {
		panic(err)
	}
}
