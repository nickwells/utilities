package main

import "github.com/nickwells/param.mod/v6/param"

func addRefs(ps *param.PSet) error {
	ps.AddReference("findCmpRm",
		"A program to find files with a given suffix and compare"+
			" them with corresponding files without the suffix."+
			" This can be useful to compare the installed snippets"+
			" with differing versions of the same snippet moved"+
			" aside during the installation. It will prompt the"+
			" user after any differences have been shown to remove"+
			" the copy of the file. It is thus useful for cleaning"+
			" up the snippet directory after installation."+
			"\n\n"+
			"This can be found in the same repository as gosh and"+
			" this command. You can install this with 'go install'"+
			" in the same way as these commands.")

	return nil
}
