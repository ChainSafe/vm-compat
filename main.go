package main

import (
	"context"
	"log"
	"os"

	"github.com/ChainSafe/vm-compat/cmd"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = os.Args[0]
	app.Usage = "VM Compatibility Analyzer"
	app.Description = "VM Compatibility Analyzer"
	app.Commands = []*cli.Command{
		cmd.AnalyzeCommand,
		cmd.TraceCommand,
	}
	err := app.RunContext(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
