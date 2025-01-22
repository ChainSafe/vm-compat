package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/ChainSafe/vm-compat/analysis"
	"github.com/ChainSafe/vm-compat/disassembler"
	"github.com/ChainSafe/vm-compat/disassembler/manager"
	"github.com/ChainSafe/vm-compat/opcode"
	"github.com/ChainSafe/vm-compat/profile"
)

var (
	vmProfile             = flag.String("vm-profile", "", "vm profile config")
	analyzer              = flag.String("analyzer", "opcode", "analyzer to run. Options: opcode, syscall")
	mode                  = flag.String("mode", "binary", "mode to run. only required for mode `opcode`. Options: binary, source")
	disassemblyOutputPath = flag.String("disassembly-output-path", "", "output file path for opcode assembly code. optional. only required for mode `opcode`. only specify if you want to write assembly code to a file")
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
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return
	}

	profile, err := profile.LoadProfile(*vmProfile)
	if err != nil {
		log.Fatalf("Error loading profile: %v", err)
	}

	switch *analyzer {
	case "opcode":
		err = analyzeOpcode(profile, args...)
		if err != nil {
			panic(err)
		}
	case "syscall":
		err = analysis.AnalyseSyscalls(profile, args...)
		if err != nil {
			panic(err)
		}
	default:
		log.Fatalf("Invalid analyzer: %s", *analyzer)
	}
}

func analyzeOpcode(profile *profile.VMProfile, paths ...string) error {
	if len(paths) == 0 {
		return fmt.Errorf("no paths provided for opcode analysis")
	}

	dis, err := manager.NewDisassembler(disassembler.TypeObjdump, profile.GOOS, profile.GoArch)
	if err != nil {
		return err
	}

	if *disassemblyOutputPath == "" {
		// add a temporary path to write the disassembly output
		*disassemblyOutputPath = filepath.Join(os.TempDir(), "temp_assembly_ouput")
		defer os.Remove(*disassemblyOutputPath)
	}

	switch *mode {
	case "binary":
		_, err = dis.Disassemble(disassembler.SourceBinary, paths[0], *disassemblyOutputPath)
		if err != nil {
			return err
		}
	case "source":
		_, err = dis.Disassemble(disassembler.SourceFile, paths[0], *disassemblyOutputPath)
		if err != nil {
			return err
		}
	}

	return opcode.AnalyseOpcodes(profile, *disassemblyOutputPath)
}
