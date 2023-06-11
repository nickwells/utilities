package main

import (
	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

// addParams will add parameters to the passed ParamSet
func addParams(prog *Prog) param.PSetOptFunc {
	const (
		actionParamName = "action"
	)
	return func(ps *param.PSet) error {
		ps.Add(actionParamName,
			psetter.Enum{
				Value: &prog.action,
				AllowedVals: psetter.AllowedVals{
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

		ps.Add("install", psetter.Nil{},
			"install the snippets.",
			param.PostAction(paction.SetVal(&prog.action, installAction)),
			param.Attrs(param.CommandLineOnly),
			param.SeeAlso(actionParamName),
		)

		ps.Add("target",
			psetter.Pathname{
				Value: &prog.toDir,
				Checks: []check.String{
					check.StringLength[string](check.ValGT[int](0)),
				},
			},
			"set the directory where the snippets are to be copied or compared.",
			param.AltNames("to", "to-dir", "t"),
			param.Attrs(param.CommandLineOnly|param.MustBeSet),
		)

		ps.Add("source",
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

		ps.Add("max-sub-dirs",
			psetter.Int64{
				Value:  &prog.maxSubDirs,
				Checks: []check.Int64{check.ValGT[int64](2)},
			},
			"how many levels of sub-directory are allowed before we assume"+
				" there is a loop in the directory path",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("no-copy", psetter.Bool{Value: &prog.noCopy},
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
