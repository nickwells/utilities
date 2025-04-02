package main

import (
	"fmt"
	"io"
	"os"
	"unicode"

	"github.com/nickwells/gogen.mod/gogen"
)

// Created: Fri Jan 17 18:31:18 2020

const (
	dfltFileName = "pkg_err_type.go"
)

// prog holds program parameters and status
type prog struct {
	// parameters
	makeFile       bool
	outputFileName string
}

// newProg returns a new Prog instance with the default values set
func newProg() *prog {
	return &prog{
		makeFile:       true,
		outputFileName: dfltFileName,
	}
}

func main() {
	prog := newProg()
	ps := makeParamSet(prog)

	ps.Parse()

	f := os.Stdout
	if prog.makeFile {
		f = gogen.MakeFileOrDie(prog.outputFileName)
		defer f.Close()
	}

	gogen.PrintPreamble(f, ps)
	gogen.PrintImports(f, "errors", "fmt")

	printFile(f)
}

// printFile prints the file contents
func printFile(f io.Writer) {
	pkgName := gogen.GetPackageOrDie()
	r := []rune(pkgName)
	r[0] = unicode.ToUpper(r[0])
	idFunc := string(r) + "Error()"

	fmt.Fprint(f, `

type Error interface {
	error

	// `+idFunc+` is a no-op function but it serves to
	// distinguish errors from this package from other errors
	`+idFunc+`
}

type pkgError string

type pkgWError struct {
	msg string
	err error
}

// Error returns the string form of the error with an appropriate prefix
func (e pkgError) Error() string {
	return "`+pkgName+` error: " + string(e)
}

// Error returns the string form of the error with an appropriate prefix
func (e pkgWError) Error() string {
	return "`+pkgName+` error: " + e.msg
}

func (e pkgError) `+idFunc+` {}

func (e pkgWError) `+idFunc+` {}

// Unwrap returns the wrapped error
func (e pkgWError) Unwrap() error {
	return e.err
}

// pkgErrorf formats its arguments into an Error
func pkgErrorf(format string, args ...any) Error {
	e := fmt.Errorf(format, args...)
	if we := errors.Unwrap(e); we != nil {
		return pkgWError{
			msg: e.Error(),
			err: we,
		}
	}
	return pkgError(e.Error())
}
`)
}
