package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChainSafe/vm-compat/analyzer"
	"github.com/ChainSafe/vm-compat/analyzer/opcode"
	"github.com/ChainSafe/vm-compat/analyzer/syscall"
	"github.com/ChainSafe/vm-compat/disassembler"
	"github.com/ChainSafe/vm-compat/disassembler/manager"
	"github.com/ChainSafe/vm-compat/profile"
	"github.com/ChainSafe/vm-compat/renderer"
	"github.com/urfave/cli/v2"
)

// TODO: update flag type

var (
	VMProfileFlag = &cli.StringFlag{
		Name:     "vm-profile",
		Usage:    "Path to the VM profile config file",
		Required: true,
	}
	AnalysisTypeFlag = &cli.StringFlag{
		Name:     "analysis-type",
		Usage:    "Type of analysis to perform. Options: opcode, syscall",
		Required: false,
	}
	DisassemblyOutputFlag = &cli.PathFlag{
		Name:     "disassembly-output-path",
		Usage:    "File path to store the disassembled assembly code",
		Required: false,
	}
	FormatFlag = &cli.StringFlag{
		Name:        "format",
		Usage:       "format of the output. Options: json, text",
		Required:    false,
		DefaultText: "text",
	}
	ReportOutputPathFlag = &cli.PathFlag{
		Name:     "report-output-path",
		Usage:    "output file path for report. Default: stdout",
		Required: false,
	}
	TraceFlag = &cli.BoolFlag{
		Name:     "with-trace",
		Usage:    "enable full stack trace output",
		Required: false,
		Value:    false,
	}
)

func CreateAnalyzeCommand(action cli.ActionFunc) *cli.Command {
	return &cli.Command{
		Name:        "analyze",
		Usage:       "Checks the program compatibility against the VM profile",
		Description: "Checks the program compatibility against the VM profile",
		Action:      action,
		Flags: []cli.Flag{
			VMProfileFlag,
			AnalysisTypeFlag,
			DisassemblyOutputFlag,
			FormatFlag,
			ReportOutputPathFlag,
			TraceFlag,
		},
	}
}

var AnalyzeCommand = CreateAnalyzeCommand(AnalyzeCompatibility)

func AnalyzeCompatibility(ctx *cli.Context) error {
	vmProfile := ctx.Path(VMProfileFlag.Name)
	prof, err := profile.LoadProfile(vmProfile)
	if err != nil {
		return fmt.Errorf("error loading profile: %w", err)
	}

	source := ctx.Args().First()
	disassemblyPath := ctx.Path(DisassemblyOutputFlag.Name)
	format := ctx.String(FormatFlag.Name)
	reportOutputPath := ctx.Path(ReportOutputPathFlag.Name)
	analysisType := ctx.String(AnalysisTypeFlag.Name)
	withTrace := ctx.Bool(TraceFlag.Name)

	disassemblyPath, err = disassemble(prof, source, disassemblyPath)
	if err != nil {
		return fmt.Errorf("error disassembling the file: %w", err)
	}

	issues, err := analyze(prof, disassemblyPath, analysisType, withTrace)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	if err := writeReport(issues, format, reportOutputPath, prof); err != nil {
		return fmt.Errorf("unable to write report: %w", err)
	}
	return nil
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
func analyze(prof *profile.VMProfile, disassemblyPath, mode string, withTrace bool) ([]*analyzer.Issue, error) {
	if mode == "opcode" {
		return opcode.NewAnalyser(prof).Analyze(disassemblyPath, withTrace)
	}
	if mode == "syscall" {
		return syscall.NewAssemblySyscallAnalyser(prof).Analyze(disassemblyPath, withTrace)
	}
	// by default analyze both
	opIssues, err := opcode.NewAnalyser(prof).Analyze(disassemblyPath, withTrace)
	if err != nil {
		return nil, err
	}
	sysIssues, err := syscall.NewAssemblySyscallAnalyser(prof).Analyze(disassemblyPath, withTrace)
	if err != nil {
		return nil, err
	}

	return append(opIssues, sysIssues...), nil
}

// writeReport outputs the results in the specified format.
func writeReport(issues []*analyzer.Issue, format, outputPath string, prof *profile.VMProfile) error {
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
