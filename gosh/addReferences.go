package main

import "github.com/nickwells/param.mod/v5/param"

func addReferences(ps *param.PSet) error {
	ps.AddReference("findCmpRm",
		"This program can be used to verify any changes made when"+
			" in-place editing (see '-"+paramNameInPlaceEdit+"'). It"+
			" will find all the files with a '"+origExt+"' extension"+
			" and give you the chance to compare them with the"+
			" updated version and then to delete the saved copy or to"+
			" revert the file to the original content"+
			"\n\n"+
			"To get this program:"+
			"\n\n"+
			"go install github.com/nickwells/utilities/findCmpRm@latest")

	ps.AddReference("gosh.snippet",
		"This program can be used to install the standard snippets,"+
			" to compare snippets between one directory and another"+
			" or to install a new directory of snippets"+
			"\n\n"+
			"To get this program:"+
			"\n\n"+
			"go install github.com/nickwells/utilities/gosh.snippet@latest")

	return nil
}
