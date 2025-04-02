package main

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/nickwells/cli.mod/cli/responder"
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/snippet.mod/snippet"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Wed Sep  4 09:58:54 2019

func main() {
	g := newGosh()
	slp := &snippetListParams{}

	ps := makeParamSet(g, slp)

	ps.Parse()

	preCheck(g)

	listSnippets(g, slp)

	defer func() { os.Exit(g.exitStatus) }()
	defer g.dbgStack.Start("main", os.Args[0])()

	g.snippets.Check(g.errMap)
	g.checkScripts()
	g.reportErrors()

	g.setEditor()
	g.reportErrors()

	g.constructGoProgram()
	g.reportErrors()

	for {
		g.dontCleanup = g.dontCleanupUserChoice

		g.editGoFile()
		g.populateImports()
		g.formatFile()
		g.tidyModule()
		g.runGoFile()

		if !g.queryEditAgain() {
			break
		}

		g.chdirInto(g.goshDir)
	}

	g.cleanup()
}

// listSnippets checks the snippet list parameters and lists the snippet
// details accordingly. If any listing is done then the program will exit
// after listing is complete.
func listSnippets(g *gosh, slp *snippetListParams) {
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

// reportGoshfiles prints out the gosh directory and file names
func (g *gosh) reportGoshfiles() {
	if g.goshDir == "" {
		fmt.Println("There is no gosh directory")
		return
	}

	fmt.Println()
	fmt.Println("gosh directory   " + g.goshDir)
	fmt.Println("gosh code        " + filepath.Join(g.goshDir, goshFilename))
	fmt.Println("gosh executable  " + filepath.Join(g.goshDir, g.execName))
	fmt.Println()
}

// cleanup removes the created program file, any module files and the
// containing directory unless the dontClearFile flag is set
func (g *gosh) cleanup() {
	defer g.dbgStack.Start("cleanup", "Cleaning-up the Go files")()
	intro := g.dbgStack.Tag()

	if g.dontCleanup {
		verbose.Println(intro, " Skipping cleanup")

		g.reportGoshfiles()

		return
	}

	err := os.RemoveAll(g.goshDir)
	g.reportFatalError("remove the gosh directory", g.goshDir, err)
}

// formatFile runs the formatter over the populated (and possibly edited)
// program file
func (g *gosh) formatFile() {
	if !g.formatCode {
		return
	}

	defer g.dbgStack.Start("formatFile", "Formatting the Go file")()
	intro := g.dbgStack.Tag()

	if !g.formatterSet {
		f, path, ok := findFormatter(g)
		if !ok {
			verbose.Println(intro,
				" No formatter is available, skipping formatting")
			return
		}

		g.formatter = path
		g.formatterArgs = f.args
		g.formatterSet = true
	}

	args := append(g.formatterArgs, goshFilename) // nolint:gocritic

	verbose.Println(intro,
		" Command: ", g.formatter, " ", strings.Join(args, " "))

	out, err := exec.Command( //nolint:gosec
		g.formatter, args...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gosh couldn't format the Go file:", err)
		fmt.Fprintln(os.Stderr, string(out))
	}
}

// populateImports runs the importPopulator over the program file
func (g *gosh) populateImports() {
	defer g.dbgStack.Start("populateImports",
		"Setting imports for the Go file")()

	intro := g.dbgStack.Tag()

	if g.dontPopulateImports {
		verbose.Println(intro, " Skipping import population")
		return
	}

	if !g.importPopulatorSet {
		f, path, ok := findImporter(g)
		if !ok {
			verbose.Println(intro,
				" No importer is available, skipping import population")
			return
		}

		verbose.Println(intro, " Using the default importer: ", f.name)
		verbose.Println(intro, "                   pathname: ", path)
		verbose.Println(intro, "                  arguments: ",
			strings.Join(f.args, " "))

		g.importPopulator = path
		g.importPopulatorArgs = f.args
		g.importPopulatorSet = true
	}

	args := append(g.importPopulatorArgs, goshFilename) // nolint:gocritic

	verbose.Println(intro,
		" Command: ", g.importPopulator, " ", strings.Join(args, " "))

	out, err := exec.Command( //nolint:gosec
		g.importPopulator, args...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr,
			"gosh couldn't populate the Go file imports: ", err)
		fmt.Fprintln(os.Stderr, string(out))
	}
}

// chdirInto will attempt to chdir into the given directory and will exit if
// it can't.
func (g gosh) chdirInto(dir string) {
	defer g.dbgStack.Start("chdirInto", "cd'ing into "+dir)()

	err := os.Chdir(dir)
	g.reportFatalError("chdir into directory", dir, err)
}

// createGoshTmpDir creates the temporary directory that gosh will use to
// generate the program. It will change directory into this dir and create
// the module files and any requested workplace
func (g *gosh) createGoshTmpDir() {
	defer g.dbgStack.Start("createGoshTmpDir", "Creating the gosh directory")()
	intro := g.dbgStack.Tag()

	verbose.Println(intro, " Creating the temporary directory")

	var err error

	g.goshDir, err = os.MkdirTemp(g.baseTempDir, "gosh-*.d")
	g.reportFatalError("create the temporary directory", g.goshDir, err)

	g.chdirInto(g.goshDir)

	g.initModule()
	g.initWorkspace()
}

