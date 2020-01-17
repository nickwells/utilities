// mkparamfilefunc
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v3/param"
	"github.com/nickwells/param.mod/v3/param/paramset"
	"github.com/nickwells/param.mod/v3/param/psetter"
	"github.com/nickwells/twrap.mod/twrap"
)

// Created: Sat May 25 16:13:02 2019

const (
	dfltFileName      = "setConfigFile.go"
	groupFileNameBase = "setConfigFileForGroup_"
)

var mustExist bool
var mustExistPersonal bool
var mustExistGlobal bool

var whichFuncs = "all"

var dontMakeFile bool
var outputFileName = dfltFileName
var groupName string
var baseDirPersonal string
var baseDirGlobal string

func main() {
	ps := paramset.NewOrDie(
		addParams,
		param.SetProgramDescription(
			"This creates a file defining functions which set the"+
				" default parameter files for the package or program."+
				" These can be passed as another argument to the call"+
				" where you create the parameter set or called"+
				" directly, passing the parameter set and checking for"+
				" errors. The paths of the files are derived from the"+
				" XDG config directories and from the import path of"+
				" the package."+
				"\n\nIf a group name is given the output filename and"+
				" the function names will be derived from the group"+
				" name."+
				"\n\nIt may be called multiple times in the same"+
				" package directory with different group names and"+
				" with none and each time it will generate the"+
				" appropriate files, overwriting any previous files"+
				" with the same name"),
	)

	ps.Parse()

	importPath, pkgName := goList()
	dirs := strings.Split(importPath, "/")
	var goFile = openGoFile(outputFileName, dontMakeFile)

	printPreamble(goFile, pkgName, ps)

	printFuncPersonal(goFile, groupName, pkgName, dirs)
	printFuncGlobal(goFile, groupName, pkgName, dirs)

	goFile.Close()
}

// makeFuncNameGlobal generates the name of the function for setting the
// global config file that this program will write.
func makeFuncNameGlobal(groupName string) string {
	if groupName == "" {
		return "SetGlobalConfigFile"
	}
	return "SetGroupGlobalConfigFile" + makeGroupSuffix(groupName)
}

// makeFuncNamePersonal generates the name of the function for setting the
// personal config file that this program will write.
func makeFuncNamePersonal(groupName string) string {
	if groupName == "" {
		return "SetConfigFile"
	}
	return "SetGroupConfigFile" + makeGroupSuffix(groupName)
}

// makeGroupSuffix generates the suffix for the function name from the group
// name. It cleans up any characters in the name which are invalid characters
// in a Go function name
func makeGroupSuffix(groupName string) string {
	groupSuffix := "_" + groupName
	groupSuffix = strings.ReplaceAll(groupSuffix, ".", "_")
	groupSuffix = strings.ReplaceAll(groupSuffix, "-", "_")
	return groupSuffix
}

// makeAddCFName generates the function name called to add the config file to
// the set of config files. If the package name is "main" then the config
// file will be strict (the parameters must exist).
//
// Note that the opening parenthesis is given as part of the name, this is
// because for group config files the first parameter (the group name) is
// generated as part of the name.
func makeAddCFName(pkgName, groupName string) string {
	if groupName != "" {
		return fmt.Sprintf("ps.AddGroupConfigFile(%q,", groupName)
	}

	if pkgName == "main" {
		return "ps.AddConfigFileStrict("
	}

	return "ps.AddConfigFile("
}

// makeConfigFileName generates the name of the config file - this varies
// according to whether or not this is for a group or main
func makeConfigFileName(groupName string) string {
	if groupName == "" {
		return "common.cfg"
	}
	return "group-" + groupName + ".cfg"
}

// openGoFile creates the file, truncating it if it already exists and
// returning the open file. If an error is detected, it is reported and the
// program aborts.
func openGoFile(filename string, dontMakeFile bool) *os.File {
	if dontMakeFile {
		return os.Stdout
	}

	f, err := os.Create(filename)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	return f
}

