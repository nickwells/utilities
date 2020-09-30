package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/cli.mod/cli/responder"
	"github.com/nickwells/dirsearch.mod/dirsearch"
	"github.com/nickwells/english.mod/english"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/twrap.mod/twrap"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Wed Oct 23 18:05:24 2019

const dfltExtension = ".orig"
const dfltDir = "."

var searchSubDirs bool
var dir string = dfltDir
var fileExtension string = dfltExtension
var tidyFiles bool

const dfltDiffCmd = "diff"

var diffCmdName = dfltDiffCmd
var diffCmdParams = []string{}

var lessCmdName = "less"
var lessCmdParams = []string{}

// Status holds counts of various operations on and problems with the files
type Status struct {
	rawFileCount                                                int
	fileErr, diffErr                                            int
	compared, skipped, deleted, reverted, kept, tidied, ignored int
	shouldQuit, revertAll, deleteAll                            bool

	twc        *twrap.TWConf
	fileChecks filecheck.Provisos
	indent     int
}

// reportVal checks that n is greater than zero, reports the value and
// returns true, false otherwise
func reportVal(n int, name string, indent int) bool {
	if n <= 0 {
		return false
	}
	fmt.Printf("%s%3d %s\n", strings.Repeat(" ", indent), n, name)
	return true
}

// Report will print out the Status structure
func (s Status) Report() {
	fmt.Printf("%3d %s found\n",
		s.rawFileCount,
		english.Plural("file", s.rawFileCount))

	reportVal(s.skipped, "skipped", 0)
	reportVal(s.tidied, "tidied", 0)
	if reportVal(s.ignored, "ignored due to error", 0) {
		fmt.Println("\tof which:")
		reportVal(s.fileErr, "due to file error", 8)
		reportVal(s.diffErr, "due to diff error", 8)
	}

	if reportVal(s.compared, "compared", 0) {
		fmt.Println("\tof which:")
		reportVal(s.deleted, "deleted", 8)
		reportVal(s.reverted, "reverted", 8)
		reportVal(s.kept, "kept", 8)
	}
	if s.revertAll {
		fmt.Printf("Some files were reverted without comparison\n")
	}
	if s.shouldQuit {
		fmt.Printf("Quit before end\n")
	}
}

func main() {
	ps := paramset.NewOrDie(addParams,
		addExamples,
		addRefs,
		verbose.AddParams,
		SetGlobalConfigFile,
		SetConfigFile,
		param.SetProgramDescription(
			"This finds any files in the given directory"+
				" (by default: "+dfltDir+") with the given extension"+
				" (by default: "+dfltExtension+"). It presents each"+
				" file and gives the user the chance to compare it"+
				" with the corresponding file without the"+
				" extension. The user is then asked whether to"+
				" remove the file with the extension. The command name"+
				" echoes this: find, compare, remove. You will also have"+
				" the opportunity to revert the file back to the original"+
				" contents."),
	)

	ps.Parse()

	filenames := getFiles()

	s := &Status{
		twc:          twrap.NewTWConfOrPanic(),
		fileChecks:   filecheck.FileExists(),
		rawFileCount: len(filenames),
	}
	if tidyFiles {
		filenames = s.tidyRedundantFiles(filenames)
	}

	s.cmpRmFiles(filenames)

	fmt.Println()
	s.Report()
}

// cmpRmFiles loops over the files prompting the user to compare each one
// with the new instance and then asking if the file should be deleted or the
// new file reverted.
func (s *Status) cmpRmFiles(filenames []string) {
	if len(filenames) == 0 {
		return
	}

	maxNameLen := getMaxNameLen(filenames)
	s.indent = maxNameLen + 2

	fmt.Println(len(filenames),
		english.Plural("file", len(filenames)),
		"found:")
	s.twc.IdxNoRptPathList(filenames, 4)

fileLoop:
	for _, nameOrig := range filenames {
		nameNew := strings.TrimSuffix(nameOrig, fileExtension)
		if s.revertAll {
			s.revertFile(nameOrig, nameNew)
			continue
		}
		if s.deleteAll {
			s.deleteFile(nameOrig)
			continue
		}

		fmt.Printf("%*.*s: ", maxNameLen, maxNameLen, nameOrig)

		if err := s.fileOK(nameNew); err != nil {
			fmt.Println("Ignoring due to:", err)
			s.ignored++
			continue
		}

		s.queryShowDiff(nameOrig, nameNew)
		if s.revertAll {
			s.revertFile(nameOrig, nameNew)
		}
		if s.deleteAll {
			s.deleteFile(nameOrig)
		}
		if s.shouldQuit {
			break fileLoop
		}
	}
}

