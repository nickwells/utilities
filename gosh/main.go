// gosh
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/nickwells/param.mod/v3/param"
	"github.com/nickwells/param.mod/v3/param/paramset"
)

// Created: Wed Sep  4 09:58:54 2019

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

// populateGoFileFunctions writes any functions into the go file
func populateGoFileFunctions(f *os.File) {
	for _, s := range funcList {
		fmt.Fprintln(f)
		if !(strings.HasPrefix(s, "func ") ||
			strings.HasPrefix(s, "func\t")) {
			fmt.Fprint(f, "func ")
		}
		fmt.Fprintln(f, s)
	}
}

// populateGoFileBefore writes the statements that come before the main
// script into the go file
func populateGoFileBefore(f *os.File) {
	for _, s := range beginScript {
		fmt.Fprintln(f, "\t"+s)
	}
	if len(beginScript) > 0 {
		fmt.Fprintln(f)
	}
}

// populateGoFileAfter writes the statements that come after the main
// script into the go file
func populateGoFileAfter(f *os.File) {
	if len(endScript) <= 0 {
		return
	}

	fmt.Fprintln(f)
	for _, s := range endScript {
		fmt.Fprintln(f, "\t"+s)
	}
}

// populateGoFileImports writes the import statements into the go file
func populateGoFileImports(f *os.File) {
	if runInReadLoop {
		imports = append([]string{"os", "bufio"}, imports...)
		if splitLine {
			imports = append([]string{"regexp"}, imports...)
		}
	}
	for _, imp := range imports {
		fmt.Fprintf(f, "import %q\n", imp)
	}
}

// populateGoFileReadLoopOpen writes the opening statements of the readloop
// (if any) into the go file
func populateGoFileReadLoopOpen(f *os.File) {
	if !runInReadLoop {
		return
	}

	if splitLine {
		fmt.Fprintln(f, "\tlineSplitter := regexp.MustCompile(`\\s+`)"+
			comment("splitLine"))
	}
	fmt.Fprintln(f, "\tline := bufio.NewScanner(os.Stdin)"+comment("readLoop"))
	fmt.Fprintln(f, "\tfor line.Scan() {"+comment("readLoop"))

	if splitLine {
		fmt.Fprintln(f, "\t\tf := lineSplitter.Split(line.Text(), -1)"+
			comment("splitLine"))
	}
}

// populateGoFileScript writes the script statements into the go file
func populateGoFileScript(f *os.File) {
	scriptIndent := "\t"
	if runInReadLoop {
		scriptIndent = "\t\t"
	}
	for _, s := range script {
		fmt.Fprintln(f, scriptIndent+s)
	}
}

// populateGoFileReadLoopClose writes the closing statements of the readloop
// (if any) into the go file
func populateGoFileReadLoopClose(f *os.File) {
	if !runInReadLoop {
		return
	}
	fmt.Fprintln(f, "\t}"+comment("readLoop"))
	fmt.Fprintln(f, "\tif err := line.Err(); err != nil {"+comment("readLoop"))
	fmt.Fprintln(f,
		"\t\tfmt.Fprintln(os.Stderr, \"reading standard input:\", err)"+
			comment("readLoop"))
	fmt.Fprintln(f, "\t}"+comment("readLoop"))
}

// populateGoFile writes the code into the go file
func populateGoFile(f *os.File) {
	fmt.Fprintln(f, "package main")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "/*")
	fmt.Fprintln(f, " code generated by gosh")
	fmt.Fprintln(f, "*/")

	populateGoFileImports(f)
	populateGoFileFunctions(f)

	fmt.Fprintln(f)
	fmt.Fprintln(f, "func main() {")

	populateGoFileBefore(f)

	populateGoFileReadLoopOpen(f)
	populateGoFileScript(f)
	populateGoFileReadLoopClose(f)

	populateGoFileAfter(f)

	fmt.Fprintln(f, "}")
}

// comment returns the standard comment string explaining why the line is
// in the generated code
func comment(reason string) string {
	return "\t// AutoGen : " + reason
}
