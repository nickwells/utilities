package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/cli.mod/cli/responder"
	"github.com/nickwells/dirsearch.mod/v2/dirsearch"
	"github.com/nickwells/english.mod/english"
	"github.com/nickwells/mathutil.mod/v2/mathutil"
	"github.com/nickwells/twrap.mod/twrap"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Wed Oct 23 18:05:24 2019

const (
	dfltExtension = ".orig"
	dfltDir       = "."
	dfltDiffCmd   = "diff"
	dfltLessCmd   = "less"

	filenameIndent = 8
)

// dupAction records the action to perform on a duplicate file
type dupAction string

// these constants represent the allowed values of a dupAction variable
const (
	daDelete = dupAction("delete")
	daQuery  = dupAction("query")
	daKeep   = dupAction("keep")
)

// cmpAction records the action to perform on a comparable file
type cmpAction string

// these constants represent the allowed values of a cmpAction variable
const (
	caShowDiff  = cmpAction("show-diffs")
	caQuery     = cmpAction("query")
	caKeepAll   = cmpAction("keep-all")
	caDeleteAll = cmpAction("delete-all")
	caRevertAll = cmpAction("revert-all")
)

// cmdInfo records the command name and the parameters to be supplied
type cmdInfo struct {
	name   string
	params []string
}

// prog holds program parameters and status
type prog struct {
	stack *verbose.Stack

	// parameters
	searchDir     string
	searchSubDirs bool
	fileExtension string

	diff cmdInfo
	less cmdInfo

	showDiffR  responder.Responder
	deleteDupR responder.Responder
	postDiffR  responder.Responder

	dupAction dupAction
	cmpAction cmpAction

	// display
	twc    *twrap.TWConf
	indent int

	// record the behaviour and outcomes
	status Status
}

// newProg returns a new Prog instance with the default values set
func newProg() *prog {
	return &prog{
		stack: &verbose.Stack{},

		searchDir:     dfltDir,
		searchSubDirs: true,
		fileExtension: dfltExtension,

		diff: cmdInfo{name: dfltDiffCmd},
		less: cmdInfo{name: dfltLessCmd},

		dupAction: daQuery,
		cmpAction: caQuery,

		twc: twrap.NewTWConfOrPanic(),

		status: InitStatus(),
	}
}

// setResponders sets the responders on the Prog
func (prog *prog) setResponders() {
	prog.showDiffR = responder.NewOrPanic(
		"Show differences",
		map[rune]string{
			'y': "to show differences",
			'n': "to skip this file",
			'd': "delete this and all subsequent files" +
				" with extension " + prog.fileExtension,
			'r': "revert this and all subsequent base files to" +
				" the contents of the files with extension " +
				prog.fileExtension,
			'q': "to quit, keeping all subsequent files",
		},
		responder.SetDefault('y'))

	prog.postDiffR = responder.NewOrPanic(
		"delete file",
		map[rune]string{
			'y': "to delete this file",
			'n': "to keep this file",
			'r': "to revert the base file to this content",
			'q': "to quit, keeping all subsequent files",
		},
		responder.SetDefault('n'))

	prog.deleteDupR = responder.NewOrPanic(
		"delete all duplicate files",
		map[rune]string{
			'y': "to delete all duplicates files with extension " +
				prog.fileExtension,
			'n': "to keep these duplicates",
		},
		responder.SetDefault('y'))
}

// badFile holds details of errors detected when processng files
type badFile struct {
	name    string
	problem string
}

func main() {
	prog := newProg()
	ps := makeParamSet(prog)

	ps.Parse()
	prog.setResponders()

	filenames, duplicates, badFiles, errs := prog.getFiles()

	if len(errs) != 0 {
		fmt.Fprintln(os.Stderr, "Couldn't find the entries:")

		for _, err := range errs {
			fmt.Fprintln(os.Stderr, "\t", err)
		}

		os.Exit(1)
	}

	prog.showBadFiles(badFiles)

	prog.showDuplicateFiles(duplicates)
	prog.processDuplicateFiles(duplicates)

	prog.showComparableFiles(filenames)
	prog.processComparableFiles(filenames)

	prog.status.Report()
}

// shortNames returns a list of filenames with the search directory name
// removed. It also returns the maximum length of the names
func (prog prog) shortNames(filenames []string) ([]string, int) {
	shortNames := make([]string, 0, len(filenames))
	maxLen := 0

	for _, fn := range filenames {
		shortName := strings.TrimPrefix(fn,
			prog.searchDir+string(os.PathSeparator))
		if len(shortName) > maxLen {
			maxLen = len(shortName)
		}

		shortNames = append(shortNames, shortName)
	}

	return shortNames, maxLen
}

