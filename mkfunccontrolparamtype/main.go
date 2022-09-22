package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/param.mod/v5/param/psetter"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// Created: Sat Mar 21 11:18:36 2020

var (
	typeName       string
	typeDesc       string
	outputFileName string

	makeFile      = true
	printPreamble = true
	printIsValid  = true
	forTesting    bool

	constNames []string
)

func main() {
	ps := paramset.NewOrDie(
		versionparams.AddParams,

		addParams,

		param.SetProgramDescription(
			"this generates a Go file containing the definition of a type"+
				" that can be used to provide a parameter to a function"+
				" that controls the behaviour of that function"),
	)

	ps.Parse()

	f := os.Stdout
	if makeFile {
		if forTesting {
			f = gogen.MakeTestFileOrDie(outputFileName)
		} else {
			f = gogen.MakeFileOrDie(outputFileName)
		}
		defer f.Close()
	}

	printFile(f, ps)
}

// printFile writes the contents of the file being generated
func printFile(f *os.File, ps *param.PSet) {
	if printPreamble {
		gogen.PrintPreamble(f, ps)

		fmt.Fprintln(f)
	}

	printTypeDeclaration(f)
	if printIsValid {
		printIsValidFunc(f)
	}
}

// printTypeDeclaration writes the type declaration and constant values to
// the file
func printTypeDeclaration(f *os.File) {
	fullTypeName := typeName + "Type"

	fmt.Fprintf(f, "// %s %s\n", fullTypeName, typeDesc)
	fmt.Fprintf(f, "type %s int\n", fullTypeName)

	suffix := " " + fullTypeName + " = iota"
	fmt.Fprint(f, `
const (
`)
	for _, v := range constNames {
		fmt.Fprintln(f, "\t"+v+suffix)
		suffix = ""
	}
	fmt.Fprint(f, `)

`)
}

// printIsValidFunc writes the IsValid function to the file being generated
func printIsValidFunc(f *os.File) {
	fullTypeName := typeName + "Type"

	const validFuncName = "IsValid"

	fmt.Fprintf(f, "// %s is a method on the %s type that can be used",
		validFuncName, fullTypeName)
	fmt.Fprint(f, `
// to check a received parameter for validity. It compares
// the value against the boundary values for the type
// and returns false if it is outside the valid range
`)
	fmt.Fprintf(f, "func (v %s) %s() bool {\n", fullTypeName, validFuncName)
	fmt.Fprintf(f, "\tif v < %s {\n", constNames[0])
	fmt.Fprintf(f, "\t\treturn false\n")
	fmt.Fprintln(f, "\t}")
	fmt.Fprintf(f, "\tif v > %s {\n", constNames[len(constNames)-1])
	fmt.Fprintf(f, "\t\treturn false\n")
	fmt.Fprintln(f, "\t}")
	fmt.Fprintf(f, "\treturn true\n")
	fmt.Fprintln(f, "}")
}

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	goNameRE := regexp.MustCompile("^[A-Za-z][a-zA-Z0-9_]*$")
	ps.Add("type-name",
		psetter.String{
			Value: &typeName,
			Checks: []check.String{
				check.StringMatchesPattern[string](goNameRE,
					"a valid Go identifier"),
			},
		},
		"give the name of the integer type to be created."+
			" Note that a name starting with a lowercase letter"+
			" will be private to the package."+
			" Also, any name given here will have 'Type' appended.",
		param.AltNames("type", "t"),
		param.Attrs(param.MustBeSet),
	)

	ps.Add("description",
		psetter.String{
			Value: &typeDesc,
			Checks: []check.String{
				check.StringLength[string](check.ValGT(0)),
			},
		},
		"text describing the type",
		param.AltNames("desc", "d"),
		param.Attrs(param.MustBeSet),
	)

	ps.Add("value-name", psetter.StrListAppender{
		Value: &constNames,
		Checks: []check.String{
			check.StringMatchesPattern[string](goNameRE,
				"a valid Go identifier"),
		},
	},
		"follow this with the name of one of the constant values you want",
		param.AltNames("value", "val", "v"),
		param.Attrs(param.MustBeSet),
	)

	fileNameParam := ps.Add("output-file-name",
		psetter.Pathname{
			Value: &outputFileName,
			Checks: []check.String{
				check.StringHasSuffix[string](".go"),
				check.Not[string](
					check.StringHasSuffix[string]("_test.go"),
					"a test file"),
			},
		},
		"set the name of the output file",
		param.AltNames("o"),
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

	ps.Add("no-preamble",
		psetter.Bool{
			Invert: true,
			Value:  &printPreamble,
		},
		"don't print the introductory comment showing that the code"+
			" was produced by this program",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("no-isvalid",
		psetter.Bool{
			Invert: true,
			Value:  &printIsValid,
		},
		"don't print the IsValid method",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("for-testing", psetter.Bool{Value: &forTesting},
		"create the type for testing only")

	ps.AddFinalCheck(func() error {
		if fileNameParam.HasBeenSet() && noFileParam.HasBeenSet() {
			return fmt.Errorf(
				"only one of %q and %q may be set at the same time",
				fileNameParam.Name(), noFileParam.Name())
		}
		if !fileNameParam.HasBeenSet() && !noFileParam.HasBeenSet() {
			suffix := ".go"
			if forTesting {
				suffix = "_test.go"
			}
			outputFileName = "type" + typeName + suffix
		}
		return nil
	})

	ps.AddFinalCheck(func() error {
		if len(constNames) < 2 {
			return errors.New("There must be at least 2 value names given")
		}
		err := check.SliceHasNoDups(constNames)
		return err
	})

	return nil
}
