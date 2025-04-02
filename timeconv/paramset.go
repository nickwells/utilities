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
			"this will convert the passed date into the equivalent time"+
				" in the given timezone. If no 'from' timezone is given"+
				" the local timezone is used. Similarly for the 'to'"+
				" timezone. If no time or date is given then the current"+
				" time is used. Only one of the time or date can be"+
				" given. A time or date must be given if the 'from'"+
				" timezone is given."),
	)
}
