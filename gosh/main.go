package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nickwells/cli.mod/cli/responder"
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/snippet.mod/snippet"
	"github.com/nickwells/timer.mod/timer"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Wed Sep  4 09:58:54 2019

// constantWidthStr returns the string formatted into a right-justified
// string of a consistent length
func constantWidthStr(s string) string {
	return fmt.Sprintf("%24.24s", s)
}

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
	fmt.Printf("%s: ------------\n", strings.Repeat(" ", len(tag)))
}

var verboseTimer VerboseTimer

// makeParamSet creates the parameter set ready for argument parsing
func makeParamSet(g *Gosh, slp *snippetListParams) *param.PSet {
	return paramset.NewOrDie(
		verbose.AddParams,

		addSnippetListParams(slp),
		addSnippetParams(g),
		addWebParams(g),
		addReadloopParams(g),
		addGoshParams(g),
		addParams(g),

		addNotes(g),
		addExamples,
		addReferences,

		param.SetProgramDescription(
			"This allows you to write lines of Go code and have them run"+
				" for you in a framework that provides the main() func"+
				" and any necessary boilerplate code for some common"+
				" requirements. The resulting program can be preserved"+
				" for subsequent editing."+
				"\n\n"+
				"You can run the code in a loop that will read lines from"+
				" the standard input or from a list of files and,"+
				" optionally, split each line into fields."+
				"\n\n"+
				"Alternatively you can quickly generate a simple webserver."+
				"\n\n"+
				"It's faster than opening an editor and writing a Go"+
				" program from scratch especially if there are only a few"+
				" lines of non-boilerplate code. You can also save the"+
				" program that it generates and edit that if the few"+
				" lines become many lines. The workflow would be that you"+
				" use this to make the first few iterations of the"+
				" command and if that is sufficient then just stop. If"+
				" you need to do more then save the file and edit it just"+
				" like a regular Go program."),
	)
}

func main() {
	defer timer.Start(constantWidthStr("main"), verboseTimer)()

	g := NewGosh()
	slp := &snippetListParams{}

	ps := makeParamSet(g, slp)

	_ = SetGlobalConfigFile(ps)
	_ = SetConfigFile(ps)

	ps.Parse()

	listSnippets(g, slp)

	g.snippets.Check(g.errMap)
	g.checkScripts()
	g.reportErrors()

	g.setEditor()
	g.reportErrors()

	g.buildGoProgram()
	g.reportErrors()

	for {
		g.editGoFile()
		g.formatFile()
		g.tidyModule()
		g.runGoFile()

		if !g.queryEditAgain() {
			break
		}
		g.chdirInto(g.goshDir)
	}

	g.clearFiles()
}

// listSnippets checks the snippet list parameters and lists the snippet
// details accordingly. If any listing is done then the program will exit
// after listing is complete.
func listSnippets(g *Gosh, slp *snippetListParams) {
	if !slp.listSnippets && !slp.listDirs {
		return
	}

	if slp.listDirs {
		for _, dir := range g.snippetDirs {
			fmt.Println(dir)
		}
	}

	if slp.listSnippets {
		lc, err := snippet.NewListCfg(os.Stdout, g.snippetDirs, g.errMap,
			snippet.SetConstraints(slp.constraints...),
			snippet.SetParts(slp.parts...),
			snippet.SetTags(slp.tags...),
			snippet.HideIntro(slp.hideIntro))
		if err != nil {
			fmt.Fprintln(os.Stderr,
				"There was a problem configuring the snippet list:")
			fmt.Fprintln(os.Stderr, "\t", err)
			os.Exit(1)
		}
		lc.List()
		g.reportErrors()
	}

	os.Exit(0)
}

// clearFiles removes the created program file, any module files and the
// containing directory unless the dontClearFile flag is set
func (g *Gosh) clearFiles() {
	intro := constantWidthStr("clearFiles")
	defer timer.Start(intro, verboseTimer)()
	verbose.Println(intro, ": Cleaning-up the Go files")

	if g.dontClearFile {
		verbose.Println(intro, ":\tSkipping")
		fmt.Println("Gosh directory:", g.goshDir)
		return
	}

	err := os.RemoveAll(g.goshDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't remove the Go files:", err)
		fmt.Fprintln(os.Stderr, "Gosh directory:", g.goshDir)
		os.Exit(1)
	}
}

