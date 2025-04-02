package main

import (
	"fmt"
	"os"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/twrap.mod/twrap"
)

// Created: Sat Mar 21 11:18:36 2020

// prog holds program parameters and status
type prog struct {
	typeName       string
	typeDesc       string
	outputFileName string

	makeFile      bool
	printPreamble bool
	printIsValid  bool
	forTesting    bool

	constNames []string
}

// newProg returns an initialised Prog struct
func newProg() *prog {
	return &prog{
		makeFile:      true,
		printPreamble: true,
		printIsValid:  true,
	}
}

func main() {
	prog := newProg()
	ps := makeParamSet(prog)

	ps.Parse()

	f := os.Stdout

	if prog.makeFile {
		if prog.forTesting {
			f = gogen.MakeTestFileOrDie(prog.outputFileName)
		} else {
			f = gogen.MakeFileOrDie(prog.outputFileName)
		}

		defer f.Close()
	}

	prog.printFile(f, ps)
}

// printFile writes the contents of the file being generated
func (prog *prog) printFile(f *os.File, ps *param.PSet) {
	if prog.printPreamble {
		gogen.PrintPreamble(f, ps)

		fmt.Fprintln(f)
	}

	prog.printTypeDeclaration(f)

	if prog.printIsValid {
		prog.printIsValidFunc(f)
	}
}

// printTypeDeclaration writes the type declaration and constant values to
// the file
func (prog *prog) printTypeDeclaration(f *os.File) {
	const targetLineLen = 75

	fullTypeName := prog.typeName + "Type"

	twc := twrap.NewTWConfOrPanic(
		twrap.SetWriter(f),
		twrap.SetTargetLineLen(targetLineLen))

	fmt.Fprintln(f, "/*")
	twc.Wrap(fullTypeName+" "+prog.typeDesc, 0)
	fmt.Fprintln(f, "*/")
	fmt.Fprintf(f, "type %s int\n\n", fullTypeName)

	fmt.Fprintf(f, "// These constants are the allowed values of %s\n",
		fullTypeName)
	fmt.Fprintln(f, "const (")

	suffix := " " + fullTypeName + " = iota"

	for _, v := range prog.constNames {
		fmt.Fprintln(f, "\t"+v+suffix)
		suffix = ""
	}

	fmt.Fprint(f, `)

`)
}

// printIsValidFunc writes the IsValid function to the file being generated
func (prog *prog) printIsValidFunc(f *os.File) {
	fullTypeName := prog.typeName + "Type"

	const validFuncName = "IsValid"

	fmt.Fprintf(f, "// %s is a method on the %s type that can be used",
		validFuncName, fullTypeName)
	fmt.Fprint(f, `
// to check a received parameter for validity. It compares
// the value against the boundary values for the type
// and returns false if it is outside the valid range
`)
	fmt.Fprintf(f, "func (v %s) %s() bool {\n", fullTypeName, validFuncName)
	fmt.Fprintf(f, "\tif v < %s {\n", prog.constNames[0])
	fmt.Fprintf(f, "\t\treturn false\n")
	fmt.Fprintln(f, "\t}")
	fmt.Fprintln(f)
	fmt.Fprintf(f, "\tif v > %s {\n", prog.constNames[len(prog.constNames)-1])
	fmt.Fprintf(f, "\t\treturn false\n")
	fmt.Fprintln(f, "\t}")
	fmt.Fprintln(f)
	fmt.Fprintf(f, "\treturn true\n")
	fmt.Fprintln(f, "}")
}
