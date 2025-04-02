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

		SetGlobalConfigFile,
		SetConfigFile,
		addNotes(prog),

		param.SetProgramDescription(
			"This creates markdown documentation for any Go program which"+
				" uses the param package"+
				" (github.com/nickwells/param.mod/*/param). It will"+
				" generate Markdown files containing various sections from"+
				" the program's help documentation."+
				" On successful completion a brief"+
				" message giving the text to be added to the README.md"+
				" file will be printed"),
	)
}
