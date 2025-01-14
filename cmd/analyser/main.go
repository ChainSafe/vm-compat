package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/ChainSafe/vm-compat/analysis"
	"github.com/ChainSafe/vm-compat/profile"
)

var (
	vmProfile = flag.String("vm-profile", "", "vm profile config")
)

const usage = `
analyser: checks the program compatibility against the vm profile

Usage:

  callgraph [-vm-profile=path_to_config] package...
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

	err = analysis.AnalyseSyscalls(profile, args...)
	if err != nil {
		panic(err)
	}
}
