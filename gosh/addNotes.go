package main

import (
	"github.com/nickwells/param.mod/v5/param"
)

// addNotes will add any notes to the param PSet
func addNotes(ps *param.PSet) error {
	ps.AddNote("In-place Editing",
		"The files given for editing are checked to make sure that"+
			" they all exist, that there is no pre-existing file with"+
			" the same name plus the '"+origExt+"' extension and that"+
			" there are no duplicate filenames. If any of these checks"+
			" fails the program aborts with an error message."+
			"\n\n"+
			"if \"-"+paramNameInPlaceEdit+"\" is given then some"+
			" filenames must be supplied"+
			" (after \""+ps.TerminalParam()+"\")."+
			"\n\n"+
			" After you have run this edit program you could use the"+
			" findCmpRm program to check that the changes were as"+
			" expected")

	ps.AddNote("A list of filenames",
		"A list of filenames to be processed can be given"+
			" (after "+ps.TerminalParam()+"). Each filename will be edited"+
			" to be an absolute path if it is not already; the current"+
			" directory will be added at the start of the path."+
			" If any files are given then some parameter for"+
			" reading them should be given. See the parameters in"+
			" group: '"+paramGroupNameReadloop+"'.")

	ps.AddNote("Gosh Variables",
		"gosh will create some variables as it builds the program."+
			" These are all listed below. You should avoid creating"+
			" any variables yourself with the same names and you"+
			" should not change the values of any of these."+
			"\n\n"+makeKnownVarList())

	return nil
}
