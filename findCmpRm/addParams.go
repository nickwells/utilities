// fileChecker
package main

import (
	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v3/param"
	"github.com/nickwells/param.mod/v3/param/psetter"
)

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	const recurseParam = "recursive"

	ps.Add("dir",
		psetter.Pathname{
			Value: &dir,
			Expectation: filecheck.Provisos{
				Existence: filecheck.MustExist,
				Checks:    []check.FileInfo{check.FileInfoIsDir},
			},
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
			Value:  &extension,
			Checks: []check.String{check.StringLenGT(0)},
		},
		"give the extension for the files to search for",
		param.AltName("e"),
	)

	ps.Add("diff-cmd",
		psetter.String{
			Value:  &diffCmdName,
			Checks: []check.String{check.StringLenGT(0)},
		},
		"give the name of the command to use when showing the"+
			" differences between files",
		param.AltName("dc"),
		param.AltName("cmd"),
	)

	ps.Add("diff-cmd-params",
		psetter.StrList{
			Value:  &diffCmdParams,
			Checks: []check.StringSlice{check.StringSliceLenGT(0)},
		},
		"give any parameters to be supplied to the diff command",
		param.AltName("cmd-params"),
		param.AltName("dcp"),
	)

	ps.Add("less-cmd",
		psetter.String{
			Value:  &lessCmdName,
			Checks: []check.String{check.StringLenGT(0)},
		},
		"give the name of the command to use for paginating the"+
			" differences calculated by the diff command",
		param.AltName("lc"),
	)

	ps.Add("less-cmd-params",
		psetter.StrList{
			Value:  &lessCmdParams,
			Checks: []check.StringSlice{check.StringSliceLenGT(0)},
		},
		"give any parameters to be supplied to the less command",
		param.AltName("lcp"),
	)

	return nil
}