// showBadFiles displays the list of files for which problems were detected
func (prog *prog) showBadFiles(badFiles []badFile) {
	if len(badFiles) == 0 {
		return
	}

	defer prog.stack.Start("showBadFiles", "Start")()

	prog.status.badFile.total = len(badFiles)

	filenames := make([]string, 0, len(badFiles))

	for _, fe := range badFiles {
		filenames = append(filenames, fe.name)
	}

	shortNames, maxNameLen := prog.shortNames(filenames)
	reportFiles(len(filenames), "problem", "found")

	fmt.Println("in", prog.searchDir)

	for i, name := range shortNames {
		fmt.Printf("%s%*s - %s\n",
			strings.Repeat(" ", filenameIndent),
			maxNameLen,
			name, badFiles[i].problem)
	}

	fmt.Println()
}

// showDuplicateFiles displays the list of duplicate files and prompts the user
// to delete them
func (prog *prog) showDuplicateFiles(dupFiles []string) {
	if len(dupFiles) == 0 {
		return
	}

	defer prog.stack.Start("showDuplicateFiles", "Start")()

	prog.status.dupFile.total = len(dupFiles)

	shortNames, _ := prog.shortNames(dupFiles)
	reportFiles(len(dupFiles), "duplicate", "found")
	fmt.Println("in", prog.searchDir)
	prog.twc.NoRptPathList(shortNames, filenameIndent)
}

// processDuplicateFiles checks the duplicate action and then either deletes
// all the duplicates, keeps them all or queries the user.
func (prog *prog) processDuplicateFiles(dupFiles []string) {
	if len(dupFiles) == 0 {
		return
	}

	defer prog.stack.Start("processDuplicateFiles", "Start")()

	switch prog.dupAction {
	case daDelete:
		prog.deleteAllFiles(dupFiles, &prog.status.dupFile)
	case daQuery:
		if prog.queryDeleteDuplicates() == 'y' {
			prog.deleteAllFiles(dupFiles, &prog.status.dupFile)
		}
	}

	fmt.Println()
}

// showComparableFiles loops over the files prompting the user to compare
// each one with the new instance and then asking if the file should be
// deleted or the new file reverted.
func (prog *prog) showComparableFiles(cmpFiles []string) {
	if len(cmpFiles) == 0 {
		return
	}

	defer prog.stack.Start("showComparableFiles", "Start")()

	prog.status.cmpFile.total = len(cmpFiles)

	shortNames, _ := prog.shortNames(cmpFiles)

	reportFiles(len(cmpFiles), "comparable", "found")
	fmt.Println("in", prog.searchDir)
	prog.twc.IdxNoRptPathList(shortNames, filenameIndent)
}

// processComparableFiles checks the comparable action and then, for each
// file it will either show the difference automatically or else prompt the
// user whether or not to show the difference between the file and its
// corresponding other. One of the choices of action is to delete all the
// files, to revert them to their original state or to keep both members of
// the pair.
func (prog *prog) processComparableFiles(cmpFiles []string) {
	if len(cmpFiles) == 0 {
		return
	}

	defer prog.stack.Start("processComparableFiles", "Start")()

	shortNames, maxNameLen := prog.shortNames(cmpFiles)

	digits := mathutil.Digits(int64(len(cmpFiles)) + 1)
	nameFormat := fmt.Sprintf("    (%%%dd / %%%dd) %%%d.%ds: ",
		digits, digits, maxNameLen, maxNameLen)
	prog.indent = len(fmt.Sprintf(nameFormat, 0, 0, ""))

loop:
	for i, nameOrig := range cmpFiles {
		nameNew := strings.TrimSuffix(nameOrig, prog.fileExtension)

		switch prog.cmpAction {
		case caQuery:
			fmt.Printf(nameFormat, i+1, len(cmpFiles), shortNames[i])
			if prog.queryShowDiff() {
				prog.showDiff(nameOrig, nameNew, &prog.status.cmpFile)
			}
		case caShowDiff:
			fmt.Printf(nameFormat, i+1, len(cmpFiles), shortNames[i])
			prog.showDiff(nameOrig, nameNew, &prog.status.cmpFile)
		}

		// queryShowDiff can change the value of prog.cmpAction so switch again
		switch prog.cmpAction {
		case caRevertAll:
			prog.revertFile(nameOrig, nameNew, &prog.status.cmpFile)
		case caDeleteAll:
			prog.deleteFile(nameOrig, &prog.status.cmpFile)
		case caKeepAll:
			filesRemaining := len(cmpFiles) - i
			reportFiles(filesRemaining, "comparable", "kept")
			break loop
		}
	}
	fmt.Println()
}