// makeExecutable runs go build to make the executable file
func (g *gosh) makeExecutable() bool {
	defer g.dbgStack.Start("makeExecutable", "Building the program")()
	intro := g.dbgStack.Tag()

	buildCmd := []string{"build"}
	buildCmd = append(buildCmd, g.buildArgs...)

	verbose.Println(intro, " Command: go "+strings.Join(buildCmd, " "))

	if !gogen.ExecGoCmdNoExit(gogen.ShowCmdIO, buildCmd...) {
		verbose.Println(intro, " Build failed")

		g.exitStatus = goshExitStatusBuildFail
		g.dontCleanup = true

		return false
	}

	return true
}

// runGoFile will call go build to generate the executable and then will run
// it unless dontRun is set.
func (g *gosh) runGoFile() {
	defer g.dbgStack.Start("runGoFile", "Running the program")()
	intro := g.dbgStack.Tag()

	if !g.makeExecutable() {
		return
	}

	if g.dontRun {
		verbose.Println(intro, " Skipping execution")
		return
	}

	g.chdirInto(g.runDir)

	g.executeProgram()
}

// executeProgram executes the newly built executeProgram
func (g *gosh) executeProgram() {
	defer g.dbgStack.Start("executeProgram",
		"Executing the program: "+g.execName)()

	intro := g.dbgStack.Tag()

	cmd := exec.Command( //nolint:gosec
		filepath.Join(g.goshDir, g.execName), g.args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := os.Environ()

	if g.clearEnv {
		env = []string{}
	}

	cmd.Env = g.populateEnv(env)

	g.exitStatus = 0

	err := cmd.Run()
	if err != nil {
		var ec int

		if ee, ok := err.(*exec.ExitError); ok {
			ec = ee.ExitCode()
		}

		switch {
		case ec > 0:
			verbose.Println(intro, fmt.Sprintf(" Program Exit Status: %d", ec))
			g.exitStatus = ec
		case ec == -1:
			verbose.Println(intro, " Program interrupted")
		default:
			fmt.Println("Error:", err.Error())

			g.exitStatus = goshExitStatusRunFail
			g.dontCleanup = true
		}
	}
}

// queryEditAgain will prompt the user asking if they want to edit the program
// again and return true if they reply yes or false if not
func (g *gosh) queryEditAgain() bool {
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
		g.dontCleanup = true
	}

	return false
}

// constructGoProgram creates the Go file and then writes the code into the
// file. Finally, it copies in any requested files.
func (g *gosh) constructGoProgram() {
	defer g.dbgStack.Start("constructGoProgram", "Constructing the program")()

	g.createGoshTmpDir()
	g.writeGoFile()
	g.copyFiles()
}

// copyFiles will read the files to be copied and write them into the gosh
// directory with a guaranteed unique name.
func (g *gosh) copyFiles() {
	const copyFilePerms = 0o600 // Owner: Read/Write, the rest, no permissions

	for i, fromName := range g.copyGoFiles {
		toName := fmt.Sprintf("goshCopy%02d%s", i, filepath.Base(fromName))

		if !filepath.IsAbs(fromName) {
			fromName = filepath.Clean(filepath.Join(g.runDir, fromName))
		}

		content, err := packageRename(fromName)
		g.reportFatalError("read the file to be copied", fromName, err)

		err = os.WriteFile(toName, content, copyFilePerms)
		g.reportFatalError("write the file to be copied", toName, err)
	}
}

// tidyModule runs go mod tidy after the file is fully constructed to
// populate the go.mod and go.sum files
func (g *gosh) tidyModule() {
	defer g.dbgStack.Start("tidyModule", "Tidying & populating module files")()
	intro := g.dbgStack.Tag()

	if g.dontRunGoModTidy {
		verbose.Println(intro, " Skipping - go mod tidy is not being run")
		return
	}

	if os.Getenv("GO111MODULE") == "off" {
		verbose.Println(intro, " Skipping - GO111MODULES == 'off'")
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
func (g *gosh) initModule() {
	defer g.dbgStack.Start("initModule", "Initialising the module files")()
	intro := g.dbgStack.Tag()

	if os.Getenv("GO111MODULE") == "off" {
		verbose.Println(intro, " Skipping - GO111MODULES == 'off'")
		return
	}

	verbose.Println(intro, " Command: go mod init "+g.execName)
	gogen.ExecGoCmd(gogen.NoCmdIO, "mod", "init", g.execName)

	keys := slices.Sorted(maps.Keys(g.localModules))

	if len(keys) > 0 {
		verbose.Println(intro, " Adding local modules")

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
func (g *gosh) initWorkspace() {
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
func (g *gosh) reportFatalError(action, name string, err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "Couldn't %s", action)

	if name != "" {
		fmt.Fprintf(os.Stderr, " %q", name)
	}

	fmt.Fprintf(os.Stderr, ": %v\n", err)

	g.reportGoshfiles()

	os.Exit(goshExitStatusMisc)
}
