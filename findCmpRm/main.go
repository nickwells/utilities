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
	dfltLessCmd   = "less"

	filenameIndent = 8
)

// Status holds counts of various operations on and problems with the files
type Status struct {
	fileCount                                           int
	dupFileCount                                        int
	badFileCount                                        int
	diffErr                                             int
	compared, skipped, deleted, reverted, kept, ignored int
	deleteFail, revertFail                              int
}

type DupAction string

const (
	DADelete = DupAction("delete")
	DAQuery  = DupAction("query")
	DAKeep   = DupAction("keep")
)

type CmpAction string

const (
	CAShowDiff = CmpAction("show-diffs")
	CAQuery    = CmpAction("query")
	CAKeep     = CmpAction("keep-all")
	CADelete   = CmpAction("delete-all")
	CARevert   = CmpAction("revert-all")
)

// Prog holds program parameters and status
type Prog struct {
	// parameters
	searchDir     string
	fileExtension string

	diffCmdName   string
	diffCmdParams []string

	lessCmdName   string
	lessCmdParams []string

	searchSubDirs bool
	tidyFiles     bool

	dupAction DupAction
	cmpAction CmpAction

	// display
	twc    *twrap.TWConf
	indent int

	// record dynamic behaviour choices
	shouldQuit bool
	revertAll  bool
	deleteAll  bool

	// record the behaviour and outcomes
	status Status
}

// NewProg returns a new Prog instance with the default values set
func NewProg() *Prog {
	return &Prog{
		searchDir:     dfltDir,
		fileExtension: dfltExtension,

		diffCmdName: dfltDiffCmd,
		lessCmdName: dfltLessCmd,

		searchSubDirs: true,

		dupAction: DAQuery,

		twc: twrap.NewTWConfOrPanic(),
	}
}

