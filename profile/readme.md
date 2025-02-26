# Creating VM Profiles

## Overview
A VM profile defines the execution environment for the analysis performed by VM Compat. It specifies the allowed opcodes and syscalls to determine program compatibility with a specific VM.

## Profile Fields
A VM profile consists of the following fields:

- `vm`: Name of the virtual machine (e.g., Cannon).
- `goos`: Target operating system (e.g., linux).
- `goarch`: Target architecture (e.g., mips64).
- `allowed_opcodes`: List of permitted opcodes with optional function values.
- `allowed_syscalls`: List of system calls allowed by the VM.
- `noop_syscalls`: List of system calls treated as no-ops by the VM.
- `ignored_functions`: List of functions or blocks disabled on the VM due to they might never be called in usual scenarios.
  Example:
    - 'syscall.setrlimit': Only executed in certain condition that doesn't meet with cannon, https://go.dev/src/syscall/rlimit.go
    - 'runtime.morestack': Should execute in case of stack overflow, but not in usual case.

## Getting Opcode and Syscall Information
Determining the correct opcodes and syscalls for a VM requires extensive research on the targeted VM
architecture and its official documentation.


