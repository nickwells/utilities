package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/dirsearch.mod/v2/dirsearch"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Thu Jun 11 12:43:33 2020

// reportNoAction reports the action being skipped in a common format
func reportNoAction(actionName, dirName string) {
	fmt.Printf("%-20.20s : %s (no-action: skipping)\n", actionName, dirName)
}

// doPrint will print the name
func doPrint(prog *prog, dirName string) {
	if prog.noAction {
		reportNoAction("print", dirName)
		return
	}

	fmt.Println(dirName)
}

// doContent will show the lines in the files in the directory that match
// the content checks
func doContent(prog *prog, dirName string) {
	defer prog.dbgStack.Start("doContent",
		"Print matching content in : "+dirName)()

	if prog.noAction {
		reportNoAction("content", dirName)
		return
	}

	keys := slices.Sorted(maps.Keys(prog.dirContent[dirName]))

	maxKeyLen := 0

	if prog.showCheckName {
		for _, k := range keys {
			maxKeyLen = max(len(k), maxKeyLen)
		}
	}

	for _, k := range keys {
		prevSource := ""

		for _, match := range prog.dirContent[dirName][k] {
			source := match.Source()
			if prog.showCheckName {
				source = fmt.Sprintf("%*.*s: %s",
					maxKeyLen, maxKeyLen, k, source)
			}

			if source != prevSource {
				fmt.Printf("%s:", source)
			} else {
				fmt.Printf("%s:", strings.Repeat(" ", len(prevSource)))
			}

			content, _ := match.Content()
			fmt.Printf("%d:%s\n", match.Idx(), content)

			prevSource = source
		}
	}
}

// doFilenames will show the names of the files in the directories that match
// the content checks
func doFilenames(prog *prog, dirName string) {
	defer prog.dbgStack.Start("doFilenames",
		"Print files with matching content in : "+dirName)()

	if prog.noAction {
		reportNoAction("filenames", dirName)
		return
	}

	keys := slices.Sorted(maps.Keys(prog.dirContent[dirName]))

	for _, k := range keys {
		for _, match := range prog.dirContent[dirName][k] {
			fmt.Println(match.Source())
		}
	}
}

// doBuild will run go build
func doBuild(prog *prog, dirName string) {
	prog.doGoCommand(dirName, "build", prog.buildArgs)
}

// doTest will run go test
func doTest(prog *prog, dirName string) {
	prog.doGoCommand(dirName, "test", prog.testArgs)
}

// doInstall will run go install
func doInstall(prog *prog, dirName string) {
	prog.doGoCommand(dirName, "install", prog.installArgs)
}

// doGenerate will run go generate
func doGenerate(prog *prog, dirName string) {
	prog.doGoCommand(dirName, "generate", prog.generateArgs)
}

// doGoCommand will run the Go subcommand with the passed args
func (prog *prog) doGoCommand(dirName, command string, cmdArgs []string) {
	defer prog.dbgStack.Start("doGoCommand", "In : "+dirName)()

	intro := prog.dbgStack.Tag()

	if prog.noAction {
		reportNoAction("go "+command, dirName)
		return
	}

	args := []string{command}
	args = append(args, cmdArgs...)

	verbose.Println(intro, "go "+strings.Join(args, " "))
	gogen.ExecGoCmd(gogen.ShowCmdIO, args...)
}

// gatherDirs will process all the directories from the command line and
// check them for existence. It will report any problems found.
func (prog *prog) gatherDirs(dirs []string) {
	dirProvisos := filecheck.DirExists()

	for _, dirName := range dirs {
		if err := dirProvisos.StatusCheck(dirName); err != nil {
			fmt.Fprintf(os.Stderr, "bad directory: %s\n", err)
			continue
		}

		prog.baseDirs = append(prog.baseDirs, dirName)
	}
}

