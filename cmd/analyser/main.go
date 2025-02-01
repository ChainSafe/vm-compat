package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/ChainSafe/vm-compat/analyser"
	"github.com/ChainSafe/vm-compat/disassembler"
	"github.com/ChainSafe/vm-compat/disassembler/manager"
	"github.com/ChainSafe/vm-compat/opcode"
	"github.com/ChainSafe/vm-compat/profile"
	"github.com/ChainSafe/vm-compat/renderer"
	"github.com/ChainSafe/vm-compat/syscall"
)

var (
	vmProfile = flag.String("vm-profile", "", "vm profile config")
	analyzer  = flag.String("analyzer", "opcode", "analyzer to run. Options: opcode, syscall")
	mode      = flag.String(
		"mode",
		"binary",
		"mode to run. only required for mode `opcode`. Options: binary, source")
	disassemblyOutputPath = flag.String(
		"disassembly-output-path",
		"",
		"output file path for opcode assembly code. optional. only required for mode `opcode`. only specify if you want to write assembly code to a file")
	format = flag.String("format", "text", "format of the output. Options: json, text")
)

const usage = `
analyser: checks the program compatibility against the vm profile

Usage:

  callgraph [-analyzer=opcode|syscall] [-vm-profile=path_to_config] package...
`

type WebData struct {
	Packages  string
	GraphJSON template.JS
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprint(os.Stderr, usage)
		return
	}

	profile, err := profile.LoadProfile(*vmProfile)
	if err != nil {
		log.Fatalf("Error loading profile: %v", err)
	}

	var issues []analyser.Issue
	switch *analyzer {
	case "opcode":
		issues, err = analyzeOpcode(profile, args[0])
		if err != nil {
			log.Fatalf("Unable to analyze Opcode: %s", err)
		}
	case "syscall":
		issues, err = syscall.AnalyseSyscalls(profile, args[0])
		if err != nil {
			log.Fatalf("Unable to analyze Syscalls: %s", err)
		}
	default:
		log.Fatalf("Invalid analyzer: %s", *analyzer)
	}

	switch *format {
	case "text":
		err = renderer.NewTextRenderer().Render(issues, os.Stdout)
		if err != nil {
			log.Fatalf("Unable to render: %s", err)
		}
	case "json":
		err = renderer.NewJSONRenderer().Render(issues, os.Stdout)
		if err != nil {
			log.Fatalf("Unable to render: %s", err)
		}
	default:
		log.Fatalf("Invalid format: %s", *format)
	}
}

func analyzeOpcode(profile *profile.VMProfile, paths string) ([]analyser.Issue, error) {
	dis, err := manager.NewDisassembler(disassembler.TypeObjdump, profile.GOOS, profile.GOARCH)
	if err != nil {
		return nil, err
	}

	if *disassemblyOutputPath == "" {
		// add a temporary path to write the disassembly output
		*disassemblyOutputPath = filepath.Join(os.TempDir(), "temp_assembly_output")
		defer os.Remove(*disassemblyOutputPath)
	}

	switch *mode {
	case "binary":
		_, err = dis.Disassemble(disassembler.SourceBinary, paths, *disassemblyOutputPath)
		if err != nil {
			return nil, err
		}
	case "source":
		_, err = dis.Disassemble(disassembler.SourceFile, paths, *disassemblyOutputPath)
		if err != nil {
			return nil, err
		}
	}

	return opcode.AnalyseOpcodes(profile, *disassemblyOutputPath)
}
