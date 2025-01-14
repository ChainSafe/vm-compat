package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// Sample Go code
	a := 10
	b := 5
	fmt.Println(a + b) // Add operation (supported opcode)
	str := "hello"
	str = strings.ToUpper(str)

	// Incompatible syscall
	_, err := os.Open("file.txt")
	if err != nil {
		fmt.Println("Error:", err)
	}
}