// formatFile runs the formatter over the populated program file
func (g *Gosh) formatFile() {
	intro := constantWidthStr("formatFile")
	defer timer.Start(intro, verboseTimer)()
	verbose.Println(intro, ": Formatting the Go file")

	if !g.formatterSet {
		if _, err := exec.LookPath(goImportsFormatter); err == nil {
			g.formatter = goImportsFormatter
			verbose.Println(intro, ":\tUsing ", goImportsFormatter)
		}
	}

	g.formatterArgs = append(g.formatterArgs, g.filename)
	verbose.Println(intro,
		":\tCommand: ", g.formatter, " ", strings.Join(g.formatterArgs, " "))
	out, err := exec.Command(g.formatter, g.formatterArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't format the Go file:", err)
		fmt.Fprintln(os.Stderr, "\tfilename:", g.filename)
		fmt.Fprintln(os.Stderr, string(out))

		if g.editRepeat {
			return
		}
		fmt.Fprintln(os.Stderr, "Gosh directory:", g.goshDir)
		os.Exit(1)
	}
}

// chdirInto will attempt to chdir into the given directory and will exit if
// it can't.
func (g Gosh) chdirInto(dir string) {
	verbose.Println(constantWidthStr("chdirInto"), ":\t", dir)

	err := os.Chdir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't chdir into %q: %v\n", dir, err)
		fmt.Fprintln(os.Stderr, "Gosh directory:", g.goshDir)
		os.Exit(1)
	}
}

// createGoFiles creates the file to hold the program and opens it. If no
// filename is given then a temporary directory is created, the program files
// and any module files are created in that directory.
func (g *Gosh) createGoFiles() {
	intro := constantWidthStr("createGoFiles")
	defer timer.Start(intro, verboseTimer)()

	verbose.Println(intro, ": Creating the Go files")

	verbose.Println(intro, ":\tCreating the temporary directory")
	var err error
	g.goshDir, err = os.MkdirTemp(g.baseTempDir, "gosh-*.d")
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Couldn't create the temporary directory %q: %v\n",
			g.goshDir, err)
		os.Exit(1)
	}

	g.chdirInto(g.goshDir)

	g.filename = filepath.Join(g.goshDir, "gosh.go")
	verbose.Println(intro, ":\tCreating the Go file: ", g.filename)
	g.makeFile()

	g.initModule()
}

// makeFile will create the go file and exit if it fails
func (g *Gosh) makeFile() {
	var err error
	g.w, err = os.Create(g.filename)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Couldn't create the Go file %q: %v\n", g.filename, err)
		fmt.Fprintln(os.Stderr, "Gosh directory:", g.goshDir)
		os.Exit(1)
	}
}

// runGoFile will call go build to generate the executable and then will run
// it unless dontRun is set.
func (g *Gosh) runGoFile() {
	intro := constantWidthStr("runGoFile")
	defer timer.Start(intro, verboseTimer)()
	verbose.Println(intro, ": Running the program")

	buildCmd := []string{"build"}
	buildCmd = append(buildCmd, g.buildArgs...)
	verbose.Println(intro, ":\tBuilding the program")
	verbose.Println(intro, ":\tCommand: go "+strings.Join(buildCmd, " "))
	if !gogen.ExecGoCmdNoExit(gogen.ShowCmdIO, buildCmd...) {
		if g.editRepeat {
			return
		}
		fmt.Fprintln(os.Stderr, "Gosh directory:", g.goshDir)
		os.Exit(1)
	}

	if g.dontRun {
		verbose.Println(intro, ":\tSkipping execution")
		return
	}

	g.chdirInto(g.runDir)

	verbose.Println(intro, ":\tExecuting the program: ", g.execName)
	cmd := exec.Command(filepath.Join(g.goshDir, g.execName))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Couldn't execute the program %q: %v\n", g.execName, err)
		if !g.editRepeat {
			fmt.Fprintln(os.Stderr, "Gosh directory:", g.goshDir)
			os.Exit(1)
		}
	}
}

// queryEditAgain will prompt the user asking if they want to edit the program
// again and return true if they reply yes or false if not
func (g Gosh) queryEditAgain() bool {
	if !g.editRepeat {
		return false
	}

	const indent = 10

	editAgainResp := responder.NewOrPanic(
		"Edit the program again",
		map[rune]string{
			'y': "to edit the file again",
			'n': "to stop editing and quit",
			'k': "to stop editing and quit, but keep the program",
		},
		responder.SetDefault('y'),
		responder.SetIndents(0, indent))

	response := editAgainResp.GetResponseOrDie()
	fmt.Println()

	switch response {
	case 'y':
		return true
	case 'k':
		g.dontClearFile = true
	}
	return false
}

// buildGoProgram creates the Go file and then writes the code into the it, then
// it formats the generated code.
func (g *Gosh) buildGoProgram() {
	intro := constantWidthStr("buildGoProgram")
	defer timer.Start(intro, verboseTimer)()

	verbose.Println(intro, ": Building the program")

	g.createGoFiles()
	defer g.w.Close()

	if g.showFilename {
		fmt.Println("Gosh filename:", g.filename)
	}
	verbose.Println(intro, ":\tGosh filename: ", g.filename)

	g.writeGoFile()
}

