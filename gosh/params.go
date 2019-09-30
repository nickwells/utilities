// gosh
package main

import (
	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/param.mod/v3/param"
	"github.com/nickwells/param.mod/v3/param/paction"
	"github.com/nickwells/param.mod/v3/param/psetter"
)

var script []string
var beginScript []string
var endScript []string
var funcList []string

var imports []string
var showFilename bool
var clearFileOnSuccess = true
var runInReadLoop bool
var splitLine bool

var filename string

var formatter = "gofmt"
var formatterSet bool
var formatterArgs = []string{"-w"}

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	ps.Add("exec", psetter.StrListAppender{Value: &script},
		"follow this with the go code to be run."+
			" This will be placed inside a main() function",
		param.AltName("e"),
		param.Attrs(param.MustBeSet),
	)

	ps.Add("begin", psetter.StrListAppender{Value: &beginScript},
		"follow this with go code to be run at the beginning."+
			" This will be placed inside a main() function before"+
			" the code given for the exec parameter and also"+
			" before any read-loop",
		param.AltName("before"),
		param.AltName("b"),
	)

	ps.Add("end", psetter.StrListAppender{Value: &endScript},
		"follow this with go code to be run at the end."+
			" This will be placed inside a main() function after"+
			" the code given for the exec parameter and most"+
			" importantly outside any read-loop",
		param.AltName("after"),
		param.AltName("a"))

	ps.Add("func", psetter.StrListAppender{Value: &funcList},
		"follow this with go code defining a function",
		param.AltName("function"))

	ps.Add("imports", psetter.StrListAppender{Value: &imports},
		"provide any explicit imports",
		param.AltName("I"))

	ps.Add("show-filename", psetter.Bool{Value: &showFilename},
		"show the filename where the program has been constructed."+
			" This will also prevent the file from being cleared"+
			" after execution has successfully completed, the"+
			" assumption being that if you want to know the"+
			" filename you will also want to examine its contents.",
		param.PostAction(paction.SetBool(&clearFileOnSuccess, false)),
	)

	ps.Add("set-filename",
		psetter.String{
			Value: &filename,
			Checks: []check.String{
				check.StringHasSuffix(".go"),
				check.StringNot(
					check.StringHasSuffix("_test.go"),
					"a string ending with _test.go"+
						" - the file must not be a test file"),
			},
		},
		"set the filename where the program will be constructed."+
			" This will also prevent the file from being cleared"+
			" after execution has successfully completed, the"+
			" assumption being that if you have set the"+
			" filename you will want to preserve its contents.",
		param.PostAction(paction.SetBool(&clearFileOnSuccess, false)),
	)

	ps.Add("run-in-readloop", psetter.Bool{Value: &runInReadLoop},
		"have the script code run within a loop that reads from stdin"+
			" one a line at a time. The value of each line can be"+
			" accessed by calling 'line.Text()'. Note that any"+
			" newline will have been removed and will need to be added"+
			" back if you want to print the line",
		param.AltName("n"),
	)

	ps.Add("split-line", psetter.Bool{Value: &splitLine},
		"split the lines into fields around runs of whitespace"+
			" characters. The fields will be available in a slice"+
			" of strings called 'f'. Setting this will also force"+
			" the script to be run in the loop reading from stdin",
		param.AltName("s"),
		param.PostAction(paction.SetBool(&runInReadLoop, true)),
	)

	ps.Add("formatter", psetter.String{Value: &formatter},
		"the name of the formatter command to run",
		param.PostAction(paction.SetBool(&formatterSet, true)),
	)

	ps.Add("formatter-args", psetter.StrList{Value: &formatterArgs},
		"the arguments to pass to the formatter command. Note that the"+
			" final argument will always be the name of the generated program")

	return nil
}
