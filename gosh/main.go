package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

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
	return fmt.Sprintf("%18.18s", s)
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

func main() {
	defer timer.Start(constantWidthStr("main"), verboseTimer)()

	g := NewGosh()
	slp := &snippetListParams{}

	ps := paramset.NewOrDie(
		verbose.AddParams,

		addSnippetListParams(slp),
		addSnippetParams(g),
		addWebParams(g),
		addReadloopParams(g),
		addParams(g),

		addNotes(g),
		addExamples,
		addReferences,

		SetGlobalConfigFile,
		SetConfigFile,

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

	ps.Parse()

	if slp.listSnippets || slp.listDirs {
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
		if slp.listDirs {
			for _, dir := range g.snippetDirs {
				fmt.Println(dir)
			}
		}
		os.Exit(0)
	}

	g.snippets.Check(g.errMap)
	g.checkScripts()
	g.reportErrors()

	g.buildGoProgram()

	g.reportErrors()

	g.runGoFile()

	g.clearFiles()
}

// clearFiles removes the created program file, any module files and the
// containing directory unless the dontClearFile flag is set
func (g *Gosh) clearFiles() {
	intro := constantWidthStr("clearFiles")
	defer timer.Start(intro, verboseTimer)()

	if g.dontClearFile {
		verbose.Print(intro, ": Skipping\n")
		return
	}

	verbose.Print(intro, ": Cleaning-up the Go files\n")

	err := os.RemoveAll(g.cleanupPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't remove the Go files:", err)
		fmt.Fprintln(os.Stderr, "\t:", g.cleanupPath)
		os.Exit(1)
	}
}

// formatFile runs the formatter over the populated program file
func (g *Gosh) formatFile() {
	intro := constantWidthStr("formatFile")
	defer timer.Start(intro, verboseTimer)()

	verbose.Print(intro, ": Formatting the Go file\n")
	if !g.formatterSet {
		if _, err := exec.LookPath(goImportsFormatter); err == nil {
			g.formatter = goImportsFormatter
			verbose.Print(intro, ":\tUsing ", goImportsFormatter, "\n")
		}
	}

	g.formatterArgs = append(g.formatterArgs, g.filename)
	verbose.Print(intro, ":\tCommand: ",
		g.formatter, " ",
		strings.Join(g.formatterArgs, " "),
		"\n")
	out, err := exec.Command(g.formatter, g.formatterArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't format the Go file:", err)
		fmt.Fprintln(os.Stderr, "\tfilename:", g.filename)
		fmt.Fprintln(os.Stderr, string(out))
		os.Exit(1)
	}
}

// createGoFiles creates the file to hold the program and opens it. If no
// filename is given then a temporary directory is created, the program files
// and any module files are created in that directory.
func (g *Gosh) createGoFiles() {
	intro := constantWidthStr("createGoFiles")
	defer timer.Start(intro, verboseTimer)()

	verbose.Print(intro, ": Creating the Go files\n")

	if g.filename != "" {
		verbose.Print(intro, ":\tCreating ", g.filename, "\n")
		g.cleanupPath = g.filename
		g.makeFile()
		return
	}

	verbose.Print(intro, ":\tCreating the temporary directory\n")
	d, err := os.MkdirTemp(g.baseTempDir, "gosh-*.d")
	if err != nil {
		fmt.Fprintln(os.Stderr,
			"Couldn't create the temporary directory:", err)
		os.Exit(1)
	}
	g.cleanupPath = d

	verbose.Print(intro, ":\tChdir'ing into ", d, "\n")
	err = os.Chdir(d)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Couldn't chdir to the temporary directory (%s): %v", d, err)
		os.Exit(1)
	}

	g.filename = filepath.Join(d, "gosh.go")
	verbose.Print(intro, ":\tCreating the Go file: ", g.filename, "\n")
	g.makeFile()

	g.initModule()
}

// makeFile will create the go file and exit if it fails
func (g *Gosh) makeFile() {
	var err error
	g.w, err = os.Create(g.filename)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Couldn't create the Go file (%s): %v", g.filename, err)
		os.Exit(1)
	}
}

// runGoFile will call go run to execute the constructed program
func (g *Gosh) runGoFile() {
	intro := constantWidthStr("runGoFile")
	defer timer.Start(intro, verboseTimer)()

	if g.dontRun {
		verbose.Print(intro, ": Skipping\n")
		return
	}

	verbose.Print(intro, ": Running the Go file\n")
	gogen.ExecGoCmd(gogen.ShowCmdIO, "run", g.filename)
}

