package main

import (
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	ps.Add("dir", psetter.Pathname{Value: &dir},
		"set the name of the directory to search from",
		param.AltName("d"),
	)

	ps.Add("actions",
		psetter.EnumMap{
			Value: &actions,
			AllowedVals: psetter.AllowedVals{
				buildAct:    "build the command/package (go build)",
				installAct:  "install the command/package (go install)",
				generateAct: "auto-generate files, if any (go generate)",
				printAct:    "print the directory name",
			},
		},
		"set the actions to perform when a Go command directory is discovered",
		param.AltName("a"),
		param.AltName("do"),
	)

	ps.Add("generate-args", psetter.StrListAppender{Value: &generateArgs},
		"set the arguments to be given to the go generate command",
		param.AltName("gen-args"),
		param.PostAction(paction.SetMapIf(actions, generateAct, true,
			paction.IsACommandLineParam)),
	)

	ps.Add("install-args", psetter.StrListAppender{Value: &installArgs},
		"set the arguments to be given to the go install command",
		param.AltName("inst-args"),
		param.PostAction(paction.SetMapIf(actions, installAct, true,
			paction.IsACommandLineParam)),
	)

	ps.Add("build-args", psetter.StrListAppender{Value: &buildArgs},
		"set the arguments to be given to the go build command",
		param.PostAction(paction.SetMapIf(actions, buildAct, true,
			paction.IsACommandLineParam)),
	)

	ps.Add("package-names", psetter.StrList{Value: &pkgNames},
		"set the names of packages to be matched. If this is not set then"+
			" any package name will be matched",
		param.AltName("package"),
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

	ps.AddFinalCheck(func() error {
		if len(actions) == 0 {
			actions[printAct] = true
		}
		return nil
	})

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
	ps.AddExample(`findGoDirs -pkg main -not-having .gitignore`,
		"This will find all the Go directories with code for building"+
			" commands that don't have a .gitignore  file. Note that when"+
			" you run go build in the directory you will get an"+
			" executable built in the directory which you don't want to"+
			" check in to git and so you need it to be ignored.")

	return nil
}