// printFunc prints the func name and signature
func printFunc(f *os.File, name string) {
	twc := twrap.NewTWConfOrPanic(twrap.SetWriter(f))
	fmt.Fprintln(f, "/*")
	twc.Wrap(name+
		" adds a config file to the set which the param parser will process"+
		" before checking the command line parameters.", 0)
	if whichFuncs == "all" {
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
func printFuncPersonal(f *os.File, groupName, pkgName string, dirs []string) {
	if whichFuncs != "all" && whichFuncs != "personalOnly" {
		return
	}

	printFunc(f, makeFuncNamePersonal(groupName))
	if baseDirPersonal != "" {
		fmt.Fprintf(f, `
	baseDir := %q
`, baseDirPersonal)
	} else {
		fmt.Fprint(f, `
	baseDir := xdg.ConfigHome()
`)
	}

	printAddCF(f, dirs,
		makeAddCFName(pkgName, groupName),
		makeConfigFileName(groupName),
		mustExist || mustExistPersonal)
	printFuncEnd(f)
}

// printFuncGlobal writes out the function for setting a shared, global
// config file
func printFuncGlobal(f *os.File, groupName, pkgName string, dirs []string) {
	if whichFuncs != "all" && whichFuncs != "globalOnly" {
		return
	}
	printFunc(f, makeFuncNameGlobal(groupName))
	if baseDirGlobal != "" {
		fmt.Fprintf(f, `
	baseDir := %q
`, baseDirGlobal)
	} else {
		fmt.Fprint(f, `
	dirs := xdg.ConfigDirs()
	if len(dirs) == 0 {
		return nil
	}
	baseDir := dirs[0]
`)
	}

	printAddCF(f, dirs,
		makeAddCFName(pkgName, groupName),
		makeConfigFileName(groupName),
		mustExist || mustExistGlobal)
	printFuncEnd(f)
}

// printAddCF prints the lines of code that will call filepath.Join(...)
// with the base directory name and the the strings from paramFileParts
func printAddCF(f *os.File, dirs []string, funcName, cfgFName string, mustExist bool) {
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
func printFuncEnd(f *os.File) {
	fmt.Fprint(f, `
	return nil
}

`)
}

// goList runs the go list command to discover the ImportPath and Name
func goList() (importPath, pkgName string) {

	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}\n{{.Name}}")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	scanner := bufio.NewScanner(stdout)

	if scanner.Scan() {
		importPath = scanner.Text()
	} else {
		fmt.Fprint(os.Stderr, "can't read the package import path")
		os.Exit(1)
	}

	if scanner.Scan() {
		pkgName = scanner.Text()
	} else {
		fmt.Fprint(os.Stderr, "can't read the package name")
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	return importPath, pkgName
}

// printPreamble prints the package and import declarations for the file
func printPreamble(f *os.File, pkgName string, ps *param.PSet) {
	fmt.Fprintln(f, "package", pkgName)
	fmt.Fprint(f, `
/*
This code was generated by mkparamfilefunc
with parameters:
`)
	for _, pg := range ps.GetGroups() {
		for _, p := range pg.Params {
			whereSet := p.WhereSet()
			if len(whereSet) > 0 {
				fmt.Fprintln(f, whereSet[len(whereSet)-1])
			}
		}
	}
	fmt.Fprint(f, `

DO NOT EDIT
*/

import (
	"path/filepath"

	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v3/param"
	"github.com/nickwells/xdg.mod/xdg"
)

`)
}

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	ps.Add("output-file-name",
		psetter.Pathname{
			Value: &outputFileName,
			Checks: []check.String{
				check.StringHasSuffix(".go"),
				check.StringNot(
					check.StringHasSuffix("_test.go"),
					"a test file"),
			},
		},
		"set the name of the output file",
		param.AltName("o"),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("group", psetter.String{
		Value:  &groupName,
		Checks: []check.String{param.GroupNameCheck},
	},
		"sets the name of the group of parameters for which we are"+
			" building the functions. If this is not given then only"+
			" common config file functions will be generated. If a"+
			" group name is given then only the group-specific config"+
			" file functions will be generated. Additionally, unless"+
			" the output file name has been changed from the default,"+
			" the output file name will be adjusted to reflect the"+
			" group name.",
		param.AltName("g"),
		param.PostAction(setFileNameForGroup),
	)

	ps.Add("must-exist", psetter.Bool{Value: &mustExist},
		"the config file will be checked to ensure that it does exist and"+
			" it will be an error if it doesn't",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("must-exist-personal", psetter.Bool{Value: &mustExistPersonal},
		"the personal config file will be checked to ensure that it"+
			" does exist and it will be an error if it doesn't",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("must-exist-global", psetter.Bool{Value: &mustExistGlobal},
		"the global config file will be checked to ensure that it"+
			" does exist and it will be an error if it doesn't",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("no-file", psetter.Bool{Value: &dontMakeFile},
		"don't create the go file, instead just print the content to"+
			" standard out. This is useful for debugging or just to "+
			"see what would have been produced",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("base-dir-personal",
		psetter.String{
			Value:  &baseDirPersonal,
			Checks: []check.String{check.StringLenGT(0)},
		},
		"set the base directory in which the parameter file will be found."+
			" This value will be used in place of the XDG config directory"+
			" for personal config files."+
			" The sub-directories (derived from the import path) will still"+
			" be used",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("base-dir-global",
		psetter.String{
			Value:  &baseDirGlobal,
			Checks: []check.String{check.StringLenGT(0)},
		},
		"set the base directory in which the parameter file will be found."+
			" This value will be used in place of the XDG config directory"+
			" for global config files."+
			" The sub-directories (derived from the import path) will still"+
			" be used",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("funcs", psetter.Enum{
		Value: &whichFuncs,
		AVM: param.AVM{
			AllowedVals: param.AValMap{
				"all": "create all functions",
				"personalOnly": "create just the personal config file" +
					" setter function",
				"globalOnly": "create just the global config file" +
					" setter function",
			},
		},
	},
		"specify which of the two functions (the global or the personal)"+
			" should be created",
		param.Attrs(param.DontShowInStdUsage),
	)

	return nil
}

// setFileNameForGroup sets the outputFileName to the group variant unless it
// is already set to some non-default value
func setFileNameForGroup(_loc location.L, _ *param.ByName, _ []string) error {
	if outputFileName == dfltFileName {
		outputFileName = groupFileNameBase + groupName + ".go"
	}
	return nil
}
