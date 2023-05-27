package main

import (
	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

// addParams will add parameters to the passed ParamSet
func addParams(prog *Prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		const noRecurseParam = "dont-recurse"

		ps.Add("dir",
			psetter.Pathname{
				Value:       &prog.searchDir,
				Expectation: filecheck.DirExists(),
			},
			"give the name of the directory to search for files."+
				"\nNote that both the directory and its,"+
				" sub-directories are searched unless"+
				" the "+noRecurseParam+" parameter is also given.",
			param.AltNames("search-dir", "d"),
		)

		ps.Add(noRecurseParam,
			psetter.Bool{
				Value:  &prog.searchSubDirs,
				Invert: true,
			},
			"this makes the command only search the given directory,"+
				" it will not recursively search sub-directories.",
			param.AltNames("no-recurse", "no-rec", "no-r"),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("extension",
			psetter.String{
				Value: &prog.fileExtension,
				Checks: []check.String{
					check.StringLength[string](check.ValGT(0)),
				},
			},
			"give the extension for the files to search for.",
			param.AltNames("e"),
		)

		ps.Add("tidy", psetter.Bool{Value: &prog.tidyFiles},
			"this makes the command remove any duplicate files."+
				"\n\n"+
				"This will remove the F.orig file corresponding to the file F.",
		)

		ps.Add("diff-cmd",
			psetter.String{
				Value: &prog.diffCmdName,
				Checks: []check.String{
					check.StringLength[string](check.ValGT(0))},
			},
			"give the name of the command to use when showing the"+
				" differences between files",
			param.AltNames("diff"),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("diff-cmd-params",
			psetter.StrList{
				Value: &prog.diffCmdParams,
				Checks: []check.StringSlice{
					check.SliceLength[[]string](check.ValGT(0)),
				},
			},
			"give any parameters to be supplied to the diff command.",
			param.AltNames("diff-params", "diff-args"),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("less-cmd",
			psetter.String{
				Value: &prog.lessCmdName,
				Checks: []check.String{
					check.StringLength[string](check.ValGT(0)),
				},
			},
			"give the name of the command to use for paginating the"+
				" differences calculated by the diff command.",
			param.AltNames("less"),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("less-cmd-params",
			psetter.StrList{
				Value: &prog.lessCmdParams,
				Checks: []check.StringSlice{
					check.SliceLength[[]string](check.ValGT(0)),
				},
			},
			"give any parameters to be supplied to the less command.",
			param.AltNames("less-params", "less-args"),
			param.Attrs(param.DontShowInStdUsage),
		)

		return nil
	}
}
