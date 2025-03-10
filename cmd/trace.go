// Package cmd defines all the commands for the cli
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ChainSafe/vm-compat/analyzer"

	"github.com/ChainSafe/vm-compat/analyzer/syscall"
	"github.com/ChainSafe/vm-compat/profile"
	"github.com/urfave/cli/v2"
)

var (
	FunctionNameFlag = &cli.StringFlag{
		Name:     "function",
		Usage:    "Name of the function to trace. Name should include with package name. Ex: syscall.read",
		Required: true,
	}
	SourceTypeFlag = &cli.StringFlag{
		Name:     "source-type",
		Usage:    "Tracing on 'go' source code or 'assembly' code. Default assembly",
		Required: false,
		Value:    "assembly",
	}
)

func CreateTraceCommand(action cli.ActionFunc) *cli.Command {
	return &cli.Command{
		Name:        "trace",
		Usage:       "Generates stack trace for a given function",
		Description: "Generates stack trace for a given function",
		Action:      action,
		Flags: []cli.Flag{
			VMProfileFlag,
			VMProfileConfigFlag,
			FunctionNameFlag,
			SourceTypeFlag,
		},
	}
}

var TraceCommand = CreateTraceCommand(TraceCaller)

func TraceCaller(ctx *cli.Context) error {
	var vmProfile *profile.VMProfile
	var err error
	prof := ctx.Path(VMProfileFlag.Name)
	if prof != "" {
		vmProfile, err = profile.LoadProfile(prof)
	} else {
		profPath := ctx.Path(VMProfileConfigFlag.Name)
		vmProfile, err = profile.LoadProfileFromConfig(profPath)
	}

	if err != nil {
		return fmt.Errorf("error loading profile: %w", err)
	}

	function := ctx.String(FunctionNameFlag.Name)
	sourceType := ctx.String(SourceTypeFlag.Name)
	path := ctx.Args().First()

	var analyzer analyzer.Analyzer
	if sourceType == "go" {
		analyzer = syscall.NewGOSyscallAnalyser(vmProfile)
	} else {
		analyzer = syscall.NewAssemblySyscallAnalyser(vmProfile)
	}

	callStack, err := analyzer.TraceStack(path, function)
	if err != nil {
		return err
	}
	str := printCallStack(callStack, "")
	_, err = os.Stdout.WriteString(str)
	if err != nil {
		return err
	}
	return nil
}

func printCallStack(source *analyzer.CallStack, str string) string {
	fileInfo := fmt.Sprintf(
		" \033[94m\033]8;;file://%s:%d\033\\%s:%d\033]8;;\033\\\033[0m",
		source.AbsPath, source.Line, source.File, source.Line,
	)
	str = strings.Join(
		[]string{str, fmt.Sprintf("-> %s : (%s)", fileInfo, source.Function)}, "\n")
	if source.CallStack != nil {
		return printCallStack(source.CallStack, str)
	}
	return str
}