// fileContentsDiffer returns true if the file contents differ
func fileContentsDiffer(f1, f2 []byte) bool {
	if len(f1) != len(f2) {
		return true
	}

	for i, b1 := range f1 {
		if b1 != f2[i] {
			return true
		}
	}

	return false
}

// queryDeleteDuplicates returns true if the user responds that the
// duplicates should be deleted
func (prog *prog) queryDeleteDuplicates() rune {
	response := prog.deleteDupR.GetResponseIndentOrDie(0, prog.indent)

	fmt.Println()

	return response
}

// showDiff shows the differences between the new file and the original and
// then queries for further actions.
func (prog *prog) showDiff(nameOrig, nameNew string, counts *Counts) {
	err := prog.diffs(nameOrig, nameNew)
	if err != nil {
		prog.twc.Wrap(fmt.Sprintf("Error: %v", err), prog.indent)

		counts.cmpErrs++

		return
	}

	counts.compared++

	prog.queryDeleteFile(nameOrig, nameNew)
}

// queryShowDiff asks if the differences between the new file and the
// original should be shown and then acts accordingly, reporting any errors
// found.
func (prog *prog) queryShowDiff() bool {
	response := prog.showDiffR.GetResponseIndentOrDie(0, prog.indent)

	fmt.Println()

	switch response {
	case 'y':
		return true
	case 'n':
		prog.verboseMsg("Skipping...")
	case 'r':
		prog.setRevertAll()
	case 'd':
		prog.setDeleteAll()
	case 'q':
		prog.setKeepAll()
	}

	return false
}

// setRevertAll sets the comparison action to Revert-All
func (prog *prog) setRevertAll() {
	prog.verboseMsg("Reverting all...")
	prog.cmpAction = caRevertAll
}

// setDeleteAll sets the comparison action to Delete-All
func (prog *prog) setDeleteAll() {
	prog.verboseMsg("Deleting all...")
	prog.cmpAction = caDeleteAll
}

// setKeepAll sets the comparison action to Keep-All
func (prog *prog) setKeepAll() {
	prog.verboseMsg("All remaining files will be kept...")
	prog.cmpAction = caKeepAll
}

// queryDeleteFile asks if the file should be deleted and then acts
// accordingly, reporting any errors found.
func (prog *prog) queryDeleteFile(nameOrig, nameNew string) {
	response := prog.postDiffR.GetResponseIndentOrDie(prog.indent, prog.indent)

	fmt.Println()

	switch response {
	case 'y':
		prog.deleteFile(nameOrig, &prog.status.cmpFile)
	case 'r':
		prog.revertFile(nameOrig, nameNew, &prog.status.cmpFile)
	case 'q':
		prog.setKeepAll()
	}
}

// deleteAllFiles deletes all of the given files
func (prog *prog) deleteAllFiles(filenames []string, count *Counts) {
	for _, fName := range filenames {
		prog.deleteFile(fName, count)
	}

	reportFiles(count.deleted, count.name, "deleted")
	reportFiles(count.delErrs, count.name, "could not be deleted")
}

// deleteFile deletes the named file, reporting any errors
func (prog *prog) deleteFile(name string, counts *Counts) {
	prog.verboseMsg("Deleting " + name + "...")

	err := os.Remove(name)
	if err != nil {
		prog.twc.Wrap(
			fmt.Sprintf("Couldn't delete the file: %v", err),
			prog.indent)

		counts.delErrs++

		return
	}

	prog.verboseMsg(name + " deleted")

	counts.deleted++
}

// revertFile reverts the file to its original contents, reporting any
// errors.
func (prog *prog) revertFile(nameOrig, nameNew string, counts *Counts) {
	prog.verboseMsg("Reverting " + nameNew + " to " + nameOrig + "...")

	err := os.Rename(nameOrig, nameNew)
	if err != nil {
		prog.twc.Wrap(
			fmt.Sprintf("Couldn't revert the file: %v", err),
			prog.indent)

		counts.revErrs++

		return
	}

	prog.verboseMsg(nameNew + " reverted to " + nameOrig)

	counts.reverted++
}

// reportFiles reports the number of files, their type and the action
// performed on them
func reportFiles(count int, desc, action string) {
	if count == 0 {
		return
	}

	fmt.Printf("%d %s %s %s\n", count, desc,
		english.Plural("file", count), action)
}

// verboseMsg Wraps the message if verbose messaging is on
func (prog prog) verboseMsg(msg string) {
	if verbose.IsOn() {
		prog.twc.Wrap(msg, prog.indent)
	}
}