// tidyRedundantFiles checks the original file list for missing new files,
// new files that are directories or new files identical to the original and
// removes the problem original file
func (s *Status) tidyRedundantFiles(filenames []string) []string {
	curatedFiles := []string{}
	for _, nameOrig := range filenames {
		if isRedundant(nameOrig) {
			s.tidy(nameOrig)
			continue
		}
		curatedFiles = append(curatedFiles, nameOrig)
	}

	return curatedFiles
}

// isRedundant returns the result of the redundancy checks
func isRedundant(nameOrig string) bool {
	nameNew := strings.TrimSuffix(nameOrig, fileExtension)
	info, err := os.Stat(nameNew)
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	if info.IsDir() {
		return true
	}
	newContent, err := ioutil.ReadFile(nameNew)
	if err != nil {
		return false
	}
	origContent, err := ioutil.ReadFile(nameOrig)
	if err != nil {
		return false
	}

	newMD5 := md5.Sum(newContent)
	origMD5 := md5.Sum(origContent)

	return newMD5 == origMD5
}

// tidy reports the tidying of the file
func (s *Status) tidy(name string) {
	s.verboseMsg("Tidying " + name + "...")

	err := os.Remove(name)
	if err != nil {
		s.twc.Wrap("Couldn't delete file: "+err.Error(), 0)
		return
	}

	s.tidied++
}

// fileOK checks that the file passes the status checks and returns true if
// it does and false otherwise
func (s *Status) fileOK(file string) error {
	err := s.fileChecks.StatusCheck(file)
	if err != nil {
		s.fileErr++
		return err
	}
	return nil
}

// queryShowDiff asks if the differences between the new file and the
// original should be shown and then acts accordingly, reporting any errors
// found.
func (s *Status) queryShowDiff(nameOrig, nameNew string) {
	showDiffResp := responder.NewOrPanic(
		"Show differences",
		map[rune]string{
			'y': "to show differences",
			'n': "to skip this file",
			'd': "delete this and all subsequent files" +
				" with extension: " + fileExtension,
			'r': "revert this and all subsequent base files" +
				" to the contents of the files with extension: " + fileExtension,
			'q': "to quit",
		},
		responder.SetDefault('y'),
		responder.SetIndents(0, s.indent))

	response := showDiffResp.GetResponseOrDie()
	fmt.Println()

	switch response {
	case 'y':
		err := showDiffs(nameOrig, nameNew)
		if err != nil {
			s.twc.Wrap(fmt.Sprintf("Ignoring due to: %v", err), s.indent)
			s.ignored++
			s.diffErr++
			return
		}
		s.compared++

		s.queryDeleteFile(nameOrig, nameNew)
	case 'n':
		s.skip()
	case 'r':
		s.setRevertAll()
	case 'd':
		s.setDeleteAll()
	case 'q':
		s.setShouldQuit()
	}
}

// skip reports the skipping of the file
func (s *Status) skip() {
	s.verboseMsg("Skipping...")
	s.skipped++
}

// setRevertAll sets the revertAll flag
func (s *Status) setRevertAll() {
	s.verboseMsg("Reverting all...")
	s.revertAll = true
}

// setDeleteAll sets the deleteAll flag
func (s *Status) setDeleteAll() {
	s.verboseMsg("Deleting all...")
	s.deleteAll = true
}