// badFile holds details of errors detected when processing files
type badFile struct {
	name string
	err  error
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
func (prog Prog) Report() {
	allFileCount := prog.status.fileCount +
		prog.status.dupFileCount +
		prog.status.badFileCount
	reportVal(allFileCount,
		english.Plural("file", allFileCount)+" found", 0)

	if allFileCount == 0 {
		return
	}
	reportVal(prog.status.badFileCount,
		"   problem "+english.Plural("file", prog.status.badFileCount), 4)
	reportVal(prog.status.dupFileCount,
		" duplicate "+english.Plural("file", prog.status.dupFileCount), 4)
	reportVal(prog.status.fileCount,
		"comparable "+english.Plural("file", prog.status.fileCount), 4)

	reportVal(prog.status.skipped, "skipped", 0)
	reportVal(prog.status.ignored, "ignored due to error", 0)

	reportVal(prog.status.compared, "compared", 0)
	reportVal(prog.status.deleted, "deleted", 0)
	reportVal(prog.status.reverted, "reverted", 0)
	reportVal(prog.status.kept, "kept", 0)
	if prog.revertAll {
		fmt.Printf("Some files were reverted without comparison\n")
	}
	if prog.shouldQuit {
		fmt.Printf("Quit before end\n")
	}
}

func main() {
	prog := NewProg()
	ps := paramset.NewOrDie(
		verbose.AddParams,
		versionparams.AddParams,

		addParams(prog),

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

	filenames, duplicates, badFiles := prog.getFiles()

	prog.showBadFiles(badFiles)
	prog.showDuplicates(duplicates)
	prog.cmpRmFiles(filenames)

	fmt.Println()
	prog.Report()
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
	prog.status.badFileCount = len(badFiles)

	filenames := make([]string, 0, len(badFiles))
	for _, fe := range badFiles {
		filenames = append(filenames, fe.name)
	}
	shortNames, maxNameLen := prog.shortNames(filenames)
	fmt.Printf("%d problem %s found\n",
		len(badFiles),
		english.Plural("file", len(badFiles)))
	fmt.Println("in", prog.searchDir)
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
func (prog *Prog) showDuplicates(filenames []string) {
	if len(filenames) == 0 {
		return
	}
	prog.status.dupFileCount = len(filenames)

	shortNames, _ := prog.shortNames(filenames)
	fmt.Printf("%d duplicate %s found\n",
		len(filenames),
		english.Plural("file", len(filenames)))
	fmt.Println("in", prog.searchDir)
	prog.twc.NoRptPathList(shortNames, filenameIndent)

	if prog.tidyFiles || prog.queryDeleteDuplicates() {
		for _, nameOrig := range filenames {
			prog.deleteFile(nameOrig)
		}
		fmt.Println("Duplicates deleted")
	}
}

// cmpRmFiles loops over the files prompting the user to compare each one
// with the new instance and then asking if the file should be deleted or the
// new file reverted.
func (prog *Prog) cmpRmFiles(filenames []string) {
	if len(filenames) == 0 {
		return
	}
	prog.status.fileCount = len(filenames)

	shortNames, maxNameLen := prog.shortNames(filenames)

	digits := mathutil.Digits(int64(len(filenames)) + 1)
	nameFormat := fmt.Sprintf("    (%%%dd / %%%dd) %%%d.%ds: ",
		digits, digits, maxNameLen, maxNameLen)
	prog.indent = len(fmt.Sprintf(nameFormat, 0, 0, ""))

	fmt.Printf("%d %s found\n",
		len(filenames),
		english.Plural("file", len(filenames)))
	fmt.Println("in", prog.searchDir)
	prog.twc.IdxNoRptPathList(shortNames, filenameIndent)

	for i, nameOrig := range filenames {
		nameNew := strings.TrimSuffix(nameOrig, prog.fileExtension)
		if prog.revertAll {
			prog.revertFile(nameOrig, nameNew)
			continue
		}
		if prog.deleteAll {
			prog.deleteFile(nameOrig)
			continue
		}

		fmt.Printf(nameFormat, i+1, len(filenames), shortNames[i])

		prog.queryShowDiff(nameOrig, nameNew)
		if prog.revertAll {
			prog.revertFile(nameOrig, nameNew)
		}
		if prog.deleteAll {
			prog.deleteFile(nameOrig)
		}
		if prog.shouldQuit {
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

// queryDeleteDuplicates returns true if the user responds that the
// duplicates should be deleted
func (prog *Prog) queryDeleteDuplicates() bool {
	deleteDuplicatesResp := responder.NewOrPanic(
		"delete all duplicate files",
		map[rune]string{
			'y': "to delete all duplicates files with extension " +
				prog.fileExtension,
			'n': "to keep these duplicates",
		},
		responder.SetDefault('y'),
		responder.SetIndents(0, prog.indent))

	response := deleteDuplicatesResp.GetResponseOrDie()
	fmt.Println()
	return response == 'y'
}

// queryShowDiff asks if the differences between the new file and the
// original should be shown and then acts accordingly, reporting any errors
// found.
func (prog *Prog) queryShowDiff(nameOrig, nameNew string) {
	showDiffResp := responder.NewOrPanic(
		"Show differences",
		map[rune]string{
			'y': "to show differences",
			'n': "to skip this file",
			'd': "delete this and all subsequent files" +
				" with extension: " + prog.fileExtension,
			'r': "revert this and all subsequent base" +
				" files to the contents of" +
				" the files with extension: " + prog.fileExtension,
			'q': "to quit",
		},
		responder.SetDefault('y'),
		responder.SetIndents(0, prog.indent))

	response := showDiffResp.GetResponseOrDie()
	fmt.Println()

	switch response {
	case 'y':
		err := prog.showDiffs(nameOrig, nameNew)
		if err != nil {
			prog.twc.Wrap(fmt.Sprintf("Ignoring due to: %v", err), prog.indent)
			prog.status.ignored++
			prog.status.diffErr++
			return
		}
		prog.status.compared++

		prog.queryDeleteFile(nameOrig, nameNew)
	case 'n':
		prog.skip()
	case 'r':
		prog.setRevertAll()
	case 'd':
		prog.setDeleteAll()
	case 'q':
		prog.setShouldQuit()
	}
}

// skip reports the skipping of the file
func (prog *Prog) skip() {
	prog.verboseMsg("Skipping...")
	prog.status.skipped++
}

// setRevertAll sets the revertAll flag
func (prog *Prog) setRevertAll() {
	prog.verboseMsg("Reverting all...")
	prog.revertAll = true
}

// setDeleteAll sets the deleteAll flag
func (prog *Prog) setDeleteAll() {
	prog.verboseMsg("Deleting all...")
	prog.deleteAll = true
}

// setShouldQuit sets the shouldQuit flag
func (prog *Prog) setShouldQuit() {
	prog.verboseMsg("Quitting...")
	prog.shouldQuit = true
}

// queryDeleteFile asks if the file should be deleted and then acts
// accordingly, reporting any errors found.
func (prog *Prog) queryDeleteFile(nameOrig, nameNew string) {
	deleteFileResp := responder.NewOrPanic(
		"delete file",
		map[rune]string{
			'y': "to delete this file",
			'n': "to keep this file",
			'r': "to revert the base file to this content",
		},
		responder.SetDefault('n'),
		responder.SetIndents(prog.indent, prog.indent))

	response := deleteFileResp.GetResponseOrDie()
	fmt.Println()

	switch response {
	case 'y':
		prog.deleteFile(nameOrig)
	case 'r':
		prog.revertFile(nameOrig, nameNew)
	default:
		prog.status.kept++
	}
}

// deleteFile deletes the named file, reporting any errors
func (prog *Prog) deleteFile(name string) {
	prog.verboseMsg("Deleting file...")

	err := os.Remove(name)
	if err != nil {
		prog.twc.Wrap(
			fmt.Sprintf("Couldn't delete the file: %v", err),
			prog.indent)
		prog.status.deleteFail++
		return
	}

	prog.verboseMsg("File deleted")
	prog.status.deleted++
}

// revertFile reverts the file to its original contents, reporting any
// errors.
func (prog *Prog) revertFile(nameOrig, nameNew string) {
	prog.verboseMsg(
		fmt.Sprintf("Reverting to the file with extension %q",
			prog.fileExtension))

	err := os.Rename(nameOrig, nameNew)
	if err != nil {
		prog.twc.Wrap(
			fmt.Sprintf("Couldn't revert the file: %v", err),
			prog.indent)
		prog.status.revertFail++
		return
	}

	prog.verboseMsg("File reverted")
	prog.status.reverted++
}

// verboseMsg Wraps the message if verbose messaging is on
func (prog Prog) verboseMsg(msg string) {
	if verbose.IsOn() {
		prog.twc.Wrap(msg, prog.indent)
	}
}

// showDiffs runs a diff command against the two filenames and pipes the
// output to less
func (prog Prog) showDiffs(nameOrig, nameNew string) error {
	r, w := io.Pipe()

	dcp := prog.diffCmdParams
	dcp = append(dcp, nameOrig, nameNew)
	diffCmd := exec.Command(prog.diffCmdName, dcp...)
	diffCmd.Stdout = w

	lessCmd := exec.Command(prog.lessCmdName, prog.lessCmdParams...)
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
					name: nameOrig,
					err: fmt.Errorf("there is no file called %q: %w",
						nameNew, err),
				})
			continue
		}

		if info.IsDir() {
			badFiles = append(badFiles,
				badFile{
					name: nameOrig,
					err: fmt.Errorf("the corresponding file is a directory: %q",
						nameNew),
				})
			continue
		}

		newContent, err := os.ReadFile(nameNew)
		if err != nil {
			badFiles = append(badFiles,
				badFile{
					name: nameOrig,
					err: fmt.Errorf("cannot read the contents of %q: %w",
						nameNew, err),
				})
			continue
		}

		origContent, err := os.ReadFile(nameOrig)
		if err != nil {
			badFiles = append(badFiles,
				badFile{
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
func (prog Prog) getFiles() (
	filenames, duplicates []string, badFiles []badFile,
) {
	findFunc := dirsearch.Find
	if prog.searchSubDirs {
		findFunc = dirsearch.FindRecurse
	}
	entries, errs := findFunc(prog.searchDir,
		check.FileInfoName(check.StringHasSuffix[string](prog.fileExtension)),
		check.FileInfoIsRegular)

	if len(errs) != 0 {
		fmt.Fprintln(os.Stderr, "Couldn't find the entries:")
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, "\t", err)
		}
		os.Exit(1)
	}

	return prog.makeFileLists(entries)
}
