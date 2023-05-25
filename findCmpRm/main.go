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
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/mathutil.mod/v2/mathutil"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/twrap.mod/twrap"
	"github.com/nickwells/verbose.mod/verbose"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// Created: Wed Oct 23 18:05:24 2019

const (
	dfltExtension = ".orig"
	dfltDir       = "."
	dfltDiffCmd   = "diff"

	filenameIndent = 8
)

var (
	searchSubDirs = true
	tidyFiles     bool
)

var (
	searchDir     = dfltDir
	fileExtension = dfltExtension

	diffCmdName = dfltDiffCmd
	lessCmdName = "less"

	diffCmdParams = []string{}
	lessCmdParams = []string{}
)

// fileErr holds details of errors detected when processing files
type fileErr struct {
	name string
	err  error
}

// Status holds counts of various operations on and problems with the files
type Status struct {
	fileCount                                           int
	dupFileCount                                        int
	badFileCount                                        int
	fileErr, diffErr                                    int
	compared, skipped, deleted, reverted, kept, ignored int
	deleteFail, revertFail                              int
	shouldQuit, revertAll, deleteAll                    bool

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
	allFileCount := s.fileCount + s.dupFileCount + s.badFileCount
	reportVal(allFileCount,
		english.Plural("file", allFileCount)+" found", 0)

	if allFileCount == 0 {
		return
	}
	reportVal(s.badFileCount,
		"   problem "+english.Plural("file", s.badFileCount), 4)
	reportVal(s.dupFileCount,
		" duplicate "+english.Plural("file", s.dupFileCount), 4)
	reportVal(s.fileCount,
		"comparable "+english.Plural("file", s.fileCount), 4)

	reportVal(s.skipped, "skipped", 0)
	if reportVal(s.ignored, "ignored due to error", 0) {
		fmt.Println("\tof which:")
		reportVal(s.fileErr, "due to file error", 8)
		reportVal(s.diffErr, "due to diff error", 8)
	}

	reportVal(s.compared, "compared", 0)
	reportVal(s.deleted, "deleted", 0)
	reportVal(s.reverted, "reverted", 0)
	reportVal(s.kept, "kept", 0)
	if s.revertAll {
		fmt.Printf("Some files were reverted without comparison\n")
	}
	if s.shouldQuit {
		fmt.Printf("Quit before end\n")
	}
}

func main() {
	ps := paramset.NewOrDie(
		verbose.AddParams,
		versionparams.AddParams,

		addParams,

		addExamples,
		addRefs,
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

	filenames, duplicates, badFiles := getFiles()

	s := &Status{
		twc:          twrap.NewTWConfOrPanic(),
		fileChecks:   filecheck.FileExists(),
		fileCount:    len(filenames),
		dupFileCount: len(duplicates),
		badFileCount: len(badFiles),
	}

	s.showBadFiles(badFiles)
	s.showDuplicates(duplicates)

	s.cmpRmFiles(filenames)

	fmt.Println()
	s.Report()
}

// shortNames returns a list of filenames with the search directory name
// removed. It also returns the maximum length of the names
func shortNames(filenames []string) ([]string, int) {
	shortNames := make([]string, 0, len(filenames))
	maxLen := 0
	for _, fn := range filenames {
		shortName := strings.TrimPrefix(fn, searchDir+string(os.PathSeparator))
		if len(shortName) > maxLen {
			maxLen = len(shortName)
		}
		shortNames = append(shortNames, shortName)
	}

	return shortNames, maxLen
}

// showBadFiles displays the list of files for which problems were detected
func (s *Status) showBadFiles(badFiles []fileErr) {
	if len(badFiles) == 0 {
		return
	}

	filenames := make([]string, 0, len(badFiles))
	for _, fe := range badFiles {
		filenames = append(filenames, fe.name)
	}
	shortNames, maxNameLen := shortNames(filenames)
	fmt.Printf("%d problem %s found\n",
		len(badFiles),
		english.Plural("file", len(badFiles)))
	fmt.Println("in", searchDir)
	for i, name := range shortNames {
		fmt.Printf("%s%*s - %s\n",
			strings.Repeat(" ", filenameIndent),
			maxNameLen,
			name, badFiles[i].err)
	}
	fmt.Println()
}

// showDuplicates displays the list of duplicate files and prompts the user
// to delete them
func (s *Status) showDuplicates(filenames []string) {
	if len(filenames) == 0 {
		return
	}

	shortNames, _ := shortNames(filenames)
	fmt.Printf("%d duplicate %s found\n",
		len(filenames),
		english.Plural("file", len(filenames)))
	fmt.Println("in", searchDir)
	s.twc.NoRptPathList(shortNames, filenameIndent)

	if tidyFiles || s.queryDeleteDuplicates() {
		for _, nameOrig := range filenames {
			s.deleteFile(nameOrig)
		}
		fmt.Println("Duplicates deleted")
	}
}

// cmpRmFiles loops over the files prompting the user to compare each one
// with the new instance and then asking if the file should be deleted or the
// new file reverted.
func (s *Status) cmpRmFiles(filenames []string) {
	if len(filenames) == 0 {
		return
	}

	shortNames, maxNameLen := shortNames(filenames)

	digits := mathutil.Digits(int64(len(filenames)) + 1)
	nameFormat := fmt.Sprintf("    (%%%dd / %%%dd) %%%d.%ds: ",
		digits, digits, maxNameLen, maxNameLen)
	s.indent = len(fmt.Sprintf(nameFormat, 0, 0, ""))

	fmt.Printf("%d %s found\n",
		len(filenames),
		english.Plural("file", len(filenames)))
	fmt.Println("in", searchDir)
	s.twc.IdxNoRptPathList(shortNames, filenameIndent)

	for i, nameOrig := range filenames {
		nameNew := strings.TrimSuffix(nameOrig, fileExtension)
		if s.revertAll {
			s.revertFile(nameOrig, nameNew)
			continue
		}
		if s.deleteAll {
			s.deleteFile(nameOrig)
			continue
		}

		fmt.Printf(nameFormat, i+1, len(filenames), shortNames[i])

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
			break
		}
	}
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

// queryDeleteDuplicates returns true if the user responds that the
// duplicates should be deleted
func (s *Status) queryDeleteDuplicates() bool {
	deleteDuplicatesResp := responder.NewOrPanic(
		"delete all duplicate files",
		map[rune]string{
			'y': "to delete all duplicates files with extension " +
				fileExtension,
			'n': "to keep these duplicates",
		},
		responder.SetDefault('y'),
		responder.SetIndents(0, s.indent))

	response := deleteDuplicatesResp.GetResponseOrDie()
	fmt.Println()
	return response == 'y'
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
			'r': "revert this and all subsequent base" +
				" files to the contents of" +
				" the files with extension: " + fileExtension,
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
		s.deleteFail++
		return
	}

	s.verboseMsg("File deleted")
	s.deleted++
}

