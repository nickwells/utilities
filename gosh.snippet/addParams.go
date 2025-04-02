package main

import (
	"fmt"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v6/paction"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
)

const (
	paramNameAction     = "action"
	paramNameInstall    = "install"
	paramNameTarget     = "target"
	paramNameSource     = "source"
	paramNameMaxSubDirs = "max-sub-dirs"
	paramNameNoCopy     = "no-copy"
)

// addParams will add parameters to the passed ParamSet
func addParams(prog *prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add(paramNameAction,
			psetter.Enum[string]{
				Value: &prog.action,
				AllowedVals: psetter.AllowedVals[string]{
					installAction: "install the default snippets in" +
						" the target directory",
					cmpAction: "compare the default snippets with" +
						" those in the target directory",
				},
			},
			"what action should be performed",
			param.AltNames("a"),
			param.Attrs(param.CommandLineOnly),
		)

		ps.Add(paramNameInstall, psetter.Nil{},
			"install the snippets.",
			param.PostAction(paction.SetVal(&prog.action, installAction)),
			param.Attrs(param.CommandLineOnly),
			param.SeeAlso(paramNameAction),
		)

		ps.Add(paramNameTarget,
			psetter.Pathname{
				Value: &prog.toDir,
				Checks: []check.String{
					check.StringLength[string](check.ValGT(0)),
				},
			},
			"set the directory where the snippets are to be copied or compared.",
			param.AltNames("to", "to-dir", "t"),
			param.Attrs(param.CommandLineOnly|param.MustBeSet),
		)

		ps.Add(paramNameSource,
			psetter.Pathname{
				Value:       &prog.fromDir,
				Expectation: filecheck.DirExists(),
			},
			"set the directory where the snippets are to be found."+
				" If this is not set then the standard collection of"+
				" snippets will be used.",
			param.AltNames("from", "from-dir", "f"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		const minSubDirs = 3

		ps.Add(paramNameMaxSubDirs,
			psetter.Int[int64]{
				Value:  &prog.maxSubDirs,
				Checks: []check.Int64{check.ValGE[int64](minSubDirs)},
			},
			"how many levels of sub-directory are allowed before we"+
				" assume there is a loop in the directory path."+
				"\n\n"+
				fmt.Sprintf("This must be at least %d.", minSubDirs),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add(paramNameNoCopy,
			psetter.Bool{Value: &prog.noCopy},
			"suppress the copying of existing files which have"+
				" changed and are being replaced."+
				"\n\n"+
				"NOTE: this deletes files from the target directory"+
				" which have the same name as files from the source."+
				" The original files cannot be recovered, no copy is kept.",
			param.AltNames("no-backup"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		return nil
	}
}
