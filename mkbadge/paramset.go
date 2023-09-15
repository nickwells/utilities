package main

import (
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// makeParamSet generates the param set ready for parsing
func makeParamSet(prog *Prog) *param.PSet {
	return paramset.NewOrPanic(
		versionparams.AddParams,

		addParams(prog),

		SetGlobalConfigFile,
		SetConfigFile,

		param.SetProgramDescription(
			"This will print the markdown for displaying badges"+
				" in your README.md file."+
				" It will also print bracketing comments with the"+
				" expectation that it can be used in a script to"+
				" automatically maintain the badges."),
	)
}
