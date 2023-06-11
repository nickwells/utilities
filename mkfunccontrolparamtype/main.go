package main

import (
	"fmt"
	"os"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v5/param"
)

// Created: Sat Mar 21 11:18:36 2020

// Prog holds program parameters and status
type Prog struct {
	typeName       string
	typeDesc       string
	outputFileName string

	makeFile      bool
	printPreamble bool
	printIsValid  bool
	forTesting    bool

	constNames []string
}

// NewProg returns an initialised Prog struct
func NewProg() *Prog {
	return &Prog{
		makeFile:      true,
		printPreamble: true,
		printIsValid:  true,
	}
}

func main() {
	prog := NewProg()
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
func (prog *Prog) printFile(f *os.File, ps *param.PSet) {
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
func (prog *Prog) printTypeDeclaration(f *os.File) {
	fullTypeName := prog.typeName + "Type"

	fmt.Fprintf(f, "// %s %s\n", fullTypeName, prog.typeDesc)
	fmt.Fprintf(f, "type %s int\n", fullTypeName)

	suffix := " " + fullTypeName + " = iota"
	fmt.Fprint(f, `
const (
`)
	for _, v := range prog.constNames {
		fmt.Fprintln(f, "\t"+v+suffix)
		suffix = ""
	}
	fmt.Fprint(f, `)

`)
}

// printIsValidFunc writes the IsValid function to the file being generated
func (prog *Prog) printIsValidFunc(f *os.File) {
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
	fmt.Fprintf(f, "\tif v > %s {\n", prog.constNames[len(prog.constNames)-1])
	fmt.Fprintf(f, "\t\treturn false\n")
	fmt.Fprintln(f, "\t}")
	fmt.Fprintf(f, "\treturn true\n")
	fmt.Fprintln(f, "}")
}
