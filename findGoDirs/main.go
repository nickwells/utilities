package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/dirsearch.mod/dirsearch"
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
)

// Created: Thu Jun 11 12:43:33 2020

const (
	printAct    = "print"
	buildAct    = "build"
	installAct  = "install"
	generateAct = "generate"
)

var noAction bool

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

var dir string = "."
var actions = make(map[string]bool)
var actionFuncs = map[string]func(string){
	printAct:    doPrint,
	buildAct:    doBuild,
	installAct:  doInstall,
	generateAct: doGenerate,
}

var generateArgs = []string{}
var installArgs = []string{}
var buildArgs = []string{}

var pkgNames []string
var filesWanted []string
var filesMissing []string

func main() {
	ps := paramset.NewOrDie(
		addParams,
		addExamples,
		param.SetProgramDescription(
			"This will search for directories containing Go packages. You"+
				" can add extra criteria for selecting the directory."+
				" Once in each selected directory you can perform certain"+
				" actions"),
	)

	ps.Parse()

	dirs, errs := dirsearch.FindRecursePrune(dir, -1,
		[]check.FileInfo{
			check.FileInfoName(check.StringNot(
				check.StringEquals("testdata"), "Ignore testdata directories")),
			check.FileInfoName(check.StringNot(
				check.StringHasPrefix("_"),
				"Ignore directories with name starting with '_'")),
			check.FileInfoName(check.StringNot(
				check.StringHasPrefix("."),
				"Ignore hidden directories (including .git)")),
		},
		check.FileInfoIsDir)
	for _, err := range errs {
		fmt.Println("Err:", err)
	}
	sortedDirs := make([]string, 0, len(dirs))
	for d := range dirs {
		sortedDirs = append(sortedDirs, d)
	}
	sort.Strings(sortedDirs)
	for _, d := range sortedDirs {
		onMatchDo(d, actions)
	}
}

// onMatchDo performs the actions if the directory is a go package directory
// meeting the criteria
func onMatchDo(dir string, actions map[string]bool) {
	undo, err := cd(dir)
	if err != nil {
		return
	}
	defer undo()

	if !isPkg(pkgNames) {
		return
	}

	if !hasFiles(filesWanted) {
		return
	}

	if len(filesMissing) > 0 && hasFiles(filesMissing) {
		return
	}

	// We force the order that actions take place - we should always generate
	// any files before building or installing (if generate is requested)
	for _, a := range []string{printAct, generateAct, buildAct, installAct} {
		if actions[a] {
			actionFuncs[a](dir)
		}
	}
}

// cd will change directory to the given directory name and return a function
// to be called to get back to the original directory
func cd(dir string) (func(), error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	err = os.Chdir(dir)
	if err != nil {
		return nil, err
	}
	return func() {
		os.Chdir(cwd) //nolint: errcheck
	}, nil
}

// isPkg will try to run the command to get the package name. If this fails,
// it returns false. Otherwise it will compare the package name against the
// list of target packages and return true only if any of them match.
func isPkg(pkgNames []string) bool {
	pkg, err := gogen.GetPackage()
	if err != nil { // it's not a package directory
		return false
	}

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

	filesInDir, err := ioutil.ReadDir(".")
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
func fileFound(name string, files []os.FileInfo) bool {
	for _, f := range files {
		if f.Name() == name {
			return true
		}
	}
	return false
}
