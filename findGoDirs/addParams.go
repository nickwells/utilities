package main

import (
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
	"github.com/nickwells/utilities/internal/stdparams"
)

const (
	paramNameHavingContent    = "having-content"
	paramNameHavingBuildTag   = "having-build-tag"
	paramNameHavingGoGenerate = "having-go-generate"

	noteNameContentChecks = "Content Checks"
)

// addParams will add parameters to the passed ParamSet
func addParams(fgd *findGoDirs) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		var dir string
		ps.Add("dir", psetter.Pathname{Value: &dir},
			"set the name of the directory to search from."+
				" If no directories are given, the current directory is"+
				" used. This parameter may be given more than once, each"+
				" time it is used the directory will be added to the"+
				" list of directories to search.",
			param.AltNames("dirs", "d"),
			param.PostAction(paction.AppendStringVal(&fgd.baseDirs, &dir)),
			param.Attrs(param.CommandLineOnly),
		)

		ps.Add("actions",
			psetter.EnumMap{
				Value: &fgd.actions,
				AllowedVals: psetter.AllowedVals{
					buildAct:    "build the command/package (go build)",
					installAct:  "install the command/package (go install)",
					generateAct: "auto-generate files, if any (go generate)",
					printAct:    "print the directory name",
					contentAct:  "print the directory name & matching content",
				},
			},
			"set the actions to perform when a Go directory matching"+
				" the supplied criteral is discovered",
			param.AltNames("a", "do"),
			param.Attrs(param.CommandLineOnly),
		)

		ps.Add("generate-arg",
			psetter.StrListAppender{Value: &fgd.generateArgs},
			"set the arguments to be given to the go generate command",
			param.AltNames("generate-args", "args-generate",
				"gen-args", "g-args"),
			param.PostAction(
				paction.SetMapIf(fgd.actions, generateAct, true,
					paction.IsACommandLineParam)),
		)

		ps.Add("install-arg",
			psetter.StrListAppender{Value: &fgd.installArgs},
			"set the arguments to be given to the go install command",
			param.AltNames("install-args", "args-install",
				"inst-args", "i-args"),
			param.PostAction(
				paction.SetMapIf(fgd.actions, installAct, true,
					paction.IsACommandLineParam)),
		)

		ps.Add("build-arg",
			psetter.StrListAppender{Value: &fgd.buildArgs},
			"set the arguments to be given to the go build command",
			param.AltNames("build-args", "args-build",
				"b-args", "b-arg"),
			param.PostAction(
				paction.SetMapIf(fgd.actions, buildAct, true,
					paction.IsACommandLineParam)),
		)

		ps.Add("package-names",
			psetter.StrList{Value: &fgd.pkgNames},
			"set the names of packages to be matched. If this is not set then"+
				" any package name will be matched",
			param.AltNames("package", "pkg"),
		)

		ps.Add("having-files", psetter.StrList{Value: &fgd.filesWanted},
			"give a list of files that the directory must contain. All the"+
				" listed files must be present for the directory to be"+
				" matched.",
			param.AltNames("having", "with"),
			param.Attrs(param.CommandLineOnly),
		)

		ps.Add("missing-files", psetter.StrList{Value: &fgd.filesMissing},
			"give a list of files that the directory may not contain. Any of"+
				" the listed files may be absent for the directory to be"+
				" matched.",
			param.AltNames("not-having", "without"),
			param.Attrs(param.CommandLineOnly),
		)

		ps.Add("having-build-tag", psetter.Nil{},
			"the directory must contain some file with a build-tag.",
			param.AltNames(
				"having-build-tags",
				"with-build-tags", "with-build-tag"),
			param.Attrs(param.CommandLineOnly),
			param.PostAction(
				func(_ location.L, _ *param.ByName, _ []string) error {
					fgd.contentChecks[buildTagChecks.name] = buildTagChecks
					return nil
				}),
		)

		ps.Add("having-go-generate", psetter.Nil{},
			"the directory must contain some file with a go:generate comment.",
			param.AltNames(
				"having-go-gen",
				"with-go-generate", "with-go-gen"),
			param.Attrs(param.CommandLineOnly),
			param.PostAction(
				func(_ location.L, _ *param.ByName, _ []string) error {
					fgd.contentChecks[gogenChecks.name] = gogenChecks
					return nil
				}),
		)

		ps.Add("no-action", psetter.Bool{Value: &fgd.noAction},
			"this will stop any action from happening. Instead the action"+
				" functions will just report what they would have done.",
			param.AltNames("do-nothing"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		stdparams.AddTiming(ps, fgd.dbgStack)

		var skipDir string
		ps.Add("skip-dir", psetter.String{Value: &skipDir},
			"exclude a directory with this name and skip any sub-directories."+
				" This parameter may be given more than once, each"+
				" time it is used the name will be added to the"+
				" list of directories to skip.",
			param.PostAction(paction.AppendStringVal(&fgd.skipDirs, &skipDir)),
		)

		ps.AddFinalCheck(func() error {
			if len(fgd.actions) == 0 {
				fgd.actions[printAct] = true
			}

			if len(fgd.baseDirs) == 0 {
				fgd.baseDirs = []string{"."}
			}
			return nil
		})

		return nil
	}
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
