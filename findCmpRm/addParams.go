// fileChecker
package main

import (
	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	const recurseParam = "recursive"

	ps.Add("dir",
		psetter.Pathname{
			Value:       &dir,
			Expectation: filecheck.DirExists(),
		},
		"give the name of the directory to search for files"+
			"\nNote that only the directory itself is searched,"+
			" sub-directories are ignored unless"+
			" the "+recurseParam+" parameter is also given",
		param.AltName("d"),
	)

	ps.Add(recurseParam, psetter.Bool{Value: &searchSubDirs},
		"this makes the command search sub-directories for matching"+
			" files not just the given directory",
		param.AltName("r"),
	)

	ps.Add("extension",
		psetter.String{
			Value:  &fileExtension,
			Checks: []check.String{check.StringLenGT(0)},
		},
		"give the extension for the files to search for",
		param.AltName("e"),
	)

	ps.Add("tidy", psetter.Bool{Value: &tidyFiles},
		"this makes the command tidy any redundant files."+
			"\n\n"+
			" Redundant"+
			" means files where there is no F corresponding to the F.orig"+
			" or where the F corresponding to the F.orig is"+
			" a directory not a file"+
			" or where F is identical to F.orig."+
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
		param.AltName("diff"),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("diff-cmd-params",
		psetter.StrList{
			Value:  &diffCmdParams,
			Checks: []check.StringSlice{check.StringSliceLenGT(0)},
		},
		"give any parameters to be supplied to the diff command",
		param.AltName("diff-params"),
		param.AltName("diff-args"),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("less-cmd",
		psetter.String{
			Value:  &lessCmdName,
			Checks: []check.String{check.StringLenGT(0)},
		},
		"give the name of the command to use for paginating the"+
			" differences calculated by the diff command",
		param.AltName("less"),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("less-cmd-params",
		psetter.StrList{
			Value:  &lessCmdParams,
			Checks: []check.StringSlice{check.StringSliceLenGT(0)},
		},
		"give any parameters to be supplied to the less command",
		param.AltName("less-params"),
		param.AltName("less-args"),
		param.Attrs(param.DontShowInStdUsage),
	)

	return nil
}