func main() {
	prog := newProg()
	ps := makeParamSet(prog)

	ps.Parse()

	prog.gatherDirs(ps.TrailingParams())

	defer prog.dbgStack.Start("main", os.Args[0])()

	sortedDirs := prog.findMatchingDirs()
	for _, d := range sortedDirs {
		prog.onMatchDo(d)
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
func (prog *prog) findMatchingDirs() []string {
	defer prog.dbgStack.Start("findMatchingDirs",
		"Find dirs matching criteria")()

	var dirs []string

	dirChecks := []check.FileInfo{
		check.FileInfoName(check.Not(
			check.ValEQ("testdata"),
			"Ignore any directory called testdata")),
		check.FileInfoName(check.Not(
			check.StringHasPrefix[string]("_"),
			"Ignore directories with name starting with '_'")),
		check.FileInfoName(
			check.Or(
				check.Not(
					check.StringHasPrefix[string]("."),
					"Ignore hidden directories (including .git)"),
				check.ValEQ("."),
				check.ValEQ(".."),
			)),
	}

	for _, skipDir := range prog.skipDirs {
		dirChecks = append(dirChecks, check.FileInfoName(check.Not(
			check.ValEQ(skipDir),
			"Ignore any directory called "+skipDir)))
	}

	fileChecks := []check.FileInfo{check.FileInfoIsDir}
	fileChecks = append(fileChecks, dirChecks...)

	if len(prog.baseDirs) == 0 {
		prog.baseDirs = []string{"."}
	}

	for _, dir := range prog.baseDirs {
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

	return slices.Compact(dirs)
}

// onMatchDo performs the actions if the directory is a go package directory
// meeting the criteria
func (prog *prog) onMatchDo(dir string) {
	defer prog.dbgStack.Start("onMatchDo", "Act on matching dir: "+dir)()

	intro := prog.dbgStack.Tag()

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

	if !prog.pkgMatches(pkg) {
		verbose.Println(intro, " Skipping: Wrong package")
		return
	}

	if !hasEntries(prog.filesWanted) {
		verbose.Println(intro, " Skipping: missing files")
		return
	}

	if len(prog.filesMissing) > 0 && hasEntries(prog.filesMissing) {
		verbose.Println(intro, " Skipping: has unwanted files")
		return
	}

	if !prog.hasRequiredContent(dir) {
		delete(prog.dirContent, dir)
		verbose.Println(intro, " Skipping: missing required content")

		return
	}

	// We force the order that actions take place - we should always generate
	// any files before building or installing (if generate is requested)
	for _, a := range []string{
		printAct, contentAct, filenameAct,
		generateAct, testAct, buildAct, installAct,
	} {
		if prog.actions[a] {
			verbose.Println(intro, " Doing: "+a)
			prog.actionFuncs[a](prog, dir)
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
func (prog *prog) pkgMatches(pkg string) bool {
	if len(prog.pkgNames) == 0 { // any name matches
		return true
	}

	if slices.Contains(prog.pkgNames, pkg) {
		return true
	}

	return false // no name matches
}

// hasEntries will check to see if any of the listed directory entries exists
// in the current directory and return false if any of them are missing. It
// will only return true if all the entries are found in the directory
func hasEntries(entries []string) bool {
	if len(entries) == 0 {
		return true
	}

	dirEntries, err := os.ReadDir(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot read the directory:", err)
		return false
	}

	for _, entryName := range entries {
		if !entryFound(entryName, dirEntries) {
			return false
		}
	}

	return true
}

// entryFound will return true if the name is in the list of directory
// entries
func entryFound(name string, entries []fs.DirEntry) bool {
	for _, f := range entries {
		if f.Name() == name {
			return true
		}
	}

	return false
}

// hasRequiredContent will check to see if any of the files in the current
// directory has the required content and return false if any of the required
// content is not in any file. It will only return true if all the required
// content is present in at least one of the files in the directory. In any
// case the map of content discovered for the given directory will have been
// populated.
func (prog *prog) hasRequiredContent(dir string) bool {
	prog.dirContent[dir] = contentMap{}

	if len(prog.contentChecks) == 0 {
		return true
	}

	dirEntries, err := os.ReadDir(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot read the directory:", err)
		return false
	}

	for _, entry := range dirEntries {
		if !entry.Type().IsRegular() {
			continue
		}

		err := prog.checkContent(dir, entry.Name())
		if err != nil {
			break
		}
	}

	return len(prog.dirContent[dir]) == len(prog.contentChecks)
}

// checkContent opens the file and finds any content matching the checks,
// writing it into the contentMap. It returns a non-nil error if the file
// cannot be opened.
func (prog *prog) checkContent(dir, fname string) error {
	statusChecks := []StatusCheck{}

	for _, c := range prog.contentChecks {
		if c.FileNameOK(fname) {
			statusChecks = append(statusChecks, StatusCheck{chk: &c})
		}
	}

	if len(statusChecks) == 0 {
		return nil
	}

	pathname := filepath.Join(dir, fname) // for error reporting

	// Use fname not pathname - the open is relative to the current directory
	// so if the directory name is a relative path (containing '..') then
	// the pathname will not necessarily be available when the fname is.
	// This is because the process's working directory will have changed as
	// the process walks the tree of directories searching for matches.
	f, err := os.Open(fname) //nolint:gosec
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't open %q: %v\n", pathname, err)
		return err
	}

	defer f.Close()

	loc := location.New(pathname)
	s := bufio.NewScanner(f)

	for s.Scan() {
		loc.Incr()

		allChecksComplete := true

		for _, sc := range statusChecks {
			if sc.CheckLine(s.Text()) {
				locCopy := *loc
				locCopy.SetContent(s.Text())
				prog.dirContent[dir][sc.chk.name] = append(
					prog.dirContent[dir][sc.chk.name], locCopy)
			}

			if !sc.stopped {
				allChecksComplete = false
			}
		}

		if allChecksComplete {
			break
		}
	}

	return nil
}
