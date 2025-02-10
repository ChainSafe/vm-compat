# VM Compatibility Analyzer

This tool analyzes Go programs for compatibility with different Virtual Machines (VMs), 
specifically targeting the Cannon VM. It checks whether the opcodes, syscalls,
and other aspects of a Go program are supported by the chosen VM, and flags any compatibility issues.

## Overview

VM Compat is a CLI tool designed for checking compatibility of GO programs with different Virtual Machines (VMs),
specifically targeting the Cannon VM.
It takes a Go source file (typically `main.go`) as input, 
generates disassembled code, and parses each instruction to detect syscalls and opcodes. 
Additionally, it performs syscall analysis using SSA call graphs in Go.

### Features

- **Static Analysis**: Analyzes Go source code without executing it.
- **Disassembler Integration**: Converts Go code into low-level instructions.
- **Opcode and Syscall Detection**: Identifies all syscalls and opcodes used in the program.
- **SSA Call Graph Analysis**: Uses Go's SSA (Static Single Assignment) form to detect syscalls within function call graphs.
- **Compatibility Checking**: Helps determine whether a given Go program is incompatible with a targeted VM.

## How It Works

VM Compat performs static analysis to ensure compatibility with the target VM. Since it does not execute the code, 
it considers all possible execution paths, detecting any syscalls or
opcodes that might be present. This approach is beneficial because it ensures a
thorough analysis, even in cases where a particular execution path may never be taken at runtime.

For example, consider the following function:

```go
func demo() {
    if condition1 {
        doSyscall1()
    }
    if condition2 {
        doSyscall2()
    }
}
```

Even if `condition2` is never met during runtime, VM Compat will still detect `doSyscall2()` as a potential syscall. 
This ensures that all possible execution paths are analyzed,
making the tool effective in identifying compatibility concerns proactively.

## Installation

To install VM Compat, clone the repository and build the binary:

```sh
git clone https://github.com/your-repo/vm-compat.git
cd vm-compat
make analyser
```

## CLI Flags

VM Compat provides several command-line flags to control its behavior:

| Flag                       | Description                                                            | Default                           |
| -------------------------- | ---------------------------------------------------------------------- |-----------------------------------|
| `-vm-profile`              | Path to the VM profile config file.                                    | `./profile/cannon/cannon-64.yaml` |
| `-analyzer`                | Type of analysis to perform. Options: `opcode`, `syscall`              | analyzes both by default          |
| `-disassembly-output-path` | File path to store the disassembled opcode assembly code.              | None                              |
| `-format`                  | Output format. Options: `json`, `text`                                 | `text`                            |
| `-report-output-path`      | Path to store the analysis report. If not provided, outputs to stdout. | Stdout                            |

## Usage

Run VM Compat with the Go source file you want to analyze:

```sh
./bin/analyser -analyzer=opcode -format=text -disassembly-output-path=sample.asm -vm-profile=./profile/cannon/cannon-64.yaml ./examples/sample.go
```
To create vm specific profile, follow [this](./profile/readme.md)

## Example Output

```
==============================
🔍 Go Compatibility Analysis Report
==============================

🖥 VM Name: Cannon
⚙️ GOOS: linux
🛠 GOARCH: mips64
📅 Timestamp: 2025-02-10 19:40:17 UTC
🔢 Analyzer Version: 1.0.0

------------------------------
🚨 Summary of Issues
------------------------------
 ❗ Critical Issues: 65
⚠️ Warnings: 26
ℹ️ Total Issues: 91

------------------------------
📌 Detailed Issues
------------------------------

1. [CRITICAL] Incompatible Syscall Detected: 5006
   - Sources:
     ->  zsyscall_linux_mips64.go:1677 : (syscall.lstat)
      ->  syscall_linux_mips64x.go:154 : (syscall.Lstat)
...
```
