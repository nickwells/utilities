package main

import "github.com/nickwells/param.mod/v6/param"

// addRefs will add the references to the standard help message
func addRefs(ps *param.PSet) error {
	ps.AddReference(`github.com/nickwells/testhelper.mod/v2/testhelper`,
		"The testhelper package provides some useful functions for"+
			" testing Go code. One of the features it offers is to"+
			" compare output against a 'golden' file. It also supports"+
			" generating goldenfiles and if a previous file of the same"+
			" name already exists it will replace it but keep the"+
			" original in a file with a suffix of '.orig'. This command"+
			" will help you to review any changes and tidy up afterwards.")

	ps.AddReference(`github.com/nickwells/utilities/gosh`,
		"The gosh program has a feature which simplifies editing files in"+
			" place. Copies of the files prior to editing are kept in"+
			" files with the original name plus a suffix of '.orig'. This"+
			" command will help you to review any changes and tidy up"+
			" afterwards.")

	return nil
}
