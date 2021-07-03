package main

import (
	"strings"

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
					buildAct:    "run 'go build' in the directory",
					installAct:  "run 'go install' in the directory",
					generateAct: "run 'go generate' in the directory",
					printAct:    "print the directory name",
					contentAct:  "print any matching content",
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
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("install-arg",
			psetter.StrListAppender{Value: &fgd.installArgs},
			"set the arguments to be given to the go install command",
			param.AltNames("install-args", "args-install",
				"inst-args", "i-args"),
			param.PostAction(
				paction.SetMapIf(fgd.actions, installAct, true,
					paction.IsACommandLineParam)),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("build-arg",
			psetter.StrListAppender{Value: &fgd.buildArgs},
			"set the arguments to be given to the go build command",
			param.AltNames("build-args", "args-build",
				"b-args", "b-arg"),
			param.PostAction(
				paction.SetMapIf(fgd.actions, buildAct, true,
					paction.IsACommandLineParam)),
			param.Attrs(param.DontShowInStdUsage),
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

		ps.Add(paramNameHavingBuildTag, psetter.Nil{},
			"the directory must contain at least one file with"+
				" a build-tag."+
				" This adds a content"+
				" check with tag name: "+buildTagChecks.name,
			param.AltNames(
				"having-build-tags",
				"with-build-tags", "with-build-tag"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.PostAction(
				func(_ location.L, _ *param.ByName, _ []string) error {
					fgd.contentChecks[buildTagChecks.name] = buildTagChecks
					return nil
				}),
			param.SeeAlso(paramNameHavingContent, paramNameHavingGoGenerate),
			param.SeeNote(noteNameContentChecks),
		)

		ps.Add(paramNameHavingGoGenerate, psetter.Nil{},
			"the directory must contain at least one file with"+
				" a go:generate comment."+
				" This adds a content"+
				" check with tag name: "+gogenChecks.name,
			param.AltNames(
				"having-go-gen",
				"with-go-generate", "with-go-gen"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.PostAction(
				func(_ location.L, _ *param.ByName, _ []string) error {
					fgd.contentChecks[gogenChecks.name] = gogenChecks
					return nil
				}),
			param.SeeAlso(paramNameHavingContent, paramNameHavingBuildTag),
			param.SeeNote(noteNameContentChecks),
		)

		ps.Add(paramNameHavingContent, ContChkSetter{Value: &fgd.contentChecks},
			"the directory must contain at least one file with the following"+
				" content. Extra criteria can be set by adding"+
				" a period to the tag name and a part name.",
			param.AltNames("containing", "with-content"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.SeeAlso(paramNameHavingBuildTag, paramNameHavingGoGenerate),
			param.SeeNote(noteNameContentChecks),
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
	ps.AddExample(`findGoDirs -having-go-generate`,
		"This will find all the Go directories with go:generate comments."+
			" These are the directories where you might need to"+
			" run 'go generate' or where 'go generate' might have"+
			" changed the directory contents.")
	ps.AddExample(`findGoDirs -having-go-generate -do content`,
		"This will find all the Go directories with go:generate comments"+
			" and prints the matching lines.")

	return nil
}

// addNotes adds some notes to the help message
func addNotes(ps *param.PSet) error {
	ps.AddNote(noteNameContentChecks,
		"You can constrain the Go directories this command will find"+
			" by checking that a matching directory has at least one"+
			" file containing certain content."+
			"\n\n"+
			"This feature can by useful, for instance, to find directories"+
			" having files with go:generate comments so you know if you"+
			" need to run 'go generate' in them."+
			"\n\n"+
			"There are some common searches which have dedicated parameters"+
			" for setting them:"+
			" '"+paramNameHavingBuildTag+"' and"+
			" '"+paramNameHavingGoGenerate+"'."+
			" These have all the correct patterns preset and"+
			" it is recommended that you use these."+
			"\n\n"+
			"A content checker has at least a pattern for matching lines"+
			" but it can be extended to only check files matching a"+
			" pattern, to stop matching after a sertain pattern is matched"+
			" and to skip otherwise matching lines if they match an"+
			" additional pattern"+
			"\n\n"+
			"You can add these additional features using the"+
			" '"+paramNameHavingContent+"' parameter. You repeat the"+
			" checker name and add\n"+
			"    a period ('.'),"+
			"    a part name,"+
			"    an equals ('=')"+
			"    and the pattern for that part.\n"+
			"Valid part names are:\n"+strings.Join(checkerPartNames(), ", ")+
			"\n\n"+
			"Before you can add a part you must first create the checker"+
			" by giving a checker name and the match pattern"+
			" (no '.part' is needed)",
		param.NoteSeeParam(
			paramNameHavingBuildTag,
			paramNameHavingGoGenerate,
			paramNameHavingContent))

	return nil
}