// revertFile reverts the file to its original contents, reporting any
// errors.
func (s *Status) revertFile(nameOrig, nameNew string) {
	s.verboseMsg(
		fmt.Sprintf("Reverting to the file with extension %q",
			fileExtension))

	err := os.Rename(nameOrig, nameNew)
	if err != nil {
		s.twc.Wrap(
			fmt.Sprintf("Couldn't revert the file: %v", err),
			s.indent)
		s.revertFail++
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

// makeFileLists takes the files in entries and splits them into three sets:
// those files where there is a corresponding file without the extension,
// those where the corresponding file is identical and those for which there
// is some error.
func makeFileLists(entries map[string]os.FileInfo) (
	filenames, duplicates []string, badFiles []fileErr,
) {
	filenames = make([]string, 0, len(entries))
	duplicates = make([]string, 0, len(entries))
	badFiles = make([]fileErr, 0, len(entries))

	for nameOrig := range entries {
		nameNew := strings.TrimSuffix(nameOrig, fileExtension)
		info, err := os.Stat(nameNew)
		if errors.Is(err, os.ErrNotExist) {
			badFiles = append(badFiles,
				fileErr{
					name: nameOrig,
					err: fmt.Errorf("there is no file called %q: %w",
						nameNew, err),
				})
			continue
		}

		if info.IsDir() {
			badFiles = append(badFiles,
				fileErr{
					name: nameOrig,
					err: fmt.Errorf("the corresponding file is a directory: %q",
						nameNew),
				})
			continue
		}

		newContent, err := os.ReadFile(nameNew)
		if err != nil {
			badFiles = append(badFiles,
				fileErr{
					name: nameOrig,
					err: fmt.Errorf("cannot read the contents of %q: %w",
						nameNew, err),
				})
			continue
		}

		origContent, err := os.ReadFile(nameOrig)
		if err != nil {
			badFiles = append(badFiles,
				fileErr{
					name: nameOrig,
					err: fmt.Errorf("cannot read the contents of %q: %w",
						nameOrig, err),
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
	return filenames, duplicates, badFiles
}

// getFiles finds all the regular files in the directory with the given
// extension
func getFiles() (
	filenames, duplicates []string, badFiles []fileErr,
) {
	findFunc := dirsearch.Find
	if searchSubDirs {
		findFunc = dirsearch.FindRecurse
	}
	entries, errs := findFunc(searchDir,
		check.FileInfoName(check.StringHasSuffix[string](fileExtension)),
		check.FileInfoIsRegular)

	if len(errs) != 0 {
		fmt.Fprintln(os.Stderr, "Couldn't find the entries:")
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, "\t", err)
		}
		os.Exit(1)
	}

	return makeFileLists(entries)
}