// diffs runs a diff command against the two filenames and pipes the
// output to less
func (prog prog) diffs(nameOrig, nameNew string) error {
	// create the commands
	dcp := prog.diff.params
	dcp = append(dcp, nameOrig, nameNew)
	diffCmd := exec.Command(prog.diff.name, dcp...) //nolint:gosec
	diffCmdStr := fmt.Sprintf("the diff command (%q)",
		prog.diff.name+" "+strings.Join(dcp, " "))
	lessCmd := exec.Command(prog.less.name, prog.less.params...) //nolint:gosec
	lessCmdStr := fmt.Sprintf("the less command (%q)",
		prog.less.name+" "+strings.Join(prog.less.params, " "))

	// connect the output of diff to the input of less
	wStdout, err := diffCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("%s could not get the StdoutPipe: %w",
			diffCmdStr, err)
	}

	lessCmd.Stdin = wStdout
	lessCmd.Stdout = os.Stdout

	// start the commands
	err = diffCmd.Start()
	if err != nil {
		return fmt.Errorf("%s could not be started: %w", diffCmdStr, err)
	}

	err = lessCmd.Start()
	if err != nil {
		_ = diffCmd.Wait()
		return fmt.Errorf("%s could not be started: %w", lessCmdStr, err)
	}

	// wait for less to finish
	err = lessCmd.Wait()
	if err != nil {
		return fmt.Errorf("%s finished with an error: %w", lessCmdStr, err)
	}

	// wait for diff to finish
	err = diffCmd.Wait()
	// the diff command returns an exit status of 1 if the files differ. This
	// does not indicate an error
	if err != nil &&
		diffCmd.ProcessState.ExitCode() != 1 {
		return fmt.Errorf("%s finished with an error: %w", diffCmdStr, err)
	}

	return nil
}

// makeFileLists takes the files in entries and splits them into three sets:
// those files where there is a corresponding file without the extension,
// those where the corresponding file is identical and those for which there
// is some error.
func (prog prog) makeFileLists(entries map[string]os.FileInfo) (
	filenames, duplicates []string, badFiles []badFile,
) {
	filenames = make([]string, 0, len(entries))
	duplicates = make([]string, 0, len(entries))
	badFiles = make([]badFile, 0, len(entries))

	for nameOrig := range entries {
		nameNew := strings.TrimSuffix(nameOrig, prog.fileExtension)

		info, err := os.Stat(nameNew)
		if errors.Is(err, os.ErrNotExist) {
			badFiles = append(badFiles,
				badFile{
					name:    nameOrig,
					problem: fmt.Sprintf("there is no file named %q", nameNew),
				})

			continue
		}

		if err != nil {
			badFiles = append(badFiles,
				badFile{
					name:    nameOrig,
					problem: err.Error(),
				})

			continue
		}

		if info.IsDir() {
			badFiles = append(badFiles,
				badFile{
					name:    nameOrig,
					problem: fmt.Sprintf("%q is a directory", nameNew),
				})

			continue
		}

		newContent, err := os.ReadFile(nameNew) //nolint:gosec
		if err != nil {
			badFiles = append(badFiles,
				badFile{
					name:    nameOrig,
					problem: fmt.Sprintf("cannot read %q: %s", nameNew, err),
				})

			continue
		}

		origContent, err := os.ReadFile(nameOrig) //nolint:gosec
		if err != nil {
			badFiles = append(badFiles,
				badFile{
					name:    nameOrig,
					problem: fmt.Sprintf("cannot read %q: %s", nameOrig, err),
				})

			continue
		}

		if !fileContentsDiffer(newContent, origContent) {
			duplicates = append(duplicates, nameOrig)
			continue
		}

		filenames = append(filenames, nameOrig)
	}

	sort.Strings(filenames)
	sort.Strings(duplicates)
	sort.Slice(badFiles,
		func(i, j int) bool {
			return badFiles[i].name < badFiles[j].name
		})

	return filenames, duplicates, badFiles
}

// getFiles finds all the regular files in the directory with the given
// extension
func (prog prog) getFiles() (
	filenames, duplicates []string, badFiles []badFile, errs []error,
) {
	findFunc := dirsearch.Find

	if prog.searchSubDirs {
		findFunc = dirsearch.FindRecurse
	}

	entries, errs := findFunc(prog.searchDir,
		check.FileInfoName(check.StringHasSuffix[string](prog.fileExtension)),
		check.FileInfoIsRegular)

	if len(errs) != 0 {
		return nil, nil, nil, errs
	}

	filenames, duplicates, badFiles = prog.makeFileLists(entries)

	return filenames, duplicates, badFiles, errs
}
