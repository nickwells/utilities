package main

import (
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// makeParamSet creates a paramset ready for parsing
func makeParamSet(prog *prog) *param.PSet {
	return paramset.NewOrPanic(
		versionparams.AddParams,

		addParams(prog),
		param.SetProgramDescription("Report on the status of file systems.\n\n"+
			"By default the file system to be reported will be that of the"+
			" current directory '.' but you can specify a list of alternative"+
			" directories by passing them after the terminating parameter"+
			" ('"+param.DfltTerminalParam+"'). The value reported will be"+
			" the available space."),
	)
}
