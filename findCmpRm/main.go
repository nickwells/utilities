package main

import (
	"errors"
	"fmt"
	"io"
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

type DupAction string

const (
	DADelete = DupAction("delete")
	DAQuery  = DupAction("query")
	DAKeep   = DupAction("keep")
)

type CmpAction string

const (
	CAShowDiff  = CmpAction("show-diffs")
	CAQuery     = CmpAction("query")
	CAKeepAll   = CmpAction("keep-all")
	CADeleteAll = CmpAction("delete-all")
	CARevertAll = CmpAction("revert-all")
)

// CmdInfo records the command name and the parameters to be supplied
type CmdInfo struct {
	name   string
	params []string
}

// Prog holds program parameters and status
type Prog struct {
	exitStatus int

	stack *verbose.Stack

	// parameters
	searchDir     string
	searchSubDirs bool
	fileExtension string

	diff CmdInfo
	less CmdInfo

	showDiffR  responder.Responder
	deleteDupR responder.Responder
	postDiffR  responder.Responder

	dupAction DupAction
	cmpAction CmpAction

	// display
	twc    *twrap.TWConf
	indent int

	// record the behaviour and outcomes
	status Status
}

// NewProg returns a new Prog instance with the default values set
func NewProg() *Prog {
	return &Prog{
		stack: &verbose.Stack{},

		searchDir:     dfltDir,
		searchSubDirs: true,
		fileExtension: dfltExtension,

		diff: CmdInfo{name: dfltDiffCmd},
		less: CmdInfo{name: dfltLessCmd},

		dupAction: DAQuery,
		cmpAction: CAQuery,

		twc: twrap.NewTWConfOrPanic(),

		status: InitStatus(),
	}
}

// setResponders sets the responders on the Prog
func (prog *Prog) setResponders() {
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
	prog := NewProg()
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
func (prog Prog) shortNames(filenames []string) ([]string, int) {
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
func (prog *Prog) showBadFiles(badFiles []badFile) {
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
func (prog *Prog) showDuplicateFiles(dupFiles []string) {
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
func (prog *Prog) processDuplicateFiles(dupFiles []string) {
	if len(dupFiles) == 0 {
		return
	}

	defer prog.stack.Start("processDuplicateFiles", "Start")()

	switch prog.dupAction {
	case DADelete:
		prog.deleteAllFiles(dupFiles, &prog.status.dupFile)
	case DAQuery:
		if prog.queryDeleteDuplicates() == 'y' {
			prog.deleteAllFiles(dupFiles, &prog.status.dupFile)
		}
	}
	fmt.Println()
}

// showComparableFiles loops over the files prompting the user to compare
// each one with the new instance and then asking if the file should be
// deleted or the new file reverted.
func (prog *Prog) showComparableFiles(cmpFiles []string) {
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
func (prog *Prog) processComparableFiles(cmpFiles []string) {
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
		case CAQuery:
			fmt.Printf(nameFormat, i+1, len(cmpFiles), shortNames[i])
			if prog.queryShowDiff() {
				prog.showDiff(nameOrig, nameNew, &prog.status.cmpFile)
			}
		case CAShowDiff:
			fmt.Printf(nameFormat, i+1, len(cmpFiles), shortNames[i])
			prog.showDiff(nameOrig, nameNew, &prog.status.cmpFile)
		}

		// queryShowDiff can change the value of prog.cmpAction so switch again
		switch prog.cmpAction {
		case CARevertAll:
			prog.revertFile(nameOrig, nameNew, &prog.status.cmpFile)
		case CADeleteAll:
			prog.deleteFile(nameOrig, &prog.status.cmpFile)
		case CAKeepAll:
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
func (prog *Prog) queryDeleteDuplicates() rune {
	response := prog.deleteDupR.GetResponseIndentOrDie(0, prog.indent)
	fmt.Println()
	return response
}

// showDiff shows the differences between the new file and the original and
// then queries for further actions.
func (prog *Prog) showDiff(nameOrig, nameNew string, counts *Counts) {
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
func (prog *Prog) queryShowDiff() bool {
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
func (prog *Prog) setRevertAll() {
	prog.verboseMsg("Reverting all...")
	prog.cmpAction = CARevertAll
}

// setDeleteAll sets the comparison action to Delete-All
func (prog *Prog) setDeleteAll() {
	prog.verboseMsg("Deleting all...")
	prog.cmpAction = CADeleteAll
}

// setKeepAll sets the comparison action to Keep-All
func (prog *Prog) setKeepAll() {
	prog.verboseMsg("All remaining files will be kept...")
	prog.cmpAction = CAKeepAll
}

// queryDeleteFile asks if the file should be deleted and then acts
// accordingly, reporting any errors found.
func (prog *Prog) queryDeleteFile(nameOrig, nameNew string) {
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
func (prog *Prog) deleteAllFiles(filenames []string, count *Counts) {
	for _, fName := range filenames {
		prog.deleteFile(fName, count)
	}
	reportFiles(count.deleted, count.name, "deleted")
	reportFiles(count.delErrs, count.name, "could not be deleted")
}

// deleteFile deletes the named file, reporting any errors
func (prog *Prog) deleteFile(name string, counts *Counts) {
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
func (prog *Prog) revertFile(nameOrig, nameNew string, counts *Counts) {
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
func (prog Prog) verboseMsg(msg string) {
	if verbose.IsOn() {
		prog.twc.Wrap(msg, prog.indent)
	}
}

// diffs runs a diff command against the two filenames and pipes the
// output to less
func (prog Prog) diffs(nameOrig, nameNew string) error {
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()

	// create the commands
	dcp := prog.diff.params
	dcp = append(dcp, nameOrig, nameNew)
	diffCmd := exec.Command(prog.diff.name, dcp...)
	lessCmd := exec.Command(prog.less.name, prog.less.params...)

	// connect the output of diff to the input of less
	diffCmd.Stdout = w
	lessCmd.Stdin = r
	lessCmd.Stdout = os.Stdout

	// start the commands
	err := diffCmd.Start()
	if err != nil {
		return fmt.Errorf("Couldn't start the diff command: %w", err)
	}
	err = lessCmd.Start()
	if err != nil {
		w.Close() // close diff's stdout
		_ = diffCmd.Wait()
		return fmt.Errorf("Couldn't start the less command: %w", err)
	}

	// wait for less to finish
	err = lessCmd.Wait()
	if err != nil {
		return fmt.Errorf("The less command finished with an error: %w", err)
	}
	w.Close() // close diff's stdout
	// wait for diff to finish
	err = diffCmd.Wait()
	// the diff command returns an exit status of 1 if the files differ. This
	// does not indicate an error
	if err != nil &&
		diffCmd.ProcessState.ExitCode() != 1 {
		return fmt.Errorf("The diff command finished with an error: %w", err)
	}
	return nil
}

// makeFileLists takes the files in entries and splits them into three sets:
// those files where there is a corresponding file without the extension,
// those where the corresponding file is identical and those for which there
// is some error.
func (prog Prog) makeFileLists(entries map[string]os.FileInfo) (
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

		newContent, err := os.ReadFile(nameNew)
		if err != nil {
			badFiles = append(badFiles,
				badFile{
					name:    nameOrig,
					problem: fmt.Sprintf("cannot read %q: %s", nameNew, err),
				})
			continue
		}

		origContent, err := os.ReadFile(nameOrig)
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
func (prog Prog) getFiles() (
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
