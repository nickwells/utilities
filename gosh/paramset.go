package main

import (
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/verbose.mod/verbose"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// paramOptFuncs returns the parameter option functions (which add the
// various parameters to the paramset). This is separated out from the
// paramset creation so that it can be used in testing to create several
// distinct paramsets.
func paramOptFuncs(g *gosh, slp *snippetListParams) []param.PSetOptFunc {
	return []param.PSetOptFunc{
		verbose.AddParams,
		verbose.AddTimingParams(g.dbgStack),

		versionparams.AddParams,

		addSnippetListParams(slp),
		addSnippetParams(g),
		addWebParams(g),
		addReadloopParams(g),
		addGoshParams(g),
		addStdinParams(g),
		addParams(g),

		addNotes,
		addExamples,
		addReferences,

		param.SetProgramDescription(
			"This allows you to write lines of Go code and have them run" +
				" for you in a framework that provides the main() func" +
				" and any necessary boilerplate code for some common" +
				" requirements. The resulting program can be preserved" +
				" for subsequent editing." +
				"\n\n" +
				"You can run the code in a loop that will read lines from" +
				" the standard input or from a list of files and," +
				" optionally, split each line into fields." +
				"\n\n" +
				"Alternatively you can quickly generate a simple webserver." +
				"\n\n" +
				"It's faster than opening an editor and writing a Go" +
				" program from scratch especially if there are only a few" +
				" lines of non-boilerplate code. You can also save the" +
				" program that it generates and edit that if the few" +
				" lines become many lines. The workflow would be that you" +
				" use this to make the first few iterations of the" +
				" command and if that is sufficient then just stop. If" +
				" you need to do more then save the file and edit it just" +
				" like a regular Go program."),

		SetGlobalConfigFile,
		SetConfigFile,
	}
}

// makeParamSet creates the parameter set ready for argument parsing
func makeParamSet(g *gosh, slp *snippetListParams) *param.PSet {
	return paramset.NewOrPanic(paramOptFuncs(g, slp)...)
}
