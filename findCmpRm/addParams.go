package main

import (
	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v6/paction"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
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
func addParams(prog *prog) param.PSetOptFunc {
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

		ps.Add(paramNameExtension,
			psetter.String[string]{
				Value: &prog.fileExtension,
				Checks: []check.String{
					check.StringLength[string](check.ValGT(0)),
				},
			},
			"give the extension for the files to search for.",
			param.AltNames("e", "suffix"),
		)

		ps.Add(paramNameTidy, psetter.Nil{},
			"delete all duplicate files."+
				"\n\n"+duplicateAction,
			param.SeeAlso(paramNameDupAction),
			param.PostAction(
				paction.SetVal(
					(*string)(&prog.dupAction),
					string(daDelete))),
		)

		ps.Add(paramNameDupAction, psetter.Enum[string]{
			Value: (*string)(&prog.dupAction),
			AllowedVals: psetter.AllowedVals[string]{
				string(daDelete): "delete all duplicate files" +
					" without prompting" +
					" (" + duplicateAction + ")",
				string(daQuery): "show all duplicate files and" +
					" prompt to see what to do with them",
				string(daKeep): "keep all duplicate files" +
					" without prompting",
			},
		}, "what action should be performed with duplicate files",
			param.AltNames("dup-action"),
			param.SeeAlso(paramNameTidy),
		)

		ps.Add(paramNameCmpAction,
			psetter.Enum[string]{
				Value: (*string)(&prog.cmpAction),
				AllowedVals: psetter.AllowedVals[string]{
					string(caShowDiff): "show file differences" +
						" without prompting",
					string(caQuery): "prompt to show differences" +
						" (the default action)",
					string(caKeepAll): "keep all comparable files" +
						" without prompting",
					string(caDeleteAll): "delete all comparable files with" +
						" the given extension" +
						" (" + dfltExtension + " by default)" +
						" without prompting",
					string(caRevertAll): "revert all comparable files back to" +
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
			psetter.String[string]{
				Value: &prog.diff.name,
				Checks: []check.String{
					check.StringLength[string](check.ValGT(0)),
				},
			},
			"give the name of the command to use when showing the"+
				" differences between files",
			param.AltNames("diff"),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("diff-cmd-params",
			psetter.StrList[string]{
				Value: &prog.diff.params,
				Checks: []check.StringSlice{
					check.SliceLength[[]string](check.ValGT(0)),
				},
			},
			"give any parameters to be supplied to the diff command.",
			param.AltNames("diff-params", "diff-args"),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("less-cmd",
			psetter.String[string]{
				Value: &prog.less.name,
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
			psetter.StrList[string]{
				Value: &prog.less.params,
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
