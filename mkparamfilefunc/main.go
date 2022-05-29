package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/param.mod/v5/param/psetter"
	"github.com/nickwells/twrap.mod/twrap"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Sat May 25 16:13:02 2019

const (
	dfltFileName      = "setConfigFile.go"
	groupFileNameBase = "setConfigFile"
)

var (
	mustExist         bool
	mustExistPersonal bool
	mustExistGlobal   bool
	privateFunc       bool
	makeFile          = true

	whichFuncs      = "all"
	outputFileName  = dfltFileName
	groupName       string
	baseDirPersonal string
	baseDirGlobal   string
)

func main() {
	ps := paramset.NewOrDie(
		gogen.AddParams(&outputFileName, &makeFile),
		verbose.AddParams,
		addParams,
		param.SetProgramDescription(
			"This creates a Go file defining functions which set the"+
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

	f := os.Stdout
	if makeFile {
		f = gogen.MakeFileOrDie(outputFileName)
		defer f.Close()
	}

	gogen.PrintPreambleOrDie(f, ps)
	gogen.PrintImports(f,
		"path/filepath",
		"github.com/nickwells/filecheck.mod/filecheck",
		"github.com/nickwells/param.mod/v5/param",
		"github.com/nickwells/xdg.mod/xdg")

	pkgName := gogen.GetPackageOrDie()
	dirs := strings.Split(gogen.GetImportPathOrDie(), "/")

	printFuncPersonal(f, groupName, pkgName, dirs)
	printFuncGlobal(f, groupName, pkgName, dirs)
}

// makeFuncNameGlobal generates the name of the function for setting the
// global config file that this program will write.
func makeFuncNameGlobal(groupName string) string {
	prefix := "Set"
	if privateFunc {
		prefix = "set"
	}

	fName := prefix + "GlobalConfigFile" + makeGroupSuffix(groupName)
	verbose.Print("Global function name: ", fName, "\n")

	return fName
}

// makeFuncNamePersonal generates the name of the function for setting the
// personal config file that this program will write.
func makeFuncNamePersonal(groupName string) string {
	prefix := "Set"
	if privateFunc {
		prefix = "set"
	}

	fName := prefix + "ConfigFile" + makeGroupSuffix(groupName)
	verbose.Print("Personal function name: ", fName, "\n")

	return fName
}

// makeGroupSuffix generates the suffix for the function name from the group
// name. It cleans up any characters in the name which are invalid characters
// in a Go function name
func makeGroupSuffix(groupName string) string {
	if groupName == "" {
		return ""
	}

	groupSuffix := "ForGroup"

	// Now split the group name into words and Titleise each word, adding it
	// to the group suffix
	groupName = strings.ReplaceAll(groupName, "-", ".")
	groupParts := strings.Split(groupName, ".")
	for _, part := range groupParts {
		groupSuffix += strings.Title(part)
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

// printFuncIntro prints the function comment and the func name and signature
func printFuncIntro(f io.Writer, name string) {
	twc := twrap.NewTWConfOrPanic(twrap.SetWriter(f))
	fmt.Fprintln(f)
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
func printFuncPersonal(f io.Writer, groupName, pkgName string, dirs []string) {
	if whichFuncs != "all" && whichFuncs != "personalOnly" {
		return
	}

	printFuncIntro(f, makeFuncNamePersonal(groupName))
	fmt.Fprint(f, "\n\tbaseDir := ")
	if baseDirPersonal != "" {
		fmt.Fprintf(f, "%q\n", baseDirPersonal)
	} else {
		fmt.Fprintln(f, "xdg.ConfigHome()")
	}

	printAddCF(f, dirs,
		makeAddCFName(pkgName, groupName),
		makeConfigFileName(groupName),
		mustExist || mustExistPersonal)
	printFuncEnd(f)
}

// printFuncGlobal writes out the function for setting a shared, global
// config file
func printFuncGlobal(f io.Writer, groupName, pkgName string, dirs []string) {
	if whichFuncs != "all" && whichFuncs != "globalOnly" {
		return
	}
	printFuncIntro(f, makeFuncNameGlobal(groupName))
	if baseDirGlobal != "" {
		fmt.Fprintf(f, "\n\tbaseDir := %q\n", baseDirGlobal)
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
func printAddCF(f io.Writer, dirs []string, funcName, cfgFName string, mustExist bool) {
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

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	ps.Add("group",
		psetter.String{
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
		param.AltNames("g"),
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
		AllowedVals: psetter.AllowedVals{
			"all": "create all functions",
			"personalOnly": "create just the personal config file" +
				" setter function",
			"globalOnly": "create just the global config file" +
				" setter function",
		},
	},
		"specify which of the two functions (the global or the personal)"+
			" should be created",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("private", psetter.Bool{Value: &privateFunc},
		"this will generate private (non-global) function names",
	)

	return nil
}

// setFileNameForGroup sets the outputFileName to the group variant unless it
// is already set to some non-default value
func setFileNameForGroup(_loc location.L, _ *param.ByName, _ []string) error {
	if outputFileName == dfltFileName {
		outputFileName = groupFileNameBase + makeGroupSuffix(groupName) + ".go"
	}
	return nil
}
