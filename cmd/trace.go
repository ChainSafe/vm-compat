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
)

func CreateTraceCommand(action cli.ActionFunc) *cli.Command {
	return &cli.Command{
		Name:        "trace",
		Usage:       "Generates stack trace for a given function",
		Description: "Generates stack trace for a given function",
		Action:      action,
		Flags: []cli.Flag{
			VMProfileFlag,
			FunctionNameFlag,
		},
	}
}

var TraceCommand = CreateTraceCommand(TraceCaller)

func TraceCaller(ctx *cli.Context) error {
	vmProfile := ctx.Path(VMProfileFlag.Name)
	prof, err := profile.LoadProfile(vmProfile)
	if err != nil {
		return fmt.Errorf("error loading profile: %w", err)
	}

	function := ctx.Path(FunctionNameFlag.Name)
	source := ctx.Args().First()

	analyzer := syscall.NewGOSyscallAnalyser(prof)
	callStack, err := analyzer.TraceStack(source, function)
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

func printCallStack(source *analyzer.IssueSource, str string) string {
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
