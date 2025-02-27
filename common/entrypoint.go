package common

import "strings"

func ProgramEntrypoint(arch string) func(function string) bool {
	switch arch {
	case "mips":
		return func(function string) bool {
			return function == "runtime.rt0_go" || // start point of a go program
				function == "main.main" || // main
				strings.Contains(function, ".init.") || // all init functions
				strings.HasSuffix(function, ".init") // vars
		}
	case "mips64":
		return func(function string) bool {
			return function == "runtime.rt0_go" || // start point of a go program
				function == "main.main" || // main
				strings.Contains(function, ".init.") || // all init functions
				strings.HasSuffix(function, ".init") // vars
		}
	}
	return func(function string) bool {
		return false
	}
}
