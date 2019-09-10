// gosh
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/param.mod/v3/param"
	"github.com/nickwells/param.mod/v3/param/paction"
	"github.com/nickwells/param.mod/v3/param/paramset"
	"github.com/nickwells/param.mod/v3/param/psetter"
)

// Created: Wed Sep  4 09:58:54 2019
var script string
var beginScript string
var endScript string

var imports []string
var showFilename bool
var clearFileOnSuccess = true
var runInReadLoop bool
var splitLine bool

var filename string

var formatter = "gofmt"
var formatterSet bool
var formatterArgs = []string{"-w"}

func main() {
	ps := paramset.NewOrDie(addParams,
		param.SetProgramDescription(
			"this will run go code in an implicit main function."+
				" Note that it is also possible to run the code in a"+
				" loop that will read lines from the standard input"+
				" and to split these lines into fields on whitespace"+
				" boundaries. It is also possible to preserve the"+
				" temporary file created for subsequent editing."),
	)

	ps.Parse()

	f := makeCmdFile(filename)
	filename = f.Name()
	if showFilename {
		fmt.Println(filename)
	}

	populateGoFile(f)
	f.Close()
	runFormatter(filename)

	runCmd(filename)

	clearFile(filename)
}

// clearFile removes the created program file if the clearFileOnSuccess flag
// is set (the default)
func clearFile(filename string) {
	if clearFileOnSuccess {
		err := os.Remove(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't remove the go file:", err)
			fmt.Fprintln(os.Stderr, "filename:", filename)
			os.Exit(1)
		}
	}
}

// runFormatter runs the formatter over the populated program file
func runFormatter(filename string) {
	if !formatterSet {
		if _, err := exec.LookPath("goimports"); err == nil {
			formatter = "goimports"
		}
	}

	formatterArgs = append(formatterArgs, filename)
	cmd := exec.Command(formatter, formatterArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error formatting the go file:", err)
		fmt.Fprintln(os.Stderr, "filename:", filename)
		os.Exit(1)
	}
}

// makeCmdFile creates the file to hold the program, opens it and returns the
// open file. If filename is empty then a temporary file is created.
func makeCmdFile(filename string) *os.File {
	if filename == "" {
		f, err := ioutil.TempFile("", "gosh-*.go")
		if err != nil {
			fmt.Fprintln(os.Stderr,
				"Error creating the temporary go file:", err)
			os.Exit(1)
		}
		return f
	}

	f, err := os.Create(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating the go file:", err)
		fmt.Fprintln(os.Stderr, "filename:", filename)
		os.Exit(1)
	}
	return f
}

// runCmd will call go run to execute the constructed program
func runCmd(filename string) {
	cmd := exec.Command("go", "run", filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error running the go file:", err)
		fmt.Fprintln(os.Stderr, "filename:", filename)
		os.Exit(1)
	}
}

// populateGoFile writes the code into the go file
func populateGoFile(f *os.File) {
	fmt.Fprintln(f, "package main")
	fmt.Fprintln(f)

	if runInReadLoop {
		imports = append([]string{"os", "bufio"}, imports...)
		if splitLine {
			imports = append([]string{"regexp"}, imports...)
		}
	}
	for _, imp := range imports {
		fmt.Fprintf(f, "import %q\n", imp)
	}

	fmt.Fprintln(f)
	fmt.Fprintln(f, "func main() {")

	fmt.Fprintln(f, "\t"+beginScript)

	if runInReadLoop {
		if splitLine {
			fmt.Fprintln(f, "\tlineSplitter := regexp.MustCompile(`\\s+`)")
		}
		fmt.Fprintln(f, "\tline := bufio.NewScanner(os.Stdin)")
		fmt.Fprintln(f, "\tfor line.Scan() {")

		if splitLine {
			fmt.Fprintln(f, "\t\tf := lineSplitter.Split(line.Text(), -1)")
		}
	}
	fmt.Fprintln(f, script)
	if runInReadLoop {
		fmt.Fprintln(f, "\t}")
		fmt.Fprintln(f, "\tif err := line.Err(); err != nil {")
		fmt.Fprintln(f,
			"\t\tfmt.Fprintln(os.Stderr, \"reading standard input:\", err)")
		fmt.Fprintln(f, "\t}")
	}

	fmt.Fprintln(f, "\t"+endScript)

	fmt.Fprintln(f, "}")
}

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	ps.Add("exec", psetter.String{Value: &script},
		"follow this with the go code to be run."+
			" This will be placed inside a main() function",
		param.AltName("e"),
		param.Attrs(param.MustBeSet),
	)

	ps.Add("begin", psetter.String{Value: &beginScript},
		"follow this with go code to be run at the beginning."+
			" This will be placed inside a main() function before"+
			" the code given for the exec parameter and also"+
			" before any read-loop",
		param.AltName("b"),
	)

	ps.Add("end", psetter.String{Value: &endScript},
		"follow this with go code to be run at the end."+
			" This will be placed inside a main() function after"+
			" the code given for the exec parameter and most"+
			" importantly outside any read-loop")

	ps.Add("imports", psetter.StrList{Value: &imports},
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
