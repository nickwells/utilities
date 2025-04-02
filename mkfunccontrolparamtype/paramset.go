package main

import (
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// makeParamSet generates the param set ready for parsing
func makeParamSet(prog *prog) *param.PSet {
	return paramset.NewOrPanic(
		versionparams.AddParams,

		addParams(prog),

		param.SetProgramDescription(
			"this generates a Go file containing the definition of a type"+
				" that can be used to provide a parameter to a function"+
				" that controls the behaviour of that function"),
	)
}
