package main

import "github.com/nickwells/param.mod/v6/param"

func addExamples(ps *param.PSet) error {
	const snipDir = "snipDir=$HOME/.config/" +
		"github.com/nickwells/utilities/gosh/snippets"

	ps.AddExample(snipDir+"\ngosh.snippet -target $snipDir",
		"This will compare the standard collection of snippets"+
			" with those in the target directory")

	ps.AddExample(snipDir+"\ngosh.snippet -target $snipDir -install",
		"This will install the standard collection of snippets"+
			" into the target directory")

	return nil
}
