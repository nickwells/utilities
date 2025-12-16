package main

import "github.com/nickwells/param.mod/v6/param"

const (
	progName   = "statfs"
	exampleDir = "/home/me"
)

// addExamples returns a function that will add some examples of how to use
// the command to the passed ParamSet
func addExamples() param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.AddExample(progName,
			"This will print the directory being tested (the current"+
				" directory by default) and the available space in bytes.")
		ps.AddExample(
			progName+
				" -show "+
				avSpStr+
				" -no-label -units GB -- "+
				exampleDir,
			"This will print just the available space (in Gigabytes)"+
				" without any label. The filesystem to be reported on is"+
				" the one on which "+exampleDir+" is found."+
				"\n\n"+
				"This form is useful if you want to use the result in a"+
				" shell script since you don't need to pass the output"+
				" to any other programs to strip any labels.")

		return nil
	}
}
