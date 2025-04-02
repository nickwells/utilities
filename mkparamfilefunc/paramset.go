package main

import (
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/verbose.mod/verbose"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// makeParamSet returns an initialised PSet
func makeParamSet(prog *prog) *param.PSet {
	return paramset.NewOrPanic(
		gogen.AddParams(&prog.outputFileName, &prog.makeFile),
		verbose.AddParams,
		versionparams.AddParams,

		addParams(prog),

		param.SetProgramDescription(
			"This creates a Go file defining functions which set the"+
				" default parameter files for the package or program."+
				" These can be passed as another argument to the call"+
				" where you create the parameter set or called"+
				" directly, passing the parameter set and checking for"+
				" errors. The paths of the files are derived from the"+
				" XDG config directories and from the import path of"+
				" the package."+
				"\n\nIf a group name is given the output filename and"+
				" the function names will be derived from the group"+
				" name."+
				"\n\nIt may be called multiple times in the same"+
				" package directory with different group names and"+
				" with none and each time it will generate the"+
				" appropriate files, overwriting any previous files"+
				" with the same name"),
	)
}
