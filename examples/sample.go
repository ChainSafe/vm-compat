package main

import (
	"os"
)

func main() {
	os.Stdout.Write([]byte("Hello World!"))
}
