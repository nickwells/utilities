package main

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

// addParams will add parameters to the passed ParamSet
func addParams(prog *Prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		goNameRE := regexp.MustCompile("^[A-Za-z][a-zA-Z0-9_]*$")
		ps.Add("type-name",
			psetter.String{
				Value: &prog.typeName,
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
				Value: &prog.typeDesc,
				Checks: []check.String{
					check.StringLength[string](check.ValGT(0)),
				},
			},
			"text describing the type",
			param.AltNames("desc", "d"),
			param.Attrs(param.MustBeSet),
		)

		ps.Add("value-name", psetter.StrListAppender{
			Value: &prog.constNames,
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
				Value: &prog.outputFileName,
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
				Value:  &prog.makeFile,
			},
			"don't create the go file, instead just print the content to"+
				" standard out. This is useful for debugging or just to "+
				"see what would have been produced",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("no-preamble",
			psetter.Bool{
				Invert: true,
				Value:  &prog.printPreamble,
			},
			"don't print the introductory comment showing that the code"+
				" was produced by this program",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("no-isvalid",
			psetter.Bool{
				Invert: true,
				Value:  &prog.printIsValid,
			},
			"don't print the IsValid method",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("for-testing", psetter.Bool{Value: &prog.forTesting},
			"create the type for testing only")

		ps.AddFinalCheck(func() error {
			if fileNameParam.HasBeenSet() && noFileParam.HasBeenSet() {
				return fmt.Errorf(
					"only one of %q and %q may be set at the same time",
					fileNameParam.Name(), noFileParam.Name())
			}
			if !fileNameParam.HasBeenSet() && !noFileParam.HasBeenSet() {
				suffix := ".go"
				if prog.forTesting {
					suffix = "_test.go"
				}
				prog.outputFileName = "type" + prog.typeName + suffix
			}
			return nil
		})

		ps.AddFinalCheck(func() error {
			if len(prog.constNames) < 2 {
				return errors.New("There must be at least 2 value names given")
			}
			err := check.SliceHasNoDups(prog.constNames)
			return err
		})

		return nil
	}
}
