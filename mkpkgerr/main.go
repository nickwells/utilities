package main

import (
	"fmt"
	"io"
	"os"
	"unicode"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// Created: Fri Jan 17 18:31:18 2020

const (
	dfltFileName = "pkg_err_type.go"
)

var makeFile = true

var outputFileName = dfltFileName

func main() {
	ps := paramset.NewOrDie(
		gogen.AddParams(&outputFileName, &makeFile),
		versionparams.AddParams,

		param.SetProgramDescription(
			"This creates a Go file defining a package-specific error"+
				" type. The default name of the file is: "+dfltFileName),
	)

	ps.Parse()

	f := os.Stdout
	if makeFile {
		f = gogen.MakeFileOrDie(outputFileName)
		defer f.Close()
	}

	gogen.PrintPreambleOrDie(f, ps)
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
