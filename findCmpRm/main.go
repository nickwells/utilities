// findCmpRm
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/cli.mod/cli/responder"
	"github.com/nickwells/dirsearch.mod/dirsearch"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v4/param"
	"github.com/nickwells/param.mod/v4/param/paramset"
	"github.com/nickwells/twrap.mod/twrap"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Wed Oct 23 18:05:24 2019

const dfltExtension = ".orig"
const dfltDir = "."

var searchSubDirs bool
var dir string = dfltDir
var extension string = dfltExtension

var diffCmdName = "diff"
var diffCmdParams = []string{}

var lessCmdName = "less"
var lessCmdParams = []string{}

// status holds counts of various operations on and problems with the files
type status struct {
	fileErr, compared, skipped, diffErr, deleted, reverted, leftUntouched int
}

func main() {
	ps := paramset.NewOrDie(addParams,
		verbose.AddParams,
		SetGlobalConfigFile,
		SetConfigFile,
		param.SetProgramDescription(
			"this finds any files in the given directory"+
				" (by default: "+dfltDir+") with the given extension"+
				" (by default: "+dfltExtension+"). It presents each"+
				" file and gives the user the chance to compare it"+
				" with the corresponding file without the"+
				" extension. The user is then asked whether to"+
				" remove the file with the extension."),
	)

	ps.Parse()

	filenames := getFiles()
	maxNameLen := getMaxNameLen(filenames)
	indent := maxNameLen + 2

	fileChks := filecheck.Provisos{
		Existence: filecheck.MustExist,
		Checks: []check.FileInfo{
			check.FileInfoIsRegular,
		},
	}

	showDiffResp := responder.NewOrPanic(
		"Show differences",
		map[rune]string{
			'y': "to show differences",
			'n': "to skip this file",
			'q': "to quit",
		},
		responder.SetDefault('y'),
		responder.SetIndents(0, indent))

	twc := twrap.NewTWConfOrPanic()

	var s status
	fmt.Println(len(filenames), " files found")
fileLoop:
	for _, nameOrig := range filenames {
		prefix := fmt.Sprintf("%*.*s: ", maxNameLen, maxNameLen, nameOrig)

		nameNew := strings.TrimSuffix(nameOrig, extension)
		err := fileChks.StatusCheck(nameNew)
		if err != nil {
			twc.WrapPrefixed(prefix,
				fmt.Sprintf(
					"problem with the other file: %q: %v",
					filepath.Base(nameNew), err),
				0)
			twc.Wrap("Skipping...", indent)
			s.fileErr++
			continue
		}
		fmt.Print(prefix)

		response := showDiffResp.GetResponseOrDie()
		fmt.Println()

		switch response {
		case 'y':
			err = showDiffs(nameOrig, nameNew)
			if err != nil {
				twc.Wrap(fmt.Sprintf("Error: %v", err), indent)
				verboseMsg(twc, "Skipping...", indent)
				s.diffErr++
				continue
			}
			s.compared++

			queryDeleteFile(nameOrig, nameNew, twc, indent, &s)
		case 'n':
			verboseMsg(twc, "Skipping...", indent)
			s.skipped++
		case 'q':
			verboseMsg(twc, "Quitting...", indent)
			break fileLoop
		}
	}

	fmt.Println()
	fmt.Printf("%3d files\n", len(filenames))
	if s.fileErr > 0 {
		fmt.Printf("%3d file errors\n", s.fileErr)
	}
	if s.diffErr > 0 {
		fmt.Printf("%3d diff errors\n", s.diffErr)
	}
	fmt.Printf("%3d skipped\n", s.skipped)
	fmt.Printf("%3d compared\n", s.compared)
	fmt.Println()
	if s.deleted > 0 {
		fmt.Printf("%3d deleted\n", s.deleted)
	}
	if s.reverted > 0 {
		fmt.Printf("%3d reverted\n", s.reverted)
	}
	if s.leftUntouched > 0 {
		fmt.Printf("%3d left untouched\n", s.leftUntouched)
	}
}

// queryDeleteFile will ask if the file should be deleted and then act
// accordingly, reporting any errors found
func queryDeleteFile(nameOrig, nameNew string, twc *twrap.TWConf, indent int, s *status) {
	deleteFileResp := responder.NewOrPanic(
		"delete file",
		map[rune]string{
			'y': "to delete this file",
			'n': "to keep this file",
			'r': "to revert this file to its original contents",
		},
		responder.SetDefault('n'),
		responder.SetIndents(indent, indent))

	response := deleteFileResp.GetResponseOrDie()
	fmt.Println()

	switch response {
	case 'y':
		deleteFile(nameOrig, twc, indent)
		s.deleted++
	case 'r':
		revertFile(nameOrig, nameNew, twc, indent)
		s.reverted++
	default:
		s.leftUntouched++
	}
}

// deleteFile deletes the named file, reporting any errors
func deleteFile(name string, twc *twrap.TWConf, indent int) {
	verboseMsg(twc, "Deleting file...", indent)

	err := os.Remove(name)
	if err != nil {
		twc.Wrap(
			fmt.Sprintf("Couldn't delete the file: %v", err),
			indent)
		return
	}

	verboseMsg(twc, "File deleted", indent)
}

// revertFile reverts the file to its original contents, reporting any
// errors.
func revertFile(nameOrig, nameNew string, twc *twrap.TWConf, indent int) {
	verboseMsg(twc, "Reverting to the original file...", indent)

	err := os.Rename(nameOrig, nameNew)
	if err != nil {
		twc.Wrap(
			fmt.Sprintf("Couldn't revert the file: %v", err),
			indent)
		return
	}

	verboseMsg(twc, "File reverted", indent)
}

// verboseMsg Wraps the message if verbose messaging is on
func verboseMsg(twc *twrap.TWConf, msg string, indent int) {
	if verbose.IsOn() {
		twc.Wrap(msg, indent)
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
		check.FileInfoName(check.StringHasSuffix(extension)),
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
