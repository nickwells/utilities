package main

import (
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/snippet.mod/snippet"
)

const (
	noteInPlaceEdit      = "Gosh - in-place editing"
	noteFilenames        = "Gosh - filenames"
	noteVars             = "Gosh - variables"
	noteSnippets         = "Gosh - snippets"
	noteSnippetsComments = "Gosh - snippet comments"
	noteSnippetsDirs     = "Gosh - snippet directories"
	noteCodeSections     = "Gosh - code sections"
	noteShebangScripts   = "Gosh - shebang scripts"
)

// addNotes will add any notes to the param PSet
func addNotes(g *Gosh) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		ps.AddNote(noteInPlaceEdit,
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

		ps.AddNote(noteFilenames,
			"A list of filenames to be processed can be given"+
				" (after "+ps.TerminalParam()+")."+
				" Each filename will be edited to be an absolute path"+
				" if it is not already; the current"+
				" directory will be added at the start of the path."+
				" If any files are given then some parameter for"+
				" reading them should be given. See the parameters in"+
				" group: '"+paramGroupNameReadloop+"'."+
				"\n\n"+
				" Note that it is an error if the same name appears twice.")

		ps.AddNote(noteVars,
			"gosh will create some variables as it builds the program."+
				" These are all listed below. You should avoid creating"+
				" any variables yourself with the same names and you"+
				" should not change the values of any of these."+
				" Note that they all start with a single underscore so"+
				" provided you start all your variable names with a"+
				" letter (as usual) you will not clash."+
				"\n\n"+makeKnownVarList())

		ps.AddNote(noteSnippets,
			"You can introduce pre-defined blocks of code (called snippets)"+
				" into your script. gosh will search through a list of"+
				" directories for a file with the snippet name and insert"+
				" that into your script."+
				" See the note"+
				"\n"+
				"'"+noteSnippetsDirs+"'"+
				"\n"+
				" for a list of the default directories that will be"+
				" searched for snippets."+
				" A filename with a full path can also be given."+
				" Any inserted code is prefixed with a comment showing"+
				" which file it came from to help with debugging."+
				"\n\n"+
				"A suggested standard is to name any variables that you"+
				" declare in a snippet file with a leading double"+
				" underscore. This will ensure that the names neither"+
				" clash with any gosh-declared variables nor any variables"+
				" declared by the user."+
				"\n\n"+
				"It is also suggested that sets of snippets which must be"+
				" used together should be grouped into their own"+
				" sub-directory in the snippets directory and named with"+
				" leading digits to indicate the order that they must be"+
				" applied.")

		ps.AddNote(noteSnippetsComments,
			"Any lines in a snippet file starting with"+
				" '// "+snippet.CommentStr+"' are"+
				" not copied but are treated as comments on the snippet"+
				" itself."+
				"\n\n"+
				"A snippet comment can have additional meaning."+
				" If it is followed by one of these values then the"+
				" rest of the line is used as described:"+
				"\n\n"+
				"- '"+snippet.NoteStr+"'"+
				"\n"+
				"The following text is reported as documentation"+
				" when the snippets are listed."+
				"\n\n"+
				"- '"+snippet.ImportStr+"'"+
				"\n"+
				"The following text is added to the list of"+
				" import statements. Note that gosh will format the"+
				" program it generates with "+goImportsFormatter+
				" (if available) which should populate the import"+
				" statement automatically but adding an import comment"+
				" can ensure that the snippet works even if "+
				goImportsFormatter+" is not available and can avoid"+
				" any mismatch where the formatter finds the wrong package."+
				"\n\n"+
				"- '"+snippet.ExpectStr+"'"+
				"\n"+
				"Records another snippet that"+
				" is expected to be given if this snippet is used. This"+
				" allows a chain of snippets to check that all necessary"+
				" parts have been used and help to ensure correct usage"+
				" of the snippet chain."+
				"\n"+
				"This is enforced by the Gosh command."+
				"\n\n"+
				"- '"+snippet.AfterStr+"'"+
				"\n"+
				"Records another snippet that"+
				" is expected to appear before this snippet is used. This"+
				" allows a chain of snippets to check that the"+
				" parts have been used in the right order."+
				"\n"+
				"This is enforced by the Gosh command."+
				"\n\n"+
				"- '"+snippet.TagStr+"'"+
				"\n"+
				"Records a documentary tag."+
				" The text will be split on a ':' and the"+
				" first part will be used as a tag with the remainder"+
				" used as a value. These are then reported when the"+
				" snippets are listed. These have no semantic"+
				" meaning and are purely for documentary purposes."+
				" It allows you to give some structure to your snippet"+
				" documentation."+
				"\n"+
				"Suggested tag names might be"+
				"\n"+
				"   'Author'   to document the snippet author"+
				"\n"+
				"   'Env'      for an environment variable the snippet uses"+
				"\n"+
				"   'Declares' for a variable that it declares.")

		ps.AddNote(noteSnippetsDirs,
			"By default snippets will be searched for in standard"+
				" directories. To see the names of the snippet directories"+
				" use the parameter '-"+paramNameSnippetListDir+"'."+
				"\n\n"+
				"More directories can be added to the list using the"+
				" parameter '-"+paramNameSnippetDir+"' which will add new"+
				" directories to the start of the list."+
				"\n\n"+
				"The directories are searched in the order given above and the"+
				" first file matching the name of the snippet will be used."+
				" Any extra directories, since they are added at the start of"+
				" the list, will be searched before the default ones.")

		ps.AddNote(noteCodeSections,
			"The program that gosh will generate is split up into several"+
				" sections and you can add code to these sections."+
				" The sections are:"+
				"\n\n"+
				globalSect+" - code at global scope, outside of main\n"+
				beforeSect+" - code at the start of the program\n"+
				execSect+"   - code, possibly in a readloop or a web handler\n"+
				afterSect+"  - code at the end of the program")

		ps.AddNote(noteShebangScripts,
			"You can use gosh in shebang scripts (executable files"+
				" starting with '#!'). Follow the '#!'"+
				" with the full pathname of the gosh command and the"+
				" parameter '-exec-file' and gosh will construct your Go"+
				" program from the contents of the rest of the file and"+
				" run it."+
				"\n\n"+
				"The first line should look something like this"+
				"\n\n"+
				"#!/path/to/gosh -exec-file"+
				"\n\n"+
				"The rest of the file is Go code to be run"+
				" inside a main() func."+
				"\n\n"+
				"Any parameters that you pass to the script will be"+
				" interpreted by gosh so you can add extra code to be run.")

		return nil
	}
}
