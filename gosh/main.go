// gosh
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v4/param"
	"github.com/nickwells/param.mod/v4/param/paramset"
)

// Created: Wed Sep  4 09:58:54 2019

const goshCommentIntro = " gosh : "

func main() {
	ps := paramset.NewOrDie(
		addParams,
		addExamples,
		param.SetProgramDescription(
			"This will run Go code in an implicit main function."+
				" It is also possible to run the code in a"+
				" loop that will read lines from the standard input"+
				" and, optionally, to split these lines into fields"+
				" on chosen boundaries. Alternatively you can run"+
				" the code as a simple webserver."+
				"\n\nIt is also possible to preserve the"+
				" temporary file created for subsequent editing."),
	)

	ps.Parse()

	goFile := makeGoFile()

	execCmd(goFile)

	clearFile(goFile)
}

// clearFile removes the created program file if the clearFileOnSuccess flag
// is set (the default)
func clearFile(filename string) {
	if dontClearFile {
		return
	}

	err := os.Remove(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't remove the Go file:", err)
		fmt.Fprintln(os.Stderr, "filename:", filename)
		os.Exit(1)
	}
}

// formatFile runs the formatter over the populated program file
func formatFile(filename string) {
	if !formatterSet {
		if _, err := exec.LookPath(goImports); err == nil {
			formatter = goImports
		}
	}

	formatterArgs = append(formatterArgs, filename)
	cmd := exec.Command(formatter, formatterArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't format the Go file:", err)
		fmt.Fprintln(os.Stderr, "filename:", filename)
		os.Exit(1)
	}
}

// openGoFile creates the file to hold the program, opens it and returns the
// open file. If no filename is given then a temporary file is created.
func openGoFile() *os.File {
	if filename == "" {
		f, err := ioutil.TempFile("", "gosh-*.go")
		if err != nil {
			fmt.Fprintln(os.Stderr,
				"Couldn't create the temporary Go file:", err)
			os.Exit(1)
		}
		return f
	}

	return gogen.MakeFileOrDie(filename)
}

// execCmd will call go run to execute the constructed program
func execCmd(filename string) {
	if dontRun {
		return
	}

	cmd := exec.Command("go", "run", filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't run the Go file:", err)
		fmt.Fprintln(os.Stderr, "filename:", filename)
		os.Exit(1)
	}
}

// writeGoFileGlobals writes any globals into the Go file
func writeGoFileGlobals(f io.Writer) {
	for _, s := range globalsList {
		fmt.Fprintln(f)
		fmt.Fprintln(f, s)
	}
}

// writeGoFilePreScript writes the statements that come before the main
// script into the Go file
func writeGoFilePreScript(f io.Writer) {
	for _, s := range preScript {
		fmt.Fprintln(f, "\t"+s)
	}
	if len(preScript) > 0 {
		fmt.Fprintln(f)
	}
}

// writeGoFilePostScript writes the statements that come after the main
// script into the Go file
func writeGoFilePostScript(f io.Writer) {
	if len(postScript) <= 0 {
		return
	}

	fmt.Fprintln(f)
	for _, s := range postScript {
		fmt.Fprintln(f, "\t"+s)
	}
}

// writeGoFileImports writes the import statements into the Go file
func writeGoFileImports(f io.Writer) {
	if runInReadLoop {
		imports = append([]string{"os", "bufio"}, imports...)
		if splitLine {
			imports = append([]string{"regexp"}, imports...)
		}
	}
	if runAsWebserver {
		imports = append([]string{"net/http"}, imports...)
	}

	for _, imp := range imports {
		fmt.Fprintf(f, "import %q\n", imp)
	}
}

// writeGoFileReadLoopOpen writes the opening statements of the readloop
// (if any) into the Go file
func writeGoFileReadLoopOpen(f io.Writer) {
	if !runInReadLoop {
		return
	}

	goshTag := comment("readLoop")

	if splitLine {
		fmt.Fprintf(f, "\tlineSplitter := regexp.MustCompile(%q)%s\n",
			splitPattern, goshTag)
	}
	fmt.Fprintln(f, "\tline := bufio.NewScanner(os.Stdin)"+goshTag)
	fmt.Fprintln(f, "\tfor line.Scan() {"+goshTag)

	if splitLine {
		fmt.Fprintln(f, "\t\tf := lineSplitter.Split(line.Text(), -1)"+goshTag)
	}
}

