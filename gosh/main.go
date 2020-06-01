// gosh
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/timer.mod/timer"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Wed Sep  4 09:58:54 2019

// constantWidthStr returns the string formatter into a right-justified
// string of a consistent length
func constantWidthStr(s string) string {
	return fmt.Sprintf("%18.18s", s)
}

var blank = constantWidthStr("")

// VerboseTimer used in conjunction with the timer and verbose packages this
// will print out how long a function took to run
type VerboseTimer struct{}

// Act will perform the action for the timer - it prints out the tag and the
// duration in milliseconds if the program is in verbose mode
func (VerboseTimer) Act(tag string, d time.Duration) {
	if !verbose.IsOn() {
		return
	}
	fmt.Printf("%s: %12.3f msecs\n", tag, float64(d/time.Microsecond)/1000.0)
	fmt.Printf("%s: ------------\n", blank)
}

var verboseTimer VerboseTimer

const goshCommentIntro = " gosh : "

var cleanupPath string

func main() {
	defer timer.Start(constantWidthStr("main"), verboseTimer)()
	ps := paramset.NewOrDie(
		verbose.AddParams,
		addParams,
		addExamples,
		param.SetProgramDescription(
			"This will run Go code in an implicit main function. It is also"+
				" possible to run the code in a loop that will read lines from"+
				" the standard input and, optionally, to split these lines"+
				" into fields on chosen boundaries. Alternatively you can"+
				" run the code as a simple webserver."+
				"\n\n"+
				"It is also possible to preserve the temporary file created"+
				" for subsequent editing."+
				"\n\n"+
				"Note that by default the program will be generated in a"+
				" temporary directory and executed from there so that any"+
				" paths used should be given in full rather than relative"+
				" to your current directory"),
	)

	ps.Parse()

	goFile := buildGoProgram()

	runGoFile(goFile)

	clearFiles()
}

// clearFiles removes the created program file, any module files and the
// containing directory unless the dontClearFile flag is set
func clearFiles() {
	intro := constantWidthStr("clearFiles")
	defer timer.Start(intro, verboseTimer)()

	if dontClearFile {
		verbose.Print(intro, ": Skipping\n")
		return
	}

	verbose.Print(intro, ": Cleaning-up the Go files\n")

	err := os.RemoveAll(cleanupPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't remove the Go files:", err)
		fmt.Fprintln(os.Stderr, "\t:", cleanupPath)
		os.Exit(1)
	}
}

// formatFile runs the formatter over the populated program file
func formatFile(filename string) {
	intro := constantWidthStr("formatFile")
	defer timer.Start(intro, verboseTimer)()

	verbose.Print(intro, ": Formatting the Go file\n")
	if !formatterSet {
		if _, err := exec.LookPath(goImports); err == nil {
			formatter = goImports
			verbose.Print(intro, ":\tUsing ", goImports, "\n")
		}
	}

	formatterArgs = append(formatterArgs, filename)
	verbose.Print(intro, ":\tCommand: ",
		formatter, " ",
		strings.Join(formatterArgs, " "),
		"\n")
	out, err := exec.Command(formatter, formatterArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't format the Go file:", err)
		fmt.Fprintln(os.Stderr, "\tfilename:", filename)
		fmt.Fprintln(os.Stderr, string(out))
		os.Exit(1)
	}
}

// createGoFiles creates the file to hold the program, opens it and returns the
// open file. If no filename is given then a temporary directory is created,
// the program changes directory to there, creates the file and initialises
// any module files if necessary.
func createGoFiles() *os.File {
	intro := constantWidthStr("createGoFiles")
	defer timer.Start(intro, verboseTimer)()

	verbose.Print(intro, ": Creating the Go files\n")

	if filename != "" {
		cleanupPath = filename
		return gogen.MakeFileOrDie(filename)
	}

	verbose.Print(intro, ":\tCreating the temporary directory\n")
	d, err := ioutil.TempDir("", "gosh-*.d")
	if err != nil {
		fmt.Fprintln(os.Stderr,
			"Couldn't create the temporary directory:", err)
		os.Exit(1)
	}
	cleanupPath = d

	verbose.Print(intro, ":\tChdir'ing into ", d, "\n")
	err = os.Chdir(d)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Couldn't chdir to the temporary directory (%s): %v", d, err)
		os.Exit(1)
	}

	fName := filepath.Join(d, "gosh.go")
	verbose.Print(intro, ":\tCreating the Go file: ", fName, "\n")
	f, err := os.Create(fName)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Couldn't create the Go file (%s): %v", fName, err)
		os.Exit(1)
	}

	if os.Getenv("GO111MODULE") != "off" {
		verbose.Print(intro,
			":\tRunning 'go mod init gosh' (creates the module files)\n")
		execGoCmd(TCmdIONone, "mod", "init", "gosh")
	}

	return f
}

// runGoFile will call go run to execute the constructed program
func runGoFile(filename string) {
	intro := constantWidthStr("runGoFile")
	defer timer.Start(intro, verboseTimer)()

	if dontRun {
		verbose.Print(intro, ": Skipping\n")
		return
	}

	verbose.Print(intro, ": Running the Go file\n")
	execGoCmd(TCmdIOShow, "run", filename)
}

// execGoCmd will exec the go program with the supplied arguments. If it
// detects an error it will report it and exit
func execGoCmd(ioMode CmdIO, args ...string) {
	cmd := exec.Command("go", args...)
	if ioMode == TCmdIOShow {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()
	if err != nil {
		cmdLine := []string{"go"}
		cmdLine = append(cmdLine, args...)
		cmdLineStr := strings.Join(cmdLine, " ")
		fmt.Fprintln(os.Stderr, "Couldn't exec the go command")
		fmt.Fprintln(os.Stderr, "\t"+cmdLineStr)
		fmt.Fprintln(os.Stderr, "\tError:", err)
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

// buildGoProgram creates the Go file and then writes the code into the it, then
// it formats the generated code. It returns the name of the generated file.
func buildGoProgram() string {
	intro := constantWidthStr("buildGoProgram")
	defer timer.Start(intro, verboseTimer)()

	verbose.Print(intro, ": Building the program\n")

	f := createGoFiles()
	defer f.Close()

	goFile := f.Name()
	if showFilename {
		fmt.Println(goFile)
	}
	verbose.Print(intro, ":\tGo file name: ", goFile, "\n")

	writeGoFile(f)

	formatFile(goFile)

	tidyModule()

	return goFile
}

// tidyModule runs go mod tidy after the file is fully constructed to
// populate the go.mod and go.sum files
func tidyModule() {
	intro := constantWidthStr("tidyModule")
	defer timer.Start(intro, verboseTimer)()

	if os.Getenv("GO111MODULE") == "off" || filename != "" {
		verbose.Print(intro, ":\tSkipping\n")
		return
	}
	verbose.Print(intro, ":\tRunning 'go mod tidy' (populates go.mod)\n")
	execGoCmd(TCmdIONone, "mod", "tidy")
}

// writeGoFile writes the contents of the Go file
func writeGoFile(f io.Writer) {
	intro := constantWidthStr("writeGoFile")
	defer timer.Start(intro, verboseTimer)()

	verbose.Print(intro, ": Writing the contents of the Go file\n")

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
