package common

import "strings"

func ProgramEntrypoint(arch string) func(function string) bool {
	switch arch {
	case "mips":
		return func(function string) bool {
			// Ignoring rt0_go directly as it contains unreachable portion
			return function == "runtime.check" ||
				function == "runtime.args" ||
				function == "runtime.osinit" ||
				function == "runtime.schedinit" ||
				function == "runtime.newproc" ||
				function == "runtime.mstart" ||
				strings.Contains(function, "main.main") || // main and closures or anonymous functions
				strings.Contains(function, ".init.") || // all init functions
				strings.HasSuffix(function, ".init") || // vars
				// Bellow functions though are not the starting point, but those have a lot of trigger points
				// and for a sample go program is expected to be called. So, to reduce stack trace, it's fine to make them as entry points
				function == "runtime.gcStart" ||
				function == "runtime.mallocgc" ||
				function == "runtime.morestack" ||
				function == "runtime.systemstack" ||
				function == "runtime.gopanic" ||
				function == "runtime.chanrecv" ||
				function == "runtime.startm" || // 32 bit specific
				function == "runtime.sysAlloc" // // 32 bit specific
		}
	case "mips64":
		return func(function string) bool {
			return strings.Contains(function, "main.main") || // main and closures or anonymous functions
				strings.Contains(function, ".init.") || // all init functions
				strings.HasSuffix(function, ".init") || // vars
				function == "runtime.rt0_go" || // start point of any go program
				// Bellow functions though are not the starting point, but those have a lot of trigger points
				// and for a sample go program is expected to be called. So, to reduce stack trace, it's fine to make them as entry points
				function == "runtime.gcStart" ||
				function == "runtime.mallocgc" ||
				function == "runtime.morestack" ||
				function == "runtime.systemstack" ||
				function == "runtime.gopanic" ||
				function == "runtime.chanrecv"
		}
	}
	return func(function string) bool {
		return false
	}
}
