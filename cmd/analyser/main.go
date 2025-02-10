package main

import (
	"flag"
	"fmt"
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
	vmProfile         = flag.String("vm-profile", "./profile/cannon/cannon-64.yaml", "vm profile config")
	analyzer          = flag.String("analyzer", "syscall", "analyzer to run. Options: opcode, syscall")
	disassemblyOutput = flag.String("disassembly-output-path", "", "output file path for opcode assembly code.")
	format            = flag.String("format", "text", "format of the output. Options: json, text")
	reportOutputPath  = flag.String("report-output-path", "", "output file path for report. Default: stdout")
)

const usage = `
analyser: checks the program compatibility against the vm profile

Usage:

  callgraph [-analyzer=opcode|syscall] [-vm-profile=path_to_config] package...
`

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, usage)
		return
	}

	prof, err := profile.LoadProfile(*vmProfile)
	if err != nil {
		log.Fatalf("Error loading profile: %v", err)
	}

	disassemblyPath, err := disassemble(prof, args[0], *disassemblyOutput)
	if err != nil {
		log.Fatalf("Error disassembling the file: %v", err)
	}

	issues, err := analyze(prof, args[0], disassemblyPath, *analyzer)
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	if err := writeReport(issues, *format, *reportOutputPath, prof); err != nil {
		log.Fatalf("Unable to write report: %v", err)
	}
}

// disassemble extracts assembly output for analysis.
func disassemble(prof *profile.VMProfile, path, outputPath string) (string, error) {
	dis, err := manager.NewDisassembler(disassembler.TypeObjdump, prof.GOOS, prof.GOARCH)
	if err != nil {
		return "", err
	}

	if outputPath == "" {
		outputPath = filepath.Join(os.TempDir(), "temp_assembly_output")
	}

	_, err = dis.Disassemble(disassembler.SourceFile, path, outputPath)
	return outputPath, err
}

// analyze runs the selected analyzer(s).
func analyze(prof *profile.VMProfile, path, disassemblyPath, mode string) ([]*analyser.Issue, error) {
	if mode == "opcode" {
		return opcode.NewAnalyser(prof).Analyze(disassemblyPath)
	}
	if mode == "syscall" {
		return analyzeSyscalls(prof, path, disassemblyPath)
	}

	opIssues, err := opcode.NewAnalyser(prof).Analyze(disassemblyPath)
	if err != nil {
		return nil, err
	}
	sysIssues, err := analyzeSyscalls(prof, path, disassemblyPath)
	if err != nil {
		return nil, err
	}

	return append(opIssues, sysIssues...), nil
}

// writeReport outputs the results in the specified format.
func writeReport(issues []*analyser.Issue, format, outputPath string, prof *profile.VMProfile) error {
	var output *os.File
	if outputPath == "" {
		output = os.Stdout
	} else {
		absPath, err := filepath.Abs(outputPath)
		if err != nil {
			return fmt.Errorf("unable to determine absolute path: %w", err)
		}
		output, err = os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("unable to open output file: %w", err)
		}
		defer func() {
			_ = output.Close()
		}()
	}

	var rendererInstance renderer.Renderer
	switch format {
	case "text":
		rendererInstance = renderer.NewTextRenderer(prof)
	case "json":
		rendererInstance = renderer.NewJSONRenderer()
	default:
		return fmt.Errorf("invalid format: %s", format)
	}

	return rendererInstance.Render(issues, output)
}

func analyzeSyscalls(profile *profile.VMProfile, source string, disassemblyPath string) ([]*analyser.Issue, error) {
	issues, err := syscall.NewGOSyscallAnalyser(profile).Analyze(source)
	if err != nil {
		return nil, err
	}
	issues2, err := syscall.NewAssemblySyscallAnalyser(profile).Analyze(disassemblyPath)
	if err != nil {
		return nil, err
	}
	return append(issues, issues2...), nil
}
