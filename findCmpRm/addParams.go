package main

import (
	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	const noRecurseParam = "dont-recurse"

	ps.Add("dir",
		psetter.Pathname{
			Value:       &searchDir,
			Expectation: filecheck.DirExists(),
		},
		"give the name of the directory to search for files."+
			"\nNote that both the directory and its,"+
			" sub-directories are searched unless"+
			" the "+noRecurseParam+" parameter is also given.",
		param.AltNames("search-dir", "d"),
	)

	ps.Add(noRecurseParam, psetter.Bool{Value: &searchSubDirs, Invert: true},
		"this makes the command only search the given directory,"+
			" it will not recursively search sub-directories.",
		param.AltNames("no-recurse", "no-rec", "no-r"),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("recursive", psetter.Bool{Value: &searchSubDirs},
		"this makes the command search sub-directories for matching"+
			" files, not just the given directory."+
			"\n\n"+
			"Note: this is already the default behaviour so this"+
			" parameter is redundant but it is kept for backwards"+
			" compatibility.",
		param.AltNames("r"),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("extension",
		psetter.String{
			Value:  &fileExtension,
			Checks: []check.String{check.StringLenGT(0)},
		},
		"give the extension for the files to search for.",
		param.AltNames("e"),
	)

	ps.Add("tidy", psetter.Bool{Value: &tidyFiles},
		"this makes the command tidy any redundant files."+
			"\n\n"+
			"Redundant means files where"+
			"\n - there is no F corresponding to the F.orig file"+
			"\n - or the F corresponding to the F.orig is"+
			" a directory not a file"+
			"\n - or F is identical to F.orig."+
			"\n\n"+
			" Tidy means to remove the F.orig file.",
	)

	ps.Add("diff-cmd",
		psetter.String{
			Value:  &diffCmdName,
			Checks: []check.String{check.StringLenGT(0)},
		},
		"give the name of the command to use when showing the"+
			" differences between files",
		param.AltNames("diff"),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("diff-cmd-params",
		psetter.StrList{
			Value:  &diffCmdParams,
			Checks: []check.StringSlice{check.StringSliceLenGT(0)},
		},
		"give any parameters to be supplied to the diff command.",
		param.AltNames("diff-params", "diff-args"),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("less-cmd",
		psetter.String{
			Value:  &lessCmdName,
			Checks: []check.String{check.StringLenGT(0)},
		},
		"give the name of the command to use for paginating the"+
			" differences calculated by the diff command.",
		param.AltNames("less"),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("less-cmd-params",
		psetter.StrList{
			Value:  &lessCmdParams,
			Checks: []check.StringSlice{check.StringSliceLenGT(0)},
		},
		"give any parameters to be supplied to the less command.",
		param.AltNames("less-params", "less-args"),
		param.Attrs(param.DontShowInStdUsage),
	)

	return nil
}
