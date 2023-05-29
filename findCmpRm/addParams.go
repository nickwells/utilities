package main

import (
	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

const (
	paramNameDupAction = "duplicate-action"
	paramNameTidy      = "tidy"
	paramNameCmpAction = "comparable-action"
	paramNameDir       = "dir"
	paramNameNoRecurse = "dont-recurse"
	paramNameExtension = "extension"
)

// addParams will add parameters to the passed ParamSet
func addParams(prog *Prog) param.PSetOptFunc {
	const (
		exampleFileName = "F"
		exampleOrigName = exampleFileName + dfltExtension
		duplicateAction = "if " + exampleFileName + " and " + exampleOrigName +
			" are the same, delete " + exampleOrigName
	)

	return func(ps *param.PSet) error {
		ps.Add(paramNameDir,
			psetter.Pathname{
				Value:       &prog.searchDir,
				Expectation: filecheck.DirExists(),
			},
			"give the name of the directory to search for files."+
				"\nNote that both the directory and its,"+
				" sub-directories are searched unless"+
				" the "+paramNameNoRecurse+" parameter is also given.",
			param.AltNames("search-dir", "d"),
			param.SeeAlso(paramNameNoRecurse),
		)

		ps.Add(paramNameNoRecurse,
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

		ps.Add(paramNameTidy, psetter.Nil{},
			"delete all duplicate files."+
				"\n\n"+duplicateAction,
			param.SeeAlso(paramNameDupAction),
			param.PostAction(
				paction.SetString(
					(*string)(&prog.dupAction),
					string(DADelete))),
		)

		ps.Add(paramNameDupAction, psetter.Enum{
			Value: (*string)(&prog.dupAction),
			AllowedVals: psetter.AllowedVals{
				string(DADelete): "delete all duplicate files" +
					" without prompting" +
					" (" + duplicateAction + ")",
				string(DAQuery): "show all duplicate files and" +
					" prompt to see what to do with them",
				string(DAKeep): "keep all duplicate files" +
					" without prompting",
			},
		}, "what action should be performed with duplicate files",
			param.AltNames("dup-action"),
			param.SeeAlso(paramNameTidy),
		)

		ps.Add(paramNameCmpAction,
			psetter.Enum{
				Value: (*string)(&prog.cmpAction),
				AllowedVals: psetter.AllowedVals{
					string(CAShowDiff): "show file differences" +
						" without prompting",
					string(CAQuery): "prompt to show differences" +
						" (the default action)",
					string(CAKeepAll): "keep all comparable files" +
						" without prompting",
					string(CADeleteAll): "delete all comparable files with" +
						" the given extension" +
						" (" + dfltExtension + " by default)" +
						" without prompting",
					string(CARevertAll): "revert all comparable files back to" +
						" the contents of the file with" +
						" the given extension" +
						" (" + dfltExtension + " by default)" +
						" without prompting",
				},
			},
			"what action should be performed with comparable files",
			param.AltNames("cmp-action"),
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