// buildGoProgram creates the Go file and then writes the code into the it, then
// it formats the generated code. It returns the name of the generated file.
func (g *Gosh) buildGoProgram() {
	intro := constantWidthStr("buildGoProgram")
	defer timer.Start(intro, verboseTimer)()

	verbose.Print(intro, ": Building the program\n")

	g.createGoFiles()
	defer g.w.Close()

	if g.showFilename {
		fmt.Println(g.filename)
	}
	verbose.Print(intro, ":\tGo file name: ", g.filename, "\n")

	g.writeGoFile()
	g.editGoProgram()
	g.reportErrors()

	g.formatFile()

	g.tidyModule()
}

// checkEditor returns true if the editor is valid - non-empty and is the
// name of an executable program. If it's non-empty but doesn't yield an
// executable program an error is added to the error map.
func (g *Gosh) checkEditor(editor, source string) bool {
	editor = strings.TrimSpace(editor)
	if editor == "" {
		return false
	}

	var err error
	if _, err = exec.LookPath(editor); err == nil {
		g.editor = editor
		return true
	}
	re := regexp.MustCompile(`\s+`)
	parts := re.Split(editor, -1)
	if parts[0] == editor {
		g.addError("bad editor",
			fmt.Errorf("Cannot find %s (source: %s): %w",
				editor, source, err))
		return false
	}

	editor = parts[0]
	if _, err = exec.LookPath(editor); err == nil {
		g.editor = editor
		g.editorParams = parts[1:]
		return true
	}
	g.addError("bad editor",
		fmt.Errorf("Cannot find %s (source: %s): %w",
			editor, source, err))
	return false
}

// editGoProgram starts an editor to edit the program
func (g *Gosh) editGoProgram() {
	if !g.edit {
		return
	}
	if !g.checkEditor(g.editor, "parameter") {
		if !g.checkEditor(os.Getenv(envVisual), envVisual) {
			if !g.checkEditor(os.Getenv(envEditor), envEditor) {
				g.addError("no editor",
					errors.New("No editor has been given."+
						" Possible sources are:"+
						" the '"+paramNameScriptEditor+"' parameter,"+
						" the '"+envVisual+"' environment variable"+
						" or the '"+envEditor+"' environment variable,"+
						" in that order."))
				return
			}
		}
	}
	g.editorParams = append(g.editorParams, g.filename)
	cmd := exec.Command(g.editor, g.editorParams...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't run the editor: %q\n", cmd.Path)
		fmt.Fprintf(os.Stderr, "\t%s\n", strings.Join(cmd.Args, " "))
		fmt.Fprintln(os.Stderr, "\tError:", err)
		os.Exit(1)
	}
}

// tidyModule runs go mod tidy after the file is fully constructed to
// populate the go.mod and go.sum files
func (g *Gosh) tidyModule() {
	intro := constantWidthStr("tidyModule")
	defer timer.Start(intro, verboseTimer)()

	if os.Getenv("GO111MODULE") == "off" {
		verbose.Print(intro, ":\tSkipping - GO111MODULES == 'off'\n")
		return
	}
	if g.filename == "" {
		verbose.Print(intro, ":\tSkipping - no filename\n")
		return
	}

	verbose.Print(intro, ":\tRunning 'go mod tidy' (populates go.mod)\n")
	gogen.ExecGoCmd(gogen.NoCmdIO, "mod", "tidy")
}

// initModule runs go mod init
func (g *Gosh) initModule() {
	intro := constantWidthStr("initModule")
	defer timer.Start(intro, verboseTimer)()

	if os.Getenv("GO111MODULE") == "off" {
		verbose.Print(intro, ":\tSkipping - GO111MODULES == 'off'\n")
		return
	}
	verbose.Print(intro,
		":\tRunning 'go mod init gosh' (creates the go.mod etc)\n")
	gogen.ExecGoCmd(gogen.NoCmdIO, "mod", "init", "gosh")

	keys := []string{}
	for k := range g.localModules {
		keys = append(keys, k)
	}
	if len(keys) > 0 {
		verbose.Print(intro, ":\tAdding local modules\n")
		sort.Strings(keys)
		for _, k := range keys {
			importPath := strings.TrimSuffix(k, "/")
			gogen.ExecGoCmd(gogen.NoCmdIO, "mod", "edit",
				"-replace="+importPath+"="+g.localModules[k])
		}
	}
}
