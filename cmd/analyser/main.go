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
	vmProfile         = flag.String("vm-profile", "", "vm profile config")
	analyzer          = flag.String("analyzer", "", "analyzer to run. Options: opcode, syscall")
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

	if err := writeReport(issues, *format, *reportOutputPath); err != nil {
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
	var issues []*analyser.Issue
	var err error

	switch mode {
	case "opcode":
		issues, err = opcode.NewAnalyser(prof).Analyze(disassemblyPath)
	case "syscall":
		issues1, err1 := syscall.NewGOSyscallAnalyser(prof).Analyze(path)
		issues2, err2 := syscall.NewAssemblySyscallAnalyser(prof).Analyze(disassemblyPath)
		err = combineErrors(err1, err2)
		issues = append(issues1, issues2...)
	default: // Run both analyzers
		issues1, err1 := opcode.NewAnalyser(prof).Analyze(disassemblyPath)
		issues2, err2 := syscall.NewGOSyscallAnalyser(prof).Analyze(path)
		issues3, err3 := syscall.NewAssemblySyscallAnalyser(prof).Analyze(disassemblyPath)
		err = combineErrors(err1, err2, err3)
		issues = append(issues1, append(issues2, issues3...)...)
	}

	return issues, err
}

// writeReport outputs the results in the specified format.
func writeReport(issues []*analyser.Issue, format, outputPath string) error {
	var output *os.File
	if outputPath == "" {
		output = os.Stdout
	} else {
		absPath, err := filepath.Abs(outputPath)
		if err != nil {
			return fmt.Errorf("unable to determine absolute path: %w", err)
		}
		output, err = os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return fmt.Errorf("unable to open output file: %w", err)
		}
		defer output.Close()
	}

	var rendererInstance renderer.Renderer
	switch format {
	case "text":
		rendererInstance = renderer.NewTextRenderer()
	case "json":
		rendererInstance = renderer.NewJSONRenderer()
	default:
		return fmt.Errorf("invalid format: %s", format)
	}

	return rendererInstance.Render(issues, output)
}

// combineErrors merges multiple errors into a single error.
func combineErrors(errs ...error) error {
	var combinedErr error
	for _, err := range errs {
		if err != nil {
			if combinedErr == nil {
				combinedErr = err
			} else {
				combinedErr = fmt.Errorf("%v; %w", combinedErr, err)
			}
		}
	}
	return combinedErr
}
