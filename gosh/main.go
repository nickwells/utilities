package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nickwells/cli.mod/cli/responder"
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/snippet.mod/snippet"
	"github.com/nickwells/verbose.mod/verbose"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// Created: Wed Sep  4 09:58:54 2019

// makeParamSet creates the parameter set ready for argument parsing
func makeParamSet(g *Gosh, slp *snippetListParams) *param.PSet {
	return paramset.NewOrDie(
		verbose.AddParams,
		versionparams.AddParams,

		addSnippetListParams(slp),
		addSnippetParams(g),
		addWebParams(g),
		addReadloopParams(g),
		addGoshParams(g),
		addParams(g),

		addNotes,
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

		SetGlobalConfigFile,
		SetConfigFile,
	)
}

func main() {
	g := newGosh()
	slp := &snippetListParams{}

	ps := makeParamSet(g, slp)

	ps.Parse()
	defer g.dbgStack.Start("main", os.Args[0])()

	listSnippets(g, slp)

	g.snippets.Check(g.errMap)
	g.checkScripts()
	g.reportErrors()

	g.setEditor()
	g.reportErrors()

	g.constructGoProgram()
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
		g.reportFatalError("configure the snippet list", "", err)

		lc.List()
		g.reportErrors()
	}

	os.Exit(0)
}

// clearFiles removes the created program file, any module files and the
// containing directory unless the dontClearFile flag is set
func (g *Gosh) clearFiles() {
	defer g.dbgStack.Start("clearFiles", "Cleaning-up the Go files")()
	intro := g.dbgStack.Tag()

	if g.dontClearFile {
		verbose.Println(intro, " Skipping")
		fmt.Println("Gosh directory:", g.goshDir)
		return
	}

	err := os.RemoveAll(g.goshDir)
	g.reportFatalError("remove the gosh directory", g.goshDir, err)
}

// formatFile runs the formatter over the populated program file
func (g *Gosh) formatFile() {
	defer g.dbgStack.Start("formatFile", "Formatting the Go file")()
	intro := g.dbgStack.Tag()

	if g.dontFormat {
		verbose.Println(intro, " Skipping formatting")
		return
	}

	if !g.formatterSet {
		if _, err := exec.LookPath(goImportsFormatter); err == nil {
			g.formatter = goImportsFormatter
			verbose.Println(intro, " Using ", goImportsFormatter)
		}
	}

	g.formatterArgs = append(g.formatterArgs, g.filename)
	verbose.Println(intro,
		" Command: ", g.formatter, " ", strings.Join(g.formatterArgs, " "))
	out, err := exec.Command(g.formatter, g.formatterArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't format the Go file:", err)
		fmt.Fprintln(os.Stderr, "\tfilename:", g.filename)
		fmt.Fprintln(os.Stderr, string(out))

		if g.editRepeat {
			return
		}
		fmt.Fprintln(os.Stderr, "Gosh directory:", g.goshDir)
		os.Exit(1) // nolint:gocritic
	}
}

// chdirInto will attempt to chdir into the given directory and will exit if
// it can't.
func (g Gosh) chdirInto(dir string) {
	defer g.dbgStack.Start("chdirInto", "cd'ing into "+dir)()

	err := os.Chdir(dir)
	g.reportFatalError("chdir into directory", dir, err)
}

// createGoFiles creates the file to hold the program and opens it. If no
// filename is given then a temporary directory is created, the program files
// and any module files are created in that directory.
func (g *Gosh) createGoFiles() {
	defer g.dbgStack.Start("createGoFiles", "Creating the Go files")()
	intro := g.dbgStack.Tag()

	verbose.Println(intro, " Creating the temporary directory")
	var err error
	g.goshDir, err = os.MkdirTemp(g.baseTempDir, "gosh-*.d")
	g.reportFatalError("create the temporary directory", g.goshDir, err)

	g.chdirInto(g.goshDir)

	g.filename = filepath.Join(g.goshDir, "gosh.go")
	verbose.Println(intro, " Creating the Go file: ", g.filename)
	g.makeFile()

	g.initModule()
	g.initWorkspace()
}

// makeFile will create the go file and exit if it fails
func (g *Gosh) makeFile() {
	var err error
	g.w, err = os.Create(g.filename)
	g.reportFatalError("create the Go file", g.filename, err)
}

// makeExecutable runs go build to make the executable file
func (g *Gosh) makeExecutable() bool {
	defer g.dbgStack.Start("makeExecutable", "Building the program")()
	intro := g.dbgStack.Tag()

	buildCmd := []string{"build"}
	buildCmd = append(buildCmd, g.buildArgs...)
	verbose.Println(intro, " Command: go "+strings.Join(buildCmd, " "))
	return gogen.ExecGoCmdNoExit(gogen.ShowCmdIO, buildCmd...)
}

// runGoFile will call go build to generate the executable and then will run
// it unless dontRun is set.
func (g *Gosh) runGoFile() {
	defer g.dbgStack.Start("runGoFile", "Running the program")()
	intro := g.dbgStack.Tag()

	if !g.makeExecutable() {
		if g.editRepeat {
			return
		}
		fmt.Fprintln(os.Stderr, "Gosh directory:", g.goshDir)
		os.Exit(1) // nolint:gocritic
	}

	if g.dontRun {
		verbose.Println(intro, " Skipping execution")
		return
	}

	g.chdirInto(g.runDir)

	g.executeProgram()
}

