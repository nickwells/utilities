package main

import (
	"fmt"

	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/groupsetter.mod/groupsetter"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v7/paction"
	"github.com/nickwells/param.mod/v7/param"
	"github.com/nickwells/param.mod/v7/psetter"
)

const (
	paramNameHavingBuildTag   = "having-build-tag"
	paramNameHavingGoGenerate = "having-go-generate"
	paramNameShowCheckName    = "show-check-name"
	paramNameCheck            = "check"
	paramNameDir              = "dir"

	noteNameContentChecks = "Content Checks"
)

// makeCheckSetter creates a param Setter for a ContentCheck
func makeCheckSetter(prog *prog) *groupsetter.List[ContentCheck] {
	const (
		paramNameMatch   = "match"
		paramNameName    = "name"
		paramNameFile    = "filename-matches"
		paramNameNotFile = "filename-does-not-match"
		paramNameSkip    = "skip-if-matches"
		paramNameStop    = "stop-if-matches"
	)

	s := groupsetter.NewList(&prog.contentChecks)

	s.AddByPosParam(
		paramNameMatch,
		psetter.Regexp{
			Value: &s.InterimVal.matchPattern,
		},
		"the pattern to search files for."+
			" If a file is found matching this pattern")

	s.AddByNameParam(
		paramNameName,
		psetter.String[string]{
			Value: &s.InterimVal.name,
		},
		"a name to give to the check")

	s.AddByNameParam(
		paramNameFile,
		psetter.Regexp{
			Value: &s.InterimVal.filenamePattern,
		},
		"limit the files to be checked."+
			" Only files whose name matches this pattern will be checked",
		param.AltNames("filename", "file"))

	s.AddByNameParam(
		paramNameNotFile,
		psetter.Regexp{
			Value: &s.InterimVal.filenameSkipPattern,
		},
		"limit the files to be checked."+
			" Only files whose name does not match"+
			" this pattern will be checked",
		param.AltNames("not-filename", "not-file"))

	s.AddByNameParam(
		paramNameSkip,
		psetter.Regexp{
			Value: &s.InterimVal.skipPattern,
		},
		"lines matching this pattern are ignored"+
			" regardless of whether they would otherwise match.",
		param.AltNames("skip"))

	s.AddByNameParam(
		paramNameStop,
		psetter.Regexp{
			Value: &s.InterimVal.stopPattern,
		},
		"stop further checking."+
			" Once a line is found matching this pattern"+
			" no more lines in the file will be checked"+
			" by this checker.",
		param.AltNames("stop"))

	return s
}

