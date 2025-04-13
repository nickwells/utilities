package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/twrap.mod/twrap"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Sat May 25 16:13:02 2019

const (
	dfltFileName      = "setConfigFile.go"
	groupFileNameBase = "setConfigFile"
)

// prog holds the parameter values and intermediate results for the program
type prog struct {
	// parameters
	mustExist         bool
	mustExistPersonal bool
	mustExistGlobal   bool
	privateFunc       bool
	makeFile          bool

	whichFuncs      string
	outputFileName  string
	groupName       string
	baseDirPersonal string
	baseDirGlobal   string

	// intermediate results
	pkgName string
	dirs    []string
}

// newProg returns an initialised Prog struct
func newProg() *prog {
	return &prog{
		makeFile: true,

		whichFuncs:     "all",
		outputFileName: dfltFileName,
	}
}

func main() {
	prog := newProg()
	ps := makeParamSet(prog)

	ps.Parse()

	f := os.Stdout
	if prog.makeFile {
		f = gogen.MakeFileOrDie(prog.outputFileName)
		defer f.Close()
	}

	gogen.PrintPreamble(f, ps)
	gogen.PrintImports(f,
		"path/filepath",
		"github.com/nickwells/filecheck.mod/filecheck",
		"github.com/nickwells/param.mod/v6/param",
		"github.com/nickwells/xdg.mod/xdg")

	prog.pkgName = gogen.GetPackageOrDie()
	prog.dirs = strings.Split(gogen.GetImportPathOrDie(), "/")

	prog.printFuncPersonal(f)
	prog.printFuncGlobal(f)
}

// makeFuncNameGlobal generates the name of the function for setting the
// global config file that this program will write.
func (prog *prog) makeFuncNameGlobal() string {
	prefix := "Set"
	if prog.privateFunc {
		prefix = "set"
	}

	fName := prefix + "GlobalConfigFile" + prog.makeGroupSuffix()
	verbose.Print("Global function name: ", fName, "\n")

	return fName
}

// makeFuncNamePersonal generates the name of the function for setting the
// personal config file that this program will write.
func (prog *prog) makeFuncNamePersonal() string {
	prefix := "Set"
	if prog.privateFunc {
		prefix = "set"
	}

	fName := prefix + "ConfigFile" + prog.makeGroupSuffix()
	verbose.Print("Personal function name: ", fName, "\n")

	return fName
}

// makeGroupSuffix generates the suffix for the function name from the group
// name. It cleans up any characters in the name which are invalid characters
// in a Go function name
func (prog *prog) makeGroupSuffix() string {
	if prog.groupName == "" {
		return ""
	}

	groupSuffix := "ForGroup"

	localGroupName := prog.groupName
	// Now split the group name into words and Titleise each word, adding it
	// to the group suffix
	localGroupName = strings.ReplaceAll(localGroupName, "-", ".")

	groupParts := strings.SplitSeq(localGroupName, ".")
	for part := range groupParts {
		r := []rune(part)
		r[0] = unicode.ToUpper(r[0])
		groupSuffix += string(r)
	}

	return groupSuffix
}

// makeAddCFName generates the function name called to add the config file to
// the set of config files. If the package name is "main" then the config
// file will be strict (the parameters must exist).
//
// Note that the opening parenthesis is given as part of the name, this is
// because for group config files the first parameter (the group name) is
// generated as part of the name.
func (prog *prog) makeAddCFName() string {
	if prog.groupName != "" {
		return fmt.Sprintf("ps.AddGroupConfigFile(%q,", prog.groupName)
	}

	if prog.pkgName == "main" {
		return "ps.AddConfigFileStrict("
	}

	return "ps.AddConfigFile("
}

// makeConfigFileName generates the name of the config file - this varies
// according to whether or not this is for a group or main
func (prog *prog) makeConfigFileName() string {
	if prog.groupName == "" {
		return "common.cfg"
	}

	return "group-" + prog.groupName + ".cfg"
}

// printFuncIntro prints the function comment and the func name and signature
func (prog *prog) printFuncIntro(f io.Writer, name string) {
	twc := twrap.NewTWConfOrPanic(twrap.SetWriter(f))
	fmt.Fprintln(f)
	fmt.Fprintln(f, "/*")
	twc.Wrap(name+
		" adds a config file to the set which the param parser will process"+
		" before checking the command line parameters.", 0)

	if prog.whichFuncs == "all" {
		fmt.Fprintln(f)
		twc.Wrap(
			"This function is one of a pair which add the global and personal"+
				" config files. It is generally best practice to add the"+
				" global config file before adding the personal one. This"+
				" allows any system-wide defaults to be overridden by personal"+
				" choices. Also any parameters which can only be set once can"+
				" be set in the global config file, thereby enforcing a global"+
				" policy.",
			0)
	}

	fmt.Fprintln(f, "*/")
	fmt.Fprint(f, "func "+name+"(ps *param.PSet) error {")
}

// printFuncPersonal writes out the function for setting a personal config file
func (prog *prog) printFuncPersonal(f io.Writer) {
	if prog.whichFuncs != "all" && prog.whichFuncs != "personalOnly" {
		return
	}

	prog.printFuncIntro(f, prog.makeFuncNamePersonal())
	fmt.Fprint(f, "\n\tbaseDir := ")

	if prog.baseDirPersonal != "" {
		fmt.Fprintf(f, "%q\n", prog.baseDirPersonal)
	} else {
		fmt.Fprintln(f, "xdg.ConfigHome()")
	}

	prog.printAddCF(f, prog.dirs,
		prog.makeAddCFName(),
		prog.makeConfigFileName(),
		prog.mustExist || prog.mustExistPersonal)
	printFuncEnd(f)
}

// printFuncGlobal writes out the function for setting a shared, global
// config file
func (prog *prog) printFuncGlobal(f io.Writer) {
	if prog.whichFuncs != "all" && prog.whichFuncs != "globalOnly" {
		return
	}

	prog.printFuncIntro(f, prog.makeFuncNameGlobal())

	if prog.baseDirGlobal != "" {
		fmt.Fprintf(f, "\n\tbaseDir := %q\n", prog.baseDirGlobal)
	} else {
		fmt.Fprint(f, `
	dirs := xdg.ConfigDirs()
	if len(dirs) == 0 {
		return nil
	}

	baseDir := dirs[0]
`)
	}

	prog.printAddCF(f, prog.dirs,
		prog.makeAddCFName(),
		prog.makeConfigFileName(),
		prog.mustExist || prog.mustExistGlobal)
	printFuncEnd(f)
}

// printAddCF prints the lines of code that will call filepath.Join(...)
// with the base directory name and the strings from paramFileParts
func (prog *prog) printAddCF(f io.Writer, dirs []string, funcName, cfgFName string, mustExist bool) {
	fmt.Fprint(f, `
	`+funcName+`
		filepath.Join(baseDir`)

	const sep = ",\n\t\t\t"

	for _, p := range dirs {
		fmt.Fprintf(f, "%s%q", sep, p)
	}

	fmt.Fprintf(f, "%s%q),", sep, cfgFName)

	if mustExist {
		fmt.Fprint(f, `
		filecheck.MustExist)`)
	} else {
		fmt.Fprint(f, `
		filecheck.Optional)`)
	}
}

// printFuncEnd prints the common last lines of the function
func printFuncEnd(f io.Writer) {
	fmt.Fprint(f, `

	return nil
}
`)
}
