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
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Thu Jun 11 12:43:33 2020

// doPrint will print the name
func doPrint(fgd *prog, name string) {
	if fgd.noAction {
		fmt.Printf("%-20.20s : %s\n", "print", name)
		return
	}

	fmt.Println(name)
}

// doContent will show the lines in the files in the directory that match
// the content checks
func doContent(fgd *prog, name string) {
	defer fgd.dbgStack.Start("doContent", "Print matching content in : "+name)()

	if fgd.noAction {
		fmt.Printf("%-20.20s : %s\n", "content", name)
		return
	}

	keys := slices.Sorted(maps.Keys(fgd.dirContent[name]))

	for _, k := range keys {
		for _, match := range fgd.dirContent[name][k] {
			fmt.Println(match.String())
		}
	}
}

// doFilenames will show the names of the files in the directories that match
// the content checks
func doFilenames(fgd *prog, name string) {
	defer fgd.dbgStack.Start("doFilenames",
		"Print files with matching content in : "+name)()

	if fgd.noAction {
		fmt.Printf("%-20.20s : %s\n", "filenames", name)
		return
	}

	keys := slices.Sorted(maps.Keys(fgd.dirContent[name]))

	for _, k := range keys {
		for _, match := range fgd.dirContent[name][k] {
			fmt.Println(match.Source())
		}
	}
}

// doBuild will run go build
func doBuild(fgd *prog, name string) {
	fgd.doGoCommand(name, "build", fgd.buildArgs)
}

// doTest will run go test
func doTest(fgd *prog, name string) {
	fgd.doGoCommand(name, "test", fgd.testArgs)
}

// doInstall will run go install
func doInstall(fgd *prog, name string) {
	fgd.doGoCommand(name, "install", fgd.installArgs)
}

// doGenerate will run go generate
func doGenerate(fgd *prog, name string) {
	fgd.doGoCommand(name, "generate", fgd.generateArgs)
}

// doGoCommand will run the Go subcommand with the passed args
func (fgd *prog) doGoCommand(name, command string, cmdArgs []string) {
	defer fgd.dbgStack.Start("doGoCommand", "In : "+name)()
	intro := fgd.dbgStack.Tag()

	if fgd.noAction {
		fmt.Printf("%-20.20s : %s\n", "go "+command, name)
		return
	}

	args := []string{command}
	args = append(args, cmdArgs...)

	verbose.Println(intro, "go "+strings.Join(args, " "))
	gogen.ExecGoCmd(gogen.ShowCmdIO, args...)
}

func main() {
	prog := newProg()
	ps := makeParamSet(prog)

	ps.Parse()

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
func (fgd *prog) findMatchingDirs() []string {
	defer fgd.dbgStack.Start("findMatchingDirs",
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

	for _, skipDir := range fgd.skipDirs {
		dirChecks = append(dirChecks, check.FileInfoName(check.Not(
			check.ValEQ(skipDir),
			"Ignore any directory called "+skipDir)))
	}

	fileChecks := []check.FileInfo{check.FileInfoIsDir}
	fileChecks = append(fileChecks, dirChecks...)

	if len(fgd.baseDirs) == 0 {
		fgd.baseDirs = []string{"."}
	}

	for _, dir := range fgd.baseDirs {
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
func (fgd *prog) onMatchDo(dir string) {
	defer fgd.dbgStack.Start("onMatchDo", "Act on matching dir: "+dir)()
	intro := fgd.dbgStack.Tag()

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

	if !fgd.pkgMatches(pkg) {
		verbose.Println(intro, " Skipping: Wrong package")
		return
	}

	if !hasEntries(fgd.filesWanted) {
		verbose.Println(intro, " Skipping: missing files")
		return
	}

	if len(fgd.filesMissing) > 0 && hasEntries(fgd.filesMissing) {
		verbose.Println(intro, " Skipping: has unwanted files")
		return
	}

	if !fgd.hasRequiredContent(dir) {
		delete(fgd.dirContent, dir)
		verbose.Println(intro, " Skipping: missing required content")

		return
	}

	// We force the order that actions take place - we should always generate
	// any files before building or installing (if generate is requested)
	for _, a := range []string{
		printAct, contentAct, filenameAct,
		generateAct, testAct, buildAct, installAct,
	} {
		if fgd.actions[a] {
			verbose.Println(intro, " Doing: "+a)
			fgd.actionFuncs[a](fgd, dir)
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
func (fgd *prog) pkgMatches(pkg string) bool {
	if len(fgd.pkgNames) == 0 { // any name matches
		return true
	}

	if slices.Contains(fgd.pkgNames, pkg) {
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
func (fgd *prog) hasRequiredContent(dir string) bool {
	fgd.dirContent[dir] = contentMap{}

	if len(fgd.contentChecks) == 0 {
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

		err := fgd.checkContent(dir, entry.Name())
		if err != nil {
			break
		}
	}

	return len(fgd.dirContent[dir]) == len(fgd.contentChecks)
}

// checkContent opens the file and finds any content matching the checks,
// writing it into the contentMap
func (fgd *prog) checkContent(dir, fname string) error {
	statusChecks := []StatusCheck{}

	for _, c := range fgd.contentChecks {
		if c.FileNameOK(fname) {
			statusChecks = append(statusChecks, StatusCheck{chk: c})
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
				fgd.dirContent[dir][sc.chk.name] = append(
					fgd.dirContent[dir][sc.chk.name], locCopy)
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