// addParams will add parameters to the passed ParamSet
func addParams(prog *prog) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		dirProvisos := filecheck.DirExists()

		ps.Add(paramNameDir,
			psetter.PathnameListAppender{
				Value:       &prog.baseDirs,
				Expectation: dirProvisos,
			},
			"set the name of the directory to search from."+
				" If no directories are given, the current directory is"+
				" used. This parameter may be given more than once, each"+
				" time it is used the directory will be added to the"+
				" list of directories to search.",
			param.AltNames("dirs", "d"),
			param.Attrs(param.CommandLineOnly),
		)

		checkSetter := makeCheckSetter(prog)
		ps.Add(paramNameCheck, checkSetter,
			"set the additional checks to perform.",
			param.Attrs(param.CommandLineOnly),
		)

		ps.Add(paramNameShowCheckName,
			psetter.Bool{Value: &prog.showCheckName},
			"When reporting the checks that have passed"+
				" also show the named check ")

		ps.Add("actions",
			psetter.EnumMap[string]{
				Value: &prog.actions,
				AllowedVals: psetter.AllowedVals[string]{
					buildAct:    "run 'go build' in the directory",
					installAct:  "run 'go install' in the directory",
					testAct:     "run 'go test' in the directory",
					generateAct: "run 'go generate' in the directory",
					printAct:    "print the directory name",
					contentAct:  "print any matching content",
					filenameAct: "print files with matching content",
				},
			},
			"set the actions to perform when a Go directory matching"+
				" the supplied criteria is discovered",
			param.AltNames("a", "do"),
			param.Attrs(param.CommandLineOnly),
		)

		ps.Add("generate-arg",
			psetter.StrListAppender[string]{Value: &prog.generateArgs},
			"set the arguments to be given to the go generate command",
			param.AltNames("generate-args", "args-generate",
				"gen-args", "g-args", "g-arg"),
			param.PostAction(
				paction.SetMapValIf(prog.actions, generateAct, true,
					paction.IsACommandLineParam)),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("install-arg",
			psetter.StrListAppender[string]{Value: &prog.installArgs},
			"set the arguments to be given to the go install command",
			param.AltNames("install-args", "args-install",
				"inst-args", "i-args", "i-arg"),
			param.PostAction(
				paction.SetMapValIf(prog.actions, installAct, true,
					paction.IsACommandLineParam)),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("test-arg",
			psetter.StrListAppender[string]{Value: &prog.testArgs},
			"set the arguments to be given to the go test command",
			param.AltNames("test-args", "args-test",
				"t-args", "t-arg"),
			param.PostAction(
				paction.SetMapValIf(prog.actions, testAct, true,
					paction.IsACommandLineParam)),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("build-arg",
			psetter.StrListAppender[string]{Value: &prog.buildArgs},
			"set the arguments to be given to the go build command",
			param.AltNames("build-args", "args-build",
				"b-args", "b-arg"),
			param.PostAction(
				paction.SetMapValIf(prog.actions, buildAct, true,
					paction.IsACommandLineParam)),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("package-names",
			psetter.StrList[string]{Value: &prog.pkgNames},
			"set the names of packages to be matched. If this is not set then"+
				" any package name will be matched",
			param.AltNames("package", "pkg"),
		)

		ps.Add("having-files",
			psetter.StrList[string]{Value: &prog.filesWanted},
			"give a list of files that the directory must contain. All the"+
				" listed files must be present for the directory to be"+
				" matched.",
			param.AltNames("having", "with"),
			param.Attrs(param.CommandLineOnly),
		)

		ps.Add("missing-files",
			psetter.StrList[string]{Value: &prog.filesMissing},
			"give a list of files that the directory may not contain. Any of"+
				" the listed files may be absent for the directory to be"+
				" matched.",
			param.AltNames("not-having", "without"),
			param.Attrs(param.CommandLineOnly),
		)

		ps.Add(paramNameHavingBuildTag, psetter.Nil{},
			"the directory must contain at least one file with"+
				" a Go build-tag.",
			param.AltNames(
				"having-build-tags",
				"with-build-tags", "with-build-tag"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.PostAction(
				func(_ location.L, _ *param.BaseParam, _ []string) error {
					prog.contentChecks = append(prog.contentChecks, buildTagChecks)
					return nil
				}),
			param.SeeAlso(paramNameCheck, paramNameHavingGoGenerate),
			param.SeeNote(noteNameContentChecks),
		)

		ps.Add(paramNameHavingGoGenerate, psetter.Nil{},
			"the directory must contain at least one file with"+
				" a go:generate comment.",
			param.AltNames(
				"having-go-gen",
				"with-go-generate", "with-go-gen"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.PostAction(
				func(_ location.L, _ *param.BaseParam, _ []string) error {
					prog.contentChecks = append(prog.contentChecks, gogenChecks)
					return nil
				}),
			param.SeeAlso(paramNameCheck, paramNameHavingBuildTag),
			param.SeeNote(noteNameContentChecks),
		)

		ps.Add("no-action", psetter.Bool{Value: &prog.noAction},
			"this will stop any action from happening. Instead the action"+
				" functions will just report what they would have done.",
			param.AltNames("do-nothing"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		var skipDir string

		ps.Add("skip-dir", psetter.String[string]{Value: &skipDir},
			"exclude a directory with this name and skip any sub-directories."+
				" This parameter may be given more than once, each"+
				" time it is used the name will be added to the"+
				" list of directories to skip.",
			param.PostAction(paction.AppendStringVal(&prog.skipDirs, &skipDir)),
		)

		// set the default program action to print if no other action is
		// specified
		ps.AddFinalCheck(func() error {
			if len(prog.actions) == 0 {
				prog.actions[printAct] = true
			}

			return nil
		})

		// set the default ContentCheck names
		ps.AddFinalCheck(func() error {
			for i, cc := range prog.contentChecks {
				if cc.name == "" {
					cc.name = fmt.Sprintf("check-%d", i+1)
					prog.contentChecks[i] = cc
				}
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
	ps.AddExample(`findGoDirs -having-go-generate`,
		"This will find all the Go directories with go:generate comments."+
			" These are the directories where you might need to"+
			" run 'go generate' or where 'go generate' might have"+
			" changed the directory contents.")
	ps.AddExample(`findGoDirs -having-go-generate -do content`,
		"This will find all the Go directories with go:generate comments"+
			" and prints the matching lines.")
	ps.AddExample(`findGoDirs -check '//nolint:;name=nolint' -do content`,
		"This will find all the Go directories with"+
			" some file having a nolint comment"+
			" and prints the matching lines.")
	ps.AddExample(`findGoDirs -check '//nolint:;skip=errcheck' -do content`,
		"This will find all the Go directories with"+
			" some file having a nolint comment but where"+
			" the line matching //nolint doesn't also match errcheck"+
			" and prints the matching lines.")

	return nil
}
