package main

import (
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/verbose.mod/verbose"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// makeParamSet generates the param set ready for parsing
func makeParamSet(prog *prog) *param.PSet {
	return paramset.NewOrPanic(
		verbose.AddParams,
		versionparams.AddParams,

		addParams(prog),
		addTimeParams(prog),
		addActionParams(prog),

		addExamples,

		param.SetProgramDescription(
			"This will sleep until a given time and then perform the"+
				" chosen actions."+
				"\n\n"+
				"You can specify either a particular time of day to sleep"+
				" until or some fragment of the day or some regular"+
				" period (which must divide the day into a whole number"+
				" of parts)."+
				"\n\n"+
				" So for instance you could choose to sleep until the next"+
				" hour and it will wake up at minute 00 rather than"+
				" 60 minutes later."+
				"\n\n"+
				"You can give an offset to the regular time and the delay"+
				" will be adjusted accordingly."),
	)
}
