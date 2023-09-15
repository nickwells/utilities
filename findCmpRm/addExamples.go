package main

import "github.com/nickwells/param.mod/v6/param"

// addExamples will add examples to the program help message
func addExamples(ps *param.PSet) error {
	ps.AddExample(
		"findCmpRm -diff sdiff -diff-args '-w,170'",
		"This will use sdiff to compare the files rather than the"+
			" default program ("+dfltDiffCmd+")")
	ps.AddExample(
		"findCmpRm -diff-args '-W,170,-y,--color=always' -less-args=-R",
		"This will use show the differences in two columns, side by side,"+
			" with differences highlighted in colour and with less taking"+
			" the colour output and displaying it."+
			"\n\n"+
			"You might want to put these parameters in the configuration"+
			" file so that you don't have to repeatedly set them on each"+
			" use of the program.")
	ps.AddExample(
		"findCmpRm -d testdata",
		"This will search the testdata directory and any"+
			" subdirectories for the files to process."+
			"\n\n"+
			"It searches for files with names"+
			" ending with '"+dfltExtension+"'.")
	ps.AddExample(
		"findCmpRm -d testdata -dont-recurse",
		"This will search the testdata directory but not any"+
			" subdirectories for the files to process.")
	ps.AddExample(
		"findCmpRm -d testdata -extension .old",
		"This will search the testdata directory for files"+
			" with names ending with '.old'.")

	return nil
}
