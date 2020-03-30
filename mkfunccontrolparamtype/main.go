// mkfunccontrolparamtype
package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v4/param"
	"github.com/nickwells/param.mod/v4/param/paramset"
	"github.com/nickwells/param.mod/v4/param/psetter"
)

// Created: Sat Mar 21 11:18:36 2020

var typeName string
var typeDesc string
var outputFileName string
var makeFile = true
var constNames []string

func main() {
	ps := paramset.NewOrDie(addParams,
		param.SetProgramDescription(
			"this generates a Go file containing the definition of a type"+
				" that can be used to provide a parameter to a function"+
				" that controls the behaviour of that function"),
	)

	ps.Parse()

	var f = os.Stdout
	if makeFile {
		f = gogen.MakeFileOrDie(outputFileName)
		defer f.Close()
	}

	printFile(f, ps)
}

// printFile writes the contents of the file being generated
func printFile(f *os.File, ps *param.PSet) {
	gogen.PrintPreambleOrDie(f, ps)

	fmt.Fprintln(f)

	fmt.Fprintf(f, "// %s %s\n", typeName, typeDesc)
	fmt.Fprintf(f, "type %s int\n\n", typeName)

	namePfx := "T" + typeName
	minValName := namePfx + "MinVal"
	maxValName := namePfx + "MaxVal"
	fmt.Fprint(f, `const (
`)
	fmt.Fprintf(f, "\t%s %s = iota\n", minValName, typeName)
	for _, v := range constNames {
		fmt.Fprintf(f, "\t%s%s\n", namePfx, v)
	}
	fmt.Fprintf(f, "\t%s\n", maxValName)
	fmt.Fprint(f, `)

`)

	const validFuncName = "IsValid"

	fmt.Fprintf(f, "// %s is a method on the %s type that can be used\n",
		validFuncName, typeName)
	fmt.Fprintln(f, "// to check a received parameter and return an error")
	fmt.Fprintln(f, "// or panic if an illegal parameter value is passed")
	fmt.Fprintf(f, "func (v %s)%s() bool {\n", typeName, validFuncName)
	fmt.Fprintf(f, "\tif v <= %s {\n", minValName)
	fmt.Fprintf(f, "\t\treturn false\n")
	fmt.Fprintln(f, "\t}")
	fmt.Fprintf(f, "\tif v >= %s {\n", maxValName)
	fmt.Fprintf(f, "\t\treturn false\n")
	fmt.Fprintln(f, "\t}")
	fmt.Fprintf(f, "\treturn true\n")
	fmt.Fprintln(f, "}")
}

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	goNameRE := regexp.MustCompile("[A-Z][a-zA-Z0-9]*")
	ps.Add("type-name",
		psetter.String{
			Value: &typeName,
			Checks: []check.String{
				check.StringMatchesPattern(goNameRE,
					"a valid Go identifier"),
			},
		},
		"give the name of the integer type to be created",
		param.AltName("type"),
		param.AltName("t"),
		param.Attrs(param.MustBeSet),
	)

	ps.Add("description",
		psetter.String{
			Value: &typeDesc,
			Checks: []check.String{
				check.StringLenGT(0),
			},
		},
		"text describing the type",
		param.AltName("desc"),
		param.AltName("d"),
		param.Attrs(param.MustBeSet),
	)

	ps.Add("value-name", psetter.StrListAppender{
		Value: &constNames,
		Checks: []check.String{
			check.StringMatchesPattern(goNameRE,
				"a valid Go identifier"),
		},
	},
		"follow this with the name of one of the constant values you want",
		param.AltName("value"),
		param.AltName("val"),
		param.AltName("v"),
		param.Attrs(param.MustBeSet),
	)

	fileNameParam := ps.Add("output-file-name",
		psetter.Pathname{
			Value: &outputFileName,
			Checks: []check.String{
				check.StringHasSuffix(".go"),
				check.StringNot(
					check.StringHasSuffix("_test.go"),
					"a test file"),
			},
		},
		"set the name of the output file",
		param.AltName("o"),
		param.Attrs(param.DontShowInStdUsage),
	)

	noFileParam := ps.Add("no-file",
		psetter.Bool{
			Invert: true,
			Value:  &makeFile,
		},
		"don't create the go file, instead just print the content to"+
			" standard out. This is useful for debugging or just to "+
			"see what would have been produced",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.AddFinalCheck(func() error {
		if fileNameParam.HasBeenSet() && noFileParam.HasBeenSet() {
			return fmt.Errorf(
				"only one of %q and %q may be set at the same time",
				fileNameParam.Name(), noFileParam.Name())
		}
		if !fileNameParam.HasBeenSet() && !noFileParam.HasBeenSet() {
			outputFileName = "type" + typeName + ".go"
		}
		return nil
	})

	ps.AddFinalCheck(func() error {
		if len(constNames) < 2 {
			return errors.New("There must be at least 2 value names given")
		}
		err := check.StringSliceNoDups(constNames)
		return err
	})

	return nil
}
