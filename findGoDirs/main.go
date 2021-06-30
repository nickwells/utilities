package main

import (
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/dirsearch.mod/dirsearch"
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/utilities/internal/callstack"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Thu Jun 11 12:43:33 2020

const (
	printAct    = "print"
	buildAct    = "build"
	installAct  = "install"
	generateAct = "generate"
)

// doPrint will print the name
func doPrint(name string) {
	if noAction {
		fmt.Printf("%-20.20s : %s\n", "print", name)
		return
	}
	fmt.Println(name)
}

// doBuild will run go build
func doBuild(name string) {
	doGoCommand(name, "build", buildArgs)
}

// doInstall will run go install
func doInstall(name string) {
	doGoCommand(name, "install", installArgs)
}

// doGenerate will run go generate
func doGenerate(name string) {
	doGoCommand(name, "generate", generateArgs)
}

// doGoCommand will run the Go subcommand with the passed args
func doGoCommand(name, command string, cmdArgs []string) {
	if noAction {
		fmt.Printf("%-20.20s : %s\n", "go "+command, name)
		return
	}
	args := []string{command}
	args = append(args, cmdArgs...)
	gogen.ExecGoCmd(gogen.ShowCmdIO, args...)
}

var (
	baseDirs     []string
	skipDirs     []string
	pkgNames     []string
	filesWanted  []string
	filesMissing []string

	noAction bool

	actions = make(map[string]bool)

	actionFuncs = map[string]func(string){
		printAct:    doPrint,
		buildAct:    doBuild,
		installAct:  doInstall,
		generateAct: doGenerate,
	}

	generateArgs = []string{}
	installArgs  = []string{}
	buildArgs    = []string{}

	dbgStack = &callstack.Stack{}
)

func main() {
	ps := paramset.NewOrDie(
		verbose.AddParams,

		addParams,
		addExamples,
		param.SetProgramDescription(
			"This will search for directories containing Go packages. You"+
				" can add extra criteria for selecting the directory."+
				" Once in each selected directory you can perform certain"+
				" actions"),
	)

	ps.Parse()

	defer dbgStack.Start("main", os.Args[0])()

	sortedDirs := findMatchingDirs()
	for _, d := range sortedDirs {
		onMatchDo(d, actions)
	}
}

// findMatchingDirs finds directories in any of the baseDirs matching the
// given criteria. Note that this just finds directories, excluding those:
//
// - called testdata
// - starting with a dot
// - starting with an underscore
//
// It does not perform any of the other tests, on package names, file
// presence etc.
func findMatchingDirs() []string {
	defer dbgStack.Start("findMatchingDirs", "Find dirs matching criteria")()

	var dirs []string
	dirChecks := []check.FileInfo{
		check.FileInfoName(check.StringNot(
			check.StringEquals("testdata"),
			"Ignore any directory called testdata")),
		check.FileInfoName(check.StringNot(
			check.StringHasPrefix("_"),
			"Ignore directories with name starting with '_'")),
		check.FileInfoName(
			check.StringOr(
				check.StringNot(
					check.StringHasPrefix("."),
					"Ignore hidden directories (including .git)"),
				check.StringEquals("."),
				check.StringEquals(".."),
			)),
	}
	for _, skipDir := range skipDirs {
		dirChecks = append(dirChecks, check.FileInfoName(check.StringNot(
			check.StringEquals(skipDir),
			"Ignore any directory called "+skipDir)))
	}

	fileChecks := []check.FileInfo{check.FileInfoIsDir}
	fileChecks = append(fileChecks, dirChecks...)

	for _, dir := range baseDirs {
		matches, errs := dirsearch.FindRecursePrune(dir, -1,
			dirChecks,
			fileChecks...)
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "Error: %q : %v\n", dir, err)
		}
		for d := range matches {
			dirs = append(dirs, d)
		}
	}
	sort.Strings(dirs)
	return dirs
}

// onMatchDo performs the actions if the directory is a go package directory
// meeting the criteria
func onMatchDo(dir string, actions map[string]bool) {
	defer dbgStack.Start("onMatchDo", "Act on matching dir: "+dir)()
	intro := dbgStack.Tag()

	undo, err := cd(dir)
	if err != nil {
		verbose.Println(intro, " Skipping: couldn't chdir")
		return
	}
	defer undo()

	pkg, err := gogen.GetPackage()
	if err != nil { // it's not a package directory
		verbose.Println(intro, " Skipping: Not a package directory")
		return
	}

	if !pkgMatches(pkg, pkgNames) {
		verbose.Println(intro, " Skipping: Wrong package")
		return
	}

	if !hasFiles(filesWanted) {
		verbose.Println(intro, " Skipping: missing files")
		return
	}

	if len(filesMissing) > 0 && hasFiles(filesMissing) {
		verbose.Println(intro, " Skipping: has unwanted files")
		return
	}

	// We force the order that actions take place - we should always generate
	// any files before building or installing (if generate is requested)
	for _, a := range []string{printAct, generateAct, buildAct, installAct} {
		if actions[a] {
			verbose.Println(intro, " Doing: "+a)
			actionFuncs[a](dir)
		}
	}
}

// cd will change directory to the given directory name and return a function
// to be called to get back to the original directory
func cd(dir string) (func(), error) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot get the current directory:", err)
		return nil, err
	}
	err = os.Chdir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot chdir to %q: %v\n", dir, err)
		return nil, err
	}
	return func() {
		os.Chdir(cwd) //nolint: errcheck
	}, nil
}

// pkgMatches will compare the package name against the list of target
// packages, if any, and return true only if any of them match. If there are
// no names to match then any name will match.
func pkgMatches(pkg string, pkgNames []string) bool {
	if len(pkgNames) == 0 { // any name matches
		return true
	}

	for _, name := range pkgNames {
		if pkg == name { // this name matches
			return true
		}
	}
	return false // no name matches
}

// hasFiles will check to see if any of the listed files exists in the
// current directory and return false if any of them are missing. It will
// only return true if all the files are found in the directory
func hasFiles(files []string) bool {
	if len(files) == 0 {
		return true
	}

	filesInDir, err := os.ReadDir(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot read the directory:", err)
		return false
	}

	for _, fname := range files {
		if !fileFound(fname, filesInDir) {
			return false
		}
	}
	return true
}

// fileFound will return true if the name is in the list of files
func fileFound(name string, files []fs.DirEntry) bool {
	for _, f := range files {
		if f.Name() == name {
			return true
		}
	}
	return false
}
