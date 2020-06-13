// findGoCmdDirs
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
	"github.com/nickwells/param.mod/v5/param/psetter"
)

// Created: Thu Jun 11 12:43:33 2020

const (
	printAct    = "print"
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

// doInstall will run go install
func doInstall(name string) {
	if noAction {
		fmt.Printf("%-20.20s : %s\n", "go install", name)
		return
	}
	gogen.ExecGoCmd(gogen.ShowCmdIO, "install")
}

// doGenerate will run go generate
func doGenerate(name string) {
	if noAction {
		fmt.Printf("%-20.20s : %s\n", "go generate", name)
		return
	}
	gogen.ExecGoCmd(gogen.ShowCmdIO, "generate")
}

var dir string = "."
var actions = []string{printAct}
var actionFuncs = map[string]func(string){
	printAct:    doPrint,
	installAct:  doInstall,
	generateAct: doGenerate,
}

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
				check.StringEquals(".git"), "Ignore .git directories")),
		},
		check.FileInfoIsDir)
	for _, err := range errs {
		fmt.Println("Err:", err)
	}
	keys := make([]string, 0, len(dirs))
	for d := range dirs {
		keys = append(keys, d)
	}
	sort.Strings(keys)
	for _, d := range keys {
		onMatchDo(d, actions...)
	}
}

// onMatchDo performs the actions if the directory is a go package directory
// meeting the criteria
func onMatchDo(dir string, actions ...string) {
	undo, err := cd(dir)
	if err != nil {
		return
	}
	defer undo()

	if !isPkg() {
		return
	}

	if !hasFiles(filesWanted) {
		return
	}

	if len(filesMissing) > 0 && hasFiles(filesMissing) {
		return
	}

	for _, action := range actions {
		actionFuncs[action](dir)
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
func isPkg() bool {
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

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	ps.Add("dir", psetter.Pathname{Value: &dir},
		"set the name of the directory to search from",
		param.AltName("d"),
	)

	ps.Add("actions",
		psetter.EnumList{
			Value: &actions,
			AllowedVals: psetter.AllowedVals{
				installAct:  "install the command (go install)",
				generateAct: "generate any auto-generated files (go generate)",
				printAct:    "print the directory name",
			},
		},
		"set the actions to perform when a Go command directory is discovered",
		param.AltName("a"),
		param.AltName("do"),
	)

	ps.Add("package-names", psetter.StrList{Value: &pkgNames},
		"set the names of packages to be matched. If this is not set then"+
			" any package name will be matched",
		param.AltName("pkg"),
	)

	ps.Add("having-files", psetter.StrList{Value: &filesWanted},
		"give a list of files that the directory must contain. All the"+
			" listed files must be present for the directory to be"+
			" matched.",
		param.AltName("having"),
		param.AltName("with"),
	)

	ps.Add("missing-files", psetter.StrList{Value: &filesMissing},
		"give a list of files that the directory may not contain. Any of"+
			" the listed files may be absent for the directory to be"+
			" matched.",
		param.AltName("not-having"),
		param.AltName("without"),
	)

	ps.Add("no-action", psetter.Bool{Value: &noAction},
		"this will stop any action from happening. Instead the action"+
			" functions will just report what they would have done.",
		param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
	)

	return nil
}

// addExamples will add some examples to the help message
func addExamples(ps *param.PSet) error {
	ps.AddExample(`findGoDirs -pkg main`,
		"This will search recursively down from the current directory for"+
			" any directory which contains Go code where the package name"+
			" is 'main', ignoring the contents of any .git directories."+
			" For each directory it finds it will print the name of the"+
			" directory.")
	ps.AddExample(`findGoDirs -pkg main -actions install`,
		"This will install all the Go programs under the current directory.")
	ps.AddExample(`findGoDirs -pkg main -d github.com/nickwells -do install`,
		"This will install all the Go programs under github.com/nickwells.")

	return nil
}
