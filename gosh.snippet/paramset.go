package main

import (
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/verbose.mod/verbose"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// makeParamSet ...
func makeParamSet(prog *prog) *param.PSet {
	return paramset.NewOrPanic(
		verbose.AddParams,
		versionparams.AddParams,

		addParams(prog),
		addExamples,
		addRefs,

		param.SetProgramDescription(
			"This can install the standard collection of useful snippets."+
				" It can also be used to install snippets from a"+
				" directory or to compare two collections of snippets."+
				"\n\n"+
				"The default behaviour is to compare the"+
				" standard collection of snippets with those"+
				" in the given target directory."),
	)
}
