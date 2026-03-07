package main

import "github.com/nickwells/param.mod/v7/param"

// addNotes adds some notes to the help message
func addNotes(ps *param.PSet) error {
	ps.AddNote(noteNameContentChecks,
		"You can constrain the Go directories this command will find"+
			" by checking that a matching directory has at least one"+
			" file containing certain content."+
			"\n\n"+
			"This feature can by useful, for instance, to find directories"+
			" having files with go:generate comments so you know if you"+
			" need to run 'go generate' in them."+
			"\n\n"+
			"There are some common searches which have dedicated parameters"+
			" for setting them:"+
			" '"+paramNameHavingBuildTag+"' and"+
			" '"+paramNameHavingGoGenerate+"'."+
			" These have all the correct patterns preset and"+
			" it is recommended that you use these."+
			"\n\n"+
			"A content checker has at least a pattern for matching lines"+
			" but it can be extended to only check files matching a"+
			" pattern, to stop matching after a certain pattern is matched"+
			" and to skip otherwise matching lines if they match a pattern"+
			"\n\n"+
			"You can add these additional features using the"+
			" '"+paramNameCheck+"' parameter. ",
		param.NoteSeeParam(
			paramNameHavingBuildTag,
			paramNameHavingGoGenerate,
			paramNameCheck))

	return nil
}