// executeProgram executes the newly built executeProgram
func (g *Gosh) executeProgram() {
	defer g.dbgStack.Start("executeProgram",
		"Executing the program: "+g.execName)()

	cmd := exec.Command(filepath.Join(g.goshDir, g.execName))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode := ee.ProcessState.ExitCode()
			if exitCode == -1 { // Program exited due to receiving a signal
				return
			}
		}
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
func (g *Gosh) queryEditAgain() bool {
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

// constructGoProgram creates the Go file and then writes the code into the it, then
// it formats the generated code.
func (g *Gosh) constructGoProgram() {
	defer g.dbgStack.Start("constructGoProgram", "Constructing the program")()

	g.createGoFiles()
	defer g.w.Close()

	if g.showFilename {
		fmt.Println("Gosh filename:", g.filename)
	}

	g.writeGoFile()
	g.copyFiles()
}

// copyFiles will read the files to be copied and write them into the gosh
// directory.
func (g *Gosh) copyFiles() {
	for i, fromName := range g.copyGoFiles {
		toName := fmt.Sprintf("goshCopy%02d%s", i, filepath.Base(fromName))
		if !filepath.IsAbs(fromName) {
			fromName = filepath.Clean(filepath.Join(g.runDir, fromName))
		}

		content, err := os.ReadFile(fromName)
		g.reportFatalError("read the file to be copied", fromName, err)

		err = os.WriteFile(toName, content, 0o644)
		g.reportFatalError("write the file to be copied", toName, err)
	}
}

// setEditor sets the script editor to be used. If the editor is set but
// cannot be found in the execution path then an error is added to the error
// map.
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

	defer g.dbgStack.Start("editGoFile", "editing the program")()
	intro := g.dbgStack.Tag()

	g.editorArgs = append(g.editorArgs, g.filename)
	verbose.Println(intro,
		" Command: "+g.editor+" "+strings.Join(g.editorArgs, " "))
	cmd := exec.Command(g.editor, g.editorArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	g.reportFatalError("run the editor",
		cmd.Path+"\t"+strings.Join(cmd.Args, ""),
		err)
}

// tidyModule runs go mod tidy after the file is fully constructed to
// populate the go.mod and go.sum files
func (g *Gosh) tidyModule() {
	defer g.dbgStack.Start("tidyModule", "Tidying & populating module files")()
	intro := g.dbgStack.Tag()

	if os.Getenv("GO111MODULE") == "off" {
		verbose.Println(intro, " Skipping - GO111MODULES == 'off'")
		return
	}
	if g.filename == "" {
		verbose.Println(intro, " Skipping - no filename")
		return
	}

	verbose.Println(intro, " Command: go mod tidy")
	if g.ignoreGoModTidyErrs {
		gogen.ExecGoCmdNoExit(gogen.NoCmdFailIO, "mod", "tidy")
	} else {
		gogen.ExecGoCmd(gogen.NoCmdIO, "mod", "tidy")
	}
}

// initModule runs go mod init
func (g *Gosh) initModule() {
	defer g.dbgStack.Start("initModule", "Initialising the module files")()
	intro := g.dbgStack.Tag()

	if os.Getenv("GO111MODULE") == "off" {
		verbose.Println(intro, " Skipping - GO111MODULES == 'off'")
		return
	}
	verbose.Println(intro, " Command: go mod init "+g.execName)
	gogen.ExecGoCmd(gogen.NoCmdIO, "mod", "init", g.execName)

	keys := []string{}
	for k := range g.localModules {
		keys = append(keys, k)
	}
	if len(keys) > 0 {
		verbose.Println(intro, " Adding local modules")
		sort.Strings(keys)
		for _, k := range keys {
			importPath := strings.TrimSuffix(k, "/")
			verbose.Println(intro,
				" Replacing "+importPath+
					" with "+g.localModules[k])
			gogen.ExecGoCmd(gogen.NoCmdIO, "mod", "edit",
				"-replace="+importPath+"="+g.localModules[k])
		}
	}
}

// initWorkspace initialises the workspace file if any workspace use values
// have been given
func (g *Gosh) initWorkspace() {
	if len(g.workspace) == 0 {
		return
	}

	defer g.dbgStack.Start("initWorkspace", "Initialising the workspace")()
	intro := g.dbgStack.Tag()

	verbose.Println(intro, " Command: go work init .")
	gogen.ExecGoCmd(gogen.NoCmdIO, "work", "init", ".")

	for _, ws := range g.workspace {
		verbose.Println(intro, " Command: go work use "+ws)
		gogen.ExecGoCmd(gogen.NoCmdIO, "work", "use", ws)
	}

	verbose.Println(intro, " Command: go work sync")
	gogen.ExecGoCmd(gogen.NoCmdIO, "work", "sync")
}

// reportFatalError will report the failure of the action if the err is
// non-nil and will exit.
func (g *Gosh) reportFatalError(action, name string, err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "Couldn't %s", action)
	if name != "" {
		fmt.Fprintf(os.Stderr, " %q", name)
	}
	fmt.Fprintf(os.Stderr, ": %v\n", err)
	if g.goshDir != "" {
		fmt.Fprintln(os.Stderr, "Gosh directory:", g.goshDir)
	}
	os.Exit(1)
}
