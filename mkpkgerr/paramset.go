package main

import (
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/versionparams.mod/versionparams"
)

func makeParamSet(prog *Prog) *param.PSet {
	return paramset.NewOrPanic(
		gogen.AddParams(&prog.outputFileName, &prog.makeFile),
		versionparams.AddParams,

		param.SetProgramDescription(
			"This creates a Go file defining a package-specific error"+
				" type. The default name of the file is: "+dfltFileName),
	)
}