// writeGoFileScript writes the script statements into the Go file
func writeGoFileScript(f io.Writer) {
	scriptIndent := "\t"
	if runInReadLoop {
		scriptIndent = "\t\t"
	}
	for _, s := range script {
		fmt.Fprintln(f, scriptIndent+s)
	}
}

// writeGoFileReadLoopClose writes the closing statements of the readloop
// (if any) into the Go file
func writeGoFileReadLoopClose(f io.Writer) {
	if !runInReadLoop {
		return
	}

	goshTag := comment("readLoop")

	fmt.Fprintln(f, "\t}"+goshTag)
	fmt.Fprintln(f, "\tif err := line.Err(); err != nil {"+goshTag)
	fmt.Fprintln(f, "\t\tfmt.Fprintln(os.Stderr,"+goshTag)
	fmt.Fprintln(f, "\t\t\t\"reading standard input:\", err)"+goshTag)
	fmt.Fprintln(f, "\t}"+goshTag)
}

// writeGoFileWebserverInit writes the webserver boilerplate code
// (if any) into the Go file
func writeGoFileWebserverInit(f io.Writer) {
	if !runAsWebserver {
		return
	}

	goshTag := comment("webServer")

	fmt.Fprintln(f, "\thttp.HandleFunc(\"/\", goshHandler)"+goshTag)
	fmt.Fprintf(f, "\thttp.ListenAndServe(\":%d\", nil)%s\n", httpPort, goshTag)
}

// writeGoFileWebserverHandler writes the webserver handler function
// (if any) into the Go file
func writeGoFileWebserverHandler(f io.Writer) {
	if !runAsWebserver {
		return
	}

	goshTag := comment("webServer")

	fmt.Fprintln(f,
		"func goshHandler(w http.ResponseWriter, r *http.Request) {"+goshTag)
	writeGoFileScript(f)
	fmt.Fprintln(f, "}"+goshTag)
}

// makeGoFile creates the Go file and then writes the code into the it, then
// it formats the generated code. It returns the name of the generated file.
func makeGoFile() string {
	f := openGoFile()
	defer f.Close()

	goFile := f.Name()
	if showFilename {
		fmt.Println(goFile)
	}

	writeGoFile(f)

	formatFile(goFile)

	return goFile
}

// writeGoFile writes the contents of the Go file
func writeGoFile(f io.Writer) {
	goshTag := comment("goshFrame")

	fmt.Fprintln(f, "package main"+goshTag)

	writeGoshComment(f)
	writeGoFileImports(f)
	writeGoFileGlobals(f)

	fmt.Fprintln(f)
	fmt.Fprintln(f, "func main() {"+goshTag)

	writeGoFilePreScript(f)

	if runAsWebserver {
		writeGoFileWebserverInit(f)
	} else {
		writeGoFileReadLoopOpen(f)
		writeGoFileScript(f)
		writeGoFileReadLoopClose(f)
	}

	writeGoFilePostScript(f)

	fmt.Fprintln(f, "}"+goshTag)

	if runAsWebserver {
		writeGoFileWebserverHandler(f)
	}
}

// writeGoshComment writes the introductory comment
func writeGoshComment(f io.Writer) {
	fmt.Fprintln(f,
		"// ==================================================================")
	fmt.Fprintln(f, "// This code was generated by gosh.")
	fmt.Fprintln(f, "// go get github.com/nickwells/utilities/gosh")
	fmt.Fprintln(f, "//")
	fmt.Fprintln(f,
		"// All code generated by gosh (apart from this) ends with a comment.")
	fmt.Fprintln(f, "// The comment will start with: '"+goshCommentIntro+"'.")
	fmt.Fprintln(f, "// User provided code will be as given.")
	fmt.Fprintln(f,
		"// ==================================================================")
	fmt.Fprintln(f)
}

// comment returns the standard comment string explaining why the line is
// in the generated code
func comment(text string) string {
	return "\t//" + goshCommentIntro + text
}
