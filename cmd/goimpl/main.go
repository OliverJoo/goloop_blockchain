package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/tools/imports"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, strings.Join([]string{
		"goimpl <file> <impl-package> <impl-struct> <interface>",
		"",
		"Example:",
		"    goimpl blockbase.go test BlockBase module.Block",
		"",
	}, "\n"))
}

func main() {
	if len(os.Args) != 5 {
		printUsage()
		os.Exit(1)
	}
	file := os.Args[1]
	pkg := os.Args[2]
	strt := os.Args[3]
	inf := os.Args[4]
	cmd := exec.Command("go",
		"run",
		"github.com/josharian/impl",
		fmt.Sprintf("_r *%s", strt),
		inf,
	)
	cmd.Stderr = os.Stderr
	cout, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}
	lines := strings.Join([]string{
		"// Code generated by go generate; DO NOT EDIT.",
		"package %s",
		"type %s struct{}",
		"%s",
	}, "\n")
	out := fmt.Sprintf(lines, pkg, strt, cout)

	iout, err := imports.Process("", []byte(out), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}

	err = ioutil.WriteFile(file, iout, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}
}
