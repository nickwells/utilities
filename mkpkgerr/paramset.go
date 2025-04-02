package main

import (
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/versionparams.mod/versionparams"
)

func makeParamSet(prog *prog) *param.PSet {
	return paramset.NewOrPanic(
		gogen.AddParams(&prog.outputFileName, &prog.makeFile),
		versionparams.AddParams,

		param.SetProgramDescription(
			"This creates a Go file defining a package-specific error"+
				" type. The default name of the file is: "+dfltFileName),
	)
}
