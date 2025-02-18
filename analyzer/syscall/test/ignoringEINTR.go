package main

import (
	"os"
)

func main() {
	os.Chown("", 1, 1)
}