// setShouldQuit sets the shouldQuit flag
func (s *Status) setShouldQuit() {
	s.verboseMsg("Quitting...")
	s.shouldQuit = true
}

// queryDeleteFile asks if the file should be deleted and then acts
// accordingly, reporting any errors found.
func (s *Status) queryDeleteFile(nameOrig, nameNew string) {
	deleteFileResp := responder.NewOrPanic(
		"delete file",
		map[rune]string{
			'y': "to delete this file",
			'n': "to keep this file",
			'r': "to revert the file to this content",
		},
		responder.SetDefault('n'),
		responder.SetIndents(s.indent, s.indent))

	response := deleteFileResp.GetResponseOrDie()
	fmt.Println()

	switch response {
	case 'y':
		s.deleteFile(nameOrig)
	case 'r':
		s.revertFile(nameOrig, nameNew)
	default:
		s.kept++
	}
}

// deleteFile deletes the named file, reporting any errors
func (s *Status) deleteFile(name string) {
	s.verboseMsg("Deleting file...")

	err := os.Remove(name)
	if err != nil {
		s.twc.Wrap(
			fmt.Sprintf("Couldn't delete the file: %v", err),
			s.indent)
		return
	}

	s.verboseMsg("File deleted")
	s.deleted++
}

// revertFile reverts the file to its original contents, reporting any
// errors.
func (s *Status) revertFile(nameOrig, nameNew string) {
	s.verboseMsg("Reverting to the file with extension '" + fileExtension + "' ...")

	err := os.Rename(nameOrig, nameNew)
	if err != nil {
		s.twc.Wrap(
			fmt.Sprintf("Couldn't revert the file: %v", err),
			s.indent)
		return
	}

	s.verboseMsg("File reverted")
	s.reverted++
}

// verboseMsg Wraps the message if verbose messaging is on
func (s *Status) verboseMsg(msg string) {
	if verbose.IsOn() {
		s.twc.Wrap(msg, s.indent)
	}
}

// showDiffs runs a diff command against the two filenames and pipes the
// output to less
func showDiffs(nameOrig, nameNew string) error {
	r, w := io.Pipe()

	dcp := diffCmdParams
	dcp = append(dcp, nameOrig, nameNew)
	diffCmd := exec.Command(diffCmdName, dcp...)
	diffCmd.Stdout = w

	lessCmd := exec.Command(lessCmdName, lessCmdParams...)
	lessCmd.Stdin = r
	lessCmd.Stdout = os.Stdout

	err := diffCmd.Start()
	if err != nil {
		return fmt.Errorf("Couldn't start the diff command: %w", err)
	}
	err = lessCmd.Start()
	if err != nil {
		return fmt.Errorf("Couldn't start the less command: %w", err)
	}
	err = diffCmd.Wait()
	// the diff command returns an exit status of 1 if the files differ. This
	// does not indicate an error
	if err != nil &&
		diffCmd.ProcessState.ExitCode() != 1 {
		return fmt.Errorf("The diff command finished with an error: %w", err)
	}
	w.Close()
	err = lessCmd.Wait()
	if err != nil {
		return fmt.Errorf("The less command finished with an error: %w", err)
	}
	return nil
}

// getMaxNameLen returns the length of the longest file name
func getMaxNameLen(filenames []string) int {
	maxNameLen := 0

	for _, name := range filenames {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}
	return maxNameLen
}

// getFiles finds all the regular files in the directory with the given
// extension
func getFiles() []string {
	findFunc := dirsearch.Find
	if searchSubDirs {
		findFunc = dirsearch.FindRecurse
	}
	entries, errs := findFunc(dir,
		check.FileInfoName(check.StringHasSuffix(fileExtension)),
		check.FileInfoIsRegular)

	if len(errs) != 0 {
		fmt.Fprintln(os.Stderr, "Couldn't find the entries:")
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, "\t", err)
		}
		os.Exit(1)
	}

	filenames := make([]string, 0, len(entries))

	for name := range entries {
		filenames = append(filenames, name)
	}

	sort.Strings(filenames)
	return filenames
}