// setEditor returns true if the editor is valid - non-empty and is the
// name of an executable program. If it's non-empty but doesn't yield an
// executable program an error is added to the error map.
func (g *Gosh) setEditor() {
	if !g.edit {
		return
	}

	for _, trialEditor := range []struct {
		editor string
		source string
	}{
		{g.editorParam, "parameter"},
		{os.Getenv(envVisual), "Environment variable: " + envVisual},
		{os.Getenv(envEditor), "Environment variable: " + envEditor},
	} {
		editor := strings.TrimSpace(trialEditor.editor)
		if editor == "" {
			continue
		}

		var err error
		if _, err = exec.LookPath(editor); err == nil {
			g.editor = editor
			return
		}
		parts := strings.Fields(editor)
		if parts[0] == editor {
			g.addError("bad editor",
				fmt.Errorf("Cannot find %s (source: %s): %w",
					editor, trialEditor.source, err))
			continue
		}

		editor = parts[0]
		if _, err = exec.LookPath(editor); err == nil {
			g.editor = editor
			g.editorArgs = parts[1:]
			return
		}
		g.addError("bad editor",
			fmt.Errorf("Cannot find %s (source: %s): %w",
				editor, trialEditor.source, err))
		continue
	}

	g.addError("no editor",
		errors.New("No editor has been given."+
			" Possible sources are:"+
			"\n    the '"+paramNameScriptEditor+"' parameter,"+
			"\n    the '"+envVisual+"' environment variable"+
			"\n or the '"+envEditor+"' environment variable,"+
			"\nin that order."))
}

// editGoFile starts an editor to edit the program
func (g *Gosh) editGoFile() {
	if !g.edit {
		return
	}
	intro := constantWidthStr("editGoFile")
	defer timer.Start(intro, verboseTimer)()
	verbose.Println(intro, ": editing the program")

	g.editorArgs = append(g.editorArgs, g.filename)
	verbose.Println(intro,
		":\tCommand: "+g.editor+" "+strings.Join(g.editorArgs, " "))
	cmd := exec.Command(g.editor, g.editorArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't run the editor: %q\n", cmd.Path)
		fmt.Fprintf(os.Stderr, "\t%s\n", strings.Join(cmd.Args, " "))
		fmt.Fprintln(os.Stderr, "\tError:", err)
		fmt.Println("Gosh directory:", g.goshDir)
		os.Exit(1)
	}
}

// tidyModule runs go mod tidy after the file is fully constructed to
// populate the go.mod and go.sum files
func (g *Gosh) tidyModule() {
	intro := constantWidthStr("tidyModule")
	defer timer.Start(intro, verboseTimer)()
	verbose.Println(intro, ": tidying and populating the module files")

	if os.Getenv("GO111MODULE") == "off" {
		verbose.Println(intro, ":\tSkipping - GO111MODULES == 'off'")
		return
	}
	if g.filename == "" {
		verbose.Println(intro, ":\tSkipping - no filename")
		return
	}

	verbose.Println(intro, ":\tTidying the module (populates go.mod etc)")
	verbose.Println(intro, ":\tCommand: go mod tidy)")
	gogen.ExecGoCmd(gogen.NoCmdIO, "mod", "tidy")
}

// initModule runs go mod init
func (g *Gosh) initModule() {
	intro := constantWidthStr("initModule")
	defer timer.Start(intro, verboseTimer)()
	verbose.Println(intro, ": initialising the module files")

	if os.Getenv("GO111MODULE") == "off" {
		verbose.Println(intro, ":\tSkipping - GO111MODULES == 'off'")
		return
	}
	verbose.Println(intro, ":\tInitialising the go.mod file etc")
	verbose.Println(intro, ":\tCommand: go mod init "+g.execName)
	gogen.ExecGoCmd(gogen.NoCmdIO, "mod", "init", g.execName)

	keys := []string{}
	for k := range g.localModules {
		keys = append(keys, k)
	}
	if len(keys) > 0 {
		verbose.Println(intro, ":\tAdding local modules")
		sort.Strings(keys)
		for _, k := range keys {
			importPath := strings.TrimSuffix(k, "/")
			verbose.Println(intro,
				":\t\tReplacing "+importPath+
					" with "+g.localModules[k])
			gogen.ExecGoCmd(gogen.NoCmdIO, "mod", "edit",
				"-replace="+importPath+"="+g.localModules[k])
		}
	}
}
