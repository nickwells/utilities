package main

import (
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/verbose.mod/verbose"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// makeParamSet generates the param set ready for parsing
func makeParamSet(prog *Prog) *param.PSet {
	return paramset.NewOrPanic(
		verbose.AddParams,
		verbose.AddTimingParams(prog.dbgStack),
		versionparams.AddParams,

		addParams(prog),

		addExamples,
		addNotes,

		param.SetProgramDescription(
			"This will search for directories containing Go packages. You"+
				" can add extra criteria for selecting the directory."+
				" Once in each selected directory you can perform certain"+
				" actions"),
	)
}
