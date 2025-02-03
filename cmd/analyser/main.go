package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/ChainSafe/vm-compat/analyser"
	"github.com/ChainSafe/vm-compat/analyser/opcode"
	"github.com/ChainSafe/vm-compat/analyser/syscall"
	"github.com/ChainSafe/vm-compat/disassembler"
	"github.com/ChainSafe/vm-compat/disassembler/manager"
	"github.com/ChainSafe/vm-compat/profile"
	"github.com/ChainSafe/vm-compat/renderer"
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
	format           = flag.String("format", "text", "format of the output. Options: json, text")
	reportOutputPath = flag.String("report-output-path", "", "output file path for report to pass. optional. default: stdout")
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
	*disassemblyOutputPath, err = disassemble(profile, args[0])
	if err != nil {
		log.Fatalf("Error disassembling the file: %v", err)
	}

	var issues []*analyser.Issue
	switch *analyzer {
	case "opcode":
		issues, err = opcode.NewAnalyser(profile).Analyze(*disassemblyOutputPath)
		if err != nil {
			log.Fatalf("Unable to analyze Opcode: %s", err)
		}
	case "syscall":
		issues, err = syscall.NewGOSyscallAnalyser(profile).Analyze(args[0])
		if err != nil {
			log.Fatalf("Unable to analyze Syscalls: %s", err)
		}
		issues2, err := syscall.NewAssemblySyscallAnalyser(profile).Analyze(*disassemblyOutputPath)
		if err != nil {
			log.Fatalf("Unable to analyze Syscalls: %s", err)
		}
		issues = append(issues, issues2...)
	default:
		log.Fatalf("Invalid analyzer: %s", *analyzer)
	}

	output := os.Stdout
	if *reportOutputPath != "" {
		path, err := filepath.Abs(*reportOutputPath)
		if err != nil {
			log.Fatalf("Unable to determine absolute path to output file: %s", err)
		}

		output, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			log.Fatalf("Unable to open output file: %s", err)
		}
		defer output.Close()
	}
	switch *format {
	case "text":
		err = renderer.NewTextRenderer().Render(issues, output)
		if err != nil {
			log.Fatalf("Unable to render: %s", err)
		}
	case "json":
		err = renderer.NewJSONRenderer().Render(issues, output)
		if err != nil {
			log.Fatalf("Unable to render: %s", err)
		}
	default:
		log.Fatalf("Invalid format: %s", *format)
	}
}

func disassemble(profile *profile.VMProfile, paths string) (string, error) {
	dis, err := manager.NewDisassembler(disassembler.TypeObjdump, profile.GOOS, profile.GOARCH)
	if err != nil {
		return "", err
	}

	if *disassemblyOutputPath == "" {
		// add a temporary path to write the disassembly output
		*disassemblyOutputPath = filepath.Join(os.TempDir(), "temp_assembly_output")
	}

	switch *mode {
	case "binary":
		_, err = dis.Disassemble(disassembler.SourceBinary, paths, *disassemblyOutputPath)
		if err != nil {
			return "", err
		}
	case "source":
		_, err = dis.Disassemble(disassembler.SourceFile, paths, *disassemblyOutputPath)
		if err != nil {
			return "", err
		}
	}
	return *disassemblyOutputPath, nil
}
