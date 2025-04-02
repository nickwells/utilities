package main

// gosh.snippet

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nickwells/english.mod/english"
	"github.com/nickwells/errutil.mod/errutil"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/twrap.mod/twrap"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Wed May 26 22:30:48 2021

const (
	installAction = "install"
	cmpAction     = "compare"
)

const (
	dfltMaxSubDirs = 10
)

const (
	listItemIndent = 8
)

const (
	dfltDirPerms = 0o755 // User: Read/Write/Search, the rest: Read/Search
)

// prog holds program data and parameter values
type prog struct {
	fromDir string
	toDir   string
	action  string

	maxSubDirs int64
	noCopy     bool

	status Status

	timestamp string

	sourceFS fs.FS
	targetFS fs.FS

	sourceSnippets sSet
	targetSnippets sSet
}

// newProg creates an initialised Prog struct
func newProg() *prog {
	return &prog{
		action:     cmpAction,
		maxSubDirs: dfltMaxSubDirs,

		status: Status{
			errs: errutil.NewErrMap(),
		},

		timestamp: time.Now().Format(".20060102-150405.000"),
	}
}

// getFileSystems populates the source and target file systems and gets their
// content
func (prog *prog) getFileSystems() {
	prog.createTargetFS()

	if prog.fromDir != "" {
		prog.sourceFS = os.DirFS(prog.fromDir)
		return
	}

	var err error

	prog.sourceFS, err = fs.Sub(snippetsDir, "_snippets")
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Can't make the sub-filesystem for the embedded directory: %v", err)
		os.Exit(1)
	}
}

// getSnippetSets populates the source and target snippet sets from the
// corresponding file systems
func (prog *prog) getSnippetSets() {
	prog.sourceSnippets = prog.getFSContent(prog.sourceFS, "Snippet source")
	if len(prog.sourceSnippets.names) == 0 {
		fmt.Fprintln(os.Stderr, "There are no snippets to "+prog.action)
		os.Exit(1)
	}

	prog.targetSnippets = prog.getFSContent(prog.targetFS, "Snippet target")
	prog.reportSnippetCounts()
}

type snippet struct {
	content []byte
	dirName string
	name    string
}

type sSet struct {
	files map[string]snippet
	names []string
}

// reportSnippetCounts reports the number of snippets in the source and
// target sets if verbose is on.
func (prog *prog) reportSnippetCounts() {
	if !verbose.IsOn() {
		return
	}

	sourceCount := len(prog.sourceSnippets.names)
	targetCount := len(prog.targetSnippets.names)

	fmt.Printf("       snippets to install:%4d\n", sourceCount)

	if targetCount == 0 {
		return
	}

	fmt.Printf("snippets already installed:%4d\n", targetCount)
}

// Status is used to record the progress of the installation
type Status struct {
	newCount         int
	dupCount         int
	diffCount        int
	clearCount       int
	timestampedCount int

	removedFiles []string
	renamedFiles []string
	badInstalls  []string

	errs *errutil.ErrMap
}

// handleErr checks the error, if it is nil, it returns false, otherwise it
// adds the error to the error map, records that the file failed to install
// and returns true.
func (l *Status) handleErr(err error, errCat, snippetName string) bool {
	if err == nil {
		return false
	}

	l.errs.AddError(errCat, err)
	l.badInstalls = append(l.badInstalls, snippetName)

	return true
}

// trimPrefix returns a slice with the same entries as in vals but with the
// prefix removed from each entry.
func trimPrefix(vals []string, prefix string) []string {
	rval := make([]string, 0, len(vals))

	for _, v := range vals {
		rval = append(rval, strings.TrimPrefix(v, prefix))
	}

	return rval
}

// report prints information about the state of the installation
func (l Status) report(dir string) {
	if verbose.IsOn() {
		fmt.Println("Snippet installation summary")
		fmt.Printf("\t        New:%4d\n", l.newCount)
		fmt.Printf("\t  Duplicate:%4d\n", l.dupCount)
		fmt.Printf("\t    Changed:%4d\n", l.diffCount)
		fmt.Printf("\tTimestamped:%4d\n", l.timestampedCount)
		fmt.Printf("\t   Failures:%4d\n", len(l.badInstalls))
	}

	twc := twrap.NewTWConfOrPanic()

	if l.clearCount > 0 {
		fmt.Printf("Existing snippets cleared:%4d\n", l.clearCount)
		fmt.Printf("         snippets changed:%4d\n", l.diffCount)

		if l.timestampedCount > 0 {
			fmt.Printf("       Timestamped copies:%4d\n", l.timestampedCount)
		}

		if len(l.removedFiles) > 0 {
			twc.Wrap("The following files were removed. Please check that"+
				" you are happy with this; if not you will need to restore"+
				" from backups (if available)", 0)
			fmt.Println()
			fmt.Println(len(l.removedFiles),
				english.Plural("file", len(l.removedFiles)),
				"removed")
			fmt.Println("in", dir)
			twc.List(
				trimPrefix(l.removedFiles, dir+string(filepath.Separator)),
				listItemIndent)
		}

		if len(l.renamedFiles) > 0 {
			twc.Wrap("You should check that you don't want to keep the"+
				" original files"+
				" and if so, remove the copies of the original snippet"+
				" files. You might find the 'findCmpRm' tool useful for"+
				" this.", 0)
			fmt.Println()
			fmt.Println(len(l.renamedFiles),
				english.Plural("file", len(l.renamedFiles)),
				"renamed")
			fmt.Println("in", dir)
			twc.List(
				trimPrefix(l.renamedFiles, dir+string(filepath.Separator)),
				listItemIndent)

			if l.timestampedCount > 0 {
				twc.Wrap("\nNote that some files have a timestamped copy"+
					" indicating that there were previous copies kept."+
					" You should consider cleaning up these old copies.", 0)
			}
		}
	}
}

// reportErrors prints any errors
func (l Status) reportErrors() {
	twc := twrap.NewTWConfOrPanic(twrap.SetWriter(os.Stderr))

	if len(l.badInstalls) > 0 {
		twc.Wrap("The following snippets could not be installed", 0)
		twc.List(l.badInstalls, listItemIndent)
	}

	if l.errs.HasErrors() {
		l.errs.Report(os.Stderr, "Installing snippets")
	}
}

//go:embed _snippets
var snippetsDir embed.FS

func main() {
	prog := newProg()
	ps := makeParamSet(prog)
	ps.Parse()

	prog.getFileSystems()
	prog.getSnippetSets()

	switch prog.action {
	case cmpAction:
		prog.compareSnippets()
	case installAction:
		prog.installSnippets()
	}
}

// createTargetFS will check that the toDir either exists in which case it must
// be a directory or else it does not exist in which case it will be created.
// Any failure to create the directory or the existence as a non-directory
// will be reported and the program will exit.
func (prog *prog) createTargetFS() {
	exists := filecheck.Provisos{Existence: filecheck.MustExist}

	if exists.StatusCheck(prog.toDir) == nil {
		if filecheck.DirExists().StatusCheck(prog.toDir) != nil {
			fmt.Fprintf(os.Stderr,
				"The target exists but is not a directory: %q\n", prog.toDir)
			os.Exit(1)
		}

		prog.targetFS = os.DirFS(prog.toDir)

		return
	}

	verbose.Println("creating the target directory: ", prog.toDir)

	err := os.MkdirAll(prog.toDir, dfltDirPerms)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Failed to create the target directory (%q): %v\n", prog.toDir, err)
		os.Exit(1)
	}

	prog.targetFS = os.DirFS(prog.toDir)
}

// compareSnippets compares the snippets in the from directory with those in
// the to directory reporting any differences.
func (prog *prog) compareSnippets() {
	verbose.Println("comparing snippets")

	for _, name := range prog.sourceSnippets.names {
		fromS := prog.sourceSnippets.files[name]
		if toS, ok := prog.targetSnippets.files[name]; ok {
			if string(toS.content) == string(fromS.content) {
				fmt.Println("Duplicate: ", name)
			} else {
				fmt.Println("  Differs: ", name)
			}
		} else {
			fmt.Println("      New: ", name)
		}
	}

	for _, name := range prog.targetSnippets.names {
		if _, ok := prog.sourceSnippets.files[name]; !ok {
			fmt.Println("    Extra: ", name)
		}
	}
}

// installSnippets installs the snippets from the source directory into
// the target directory, reporting any differences.
func (prog *prog) installSnippets() {
	verbose.Println("Installing snippets into ", prog.toDir)

	var err error

	for _, snippetName := range prog.sourceSnippets.names {
		verbose.Println("\tinstalling ", snippetName)
		fromS := prog.sourceSnippets.files[snippetName]
		toS, toFileExists := prog.targetSnippets.files[snippetName]

		fileName := filepath.Join(prog.toDir, snippetName)

		if toFileExists {
			if string(toS.content) == string(fromS.content) {
				// duplicate snippet
				prog.status.dupCount++
				continue
			}
			// changed snippet
			prog.status.diffCount++
			if prog.clearFile(snippetName, fileName) {
				err = writeSnippet(fromS, fileName)
				prog.status.handleErr(err, "Write failure", snippetName)
			}

			continue
		}
		// new snippet
		prog.status.newCount++

		err = prog.makeSubDir(fromS)
		if prog.status.handleErr(err, "Mkdir failure", snippetName) {
			continue
		}

		err = writeSnippet(fromS, fileName)
		if prog.status.handleErr(err, "Write failure", snippetName) {
			continue
		}
	}

	prog.status.report(prog.toDir)
	prog.status.reportErrors()
}

// makeSubDir creates the snippet's corresponding sub-directory in the target
// directory if necessary.
func (prog *prog) makeSubDir(s snippet) error {
	if s.dirName == "" {
		return nil
	}

	subDirName := filepath.Join(prog.toDir, s.dirName)
	if filecheck.DirExists().StatusCheck(subDirName) == nil {
		return nil
	}

	err := os.MkdirAll(subDirName, dfltDirPerms)
	if err == nil {
		return nil
	}

	name := subDirName

	exists := filecheck.Provisos{Existence: filecheck.MustExist}
	for exists.StatusCheck(name) != nil &&
		name != prog.toDir {
		name = filepath.Dir(name)
	}

	if name == prog.toDir {
		return err
	}

	if prog.clearFile(s.name, name) {
		return os.MkdirAll(subDirName, dfltDirPerms)
	}

	return errors.New("Cannot clear the blocking non-dir: " + name)
}

// clearFile either moves the file aside or removes it. It updates the
// installation log and records any errors. It returns true if there were no
// errors, false otherwise.
func (prog *prog) clearFile(snippetName, fileName string) bool {
	prog.status.clearCount++

	if prog.noCopy {
		prog.status.removedFiles = append(prog.status.removedFiles, fileName)
		err := os.Remove(fileName)

		return !prog.status.handleErr(err, "Remove failure", snippetName)
	}

	exists := filecheck.Provisos{Existence: filecheck.MustExist}
	copyName := fileName + ".orig"

	if exists.StatusCheck(copyName) == nil {
		copyName += prog.timestamp
		prog.status.timestampedCount++
	}

	prog.status.renamedFiles = append(prog.status.renamedFiles, copyName)

	err := os.Rename(fileName, copyName)

	return !prog.status.handleErr(err, "Rename failure", snippetName)
}

// writeSnippet creates the named file and writes the snippet into it
func writeSnippet(s snippet, name string) error {
	f, err := os.Create(name) //nolint:gosec
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(s.content)

	return err
}

// getFSContent gets the contents of the supplied filesystem, recursively
// descending into subdirectories. It returns the results as a snippet set.
func (prog *prog) getFSContent(f fs.FS, name string) sSet {
	errs := errutil.NewErrMap()
	defer func() {
		if errCount, _ := errs.CountErrors(); errCount != 0 {
			errs.Report(os.Stderr, name)
			os.Exit(1)
		}
	}()

	snipSet := sSet{
		files: map[string]snippet{},
	}

	dirEnts, err := fs.ReadDir(f, ".")
	if err != nil {
		errs.AddError("ReadDir", err)
		return snipSet
	}

	for _, de := range dirEnts {
		if de.IsDir() {
			prog.readSubDir(f, []string{de.Name()}, &snipSet, errs)
			continue
		}

		err := addSnippet(f, de, []string{}, &snipSet)
		if err != nil {
			errs.AddError("addSnippet", err)
			continue
		}
	}

	return snipSet
}

// readSnippet reads the snippet contents from the FS
func readSnippet(f fs.FS, de fs.DirEntry) (snippet, error) {
	s := snippet{}

	fi, err := de.Info()
	if err != nil {
		return s, err
	}

	file, err := f.Open(de.Name())
	if err != nil {
		return s, err
	}

	defer file.Close() //nolint:errcheck

	s.content = make([]byte, fi.Size())

	_, err = file.Read(s.content)
	if err != nil {
		return s, err
	}

	return s, nil
}

// readSubDir reads the directory, populating the content and recording
// any errors, it will recursively descend into any subdirectories. If the
// total depth of subdirectories is greater than maxSubDirs then it will
// assume that there is a loop in the directory tree and will abort
func (prog *prog) readSubDir(
	f fs.FS, names []string, snips *sSet, errs *errutil.ErrMap,
) {
	if int64(len(names)) > prog.maxSubDirs {
		errs.AddError("Directories too deep - suspected loop",
			fmt.Errorf(
				"the directories at %q exceed the maximum directory depth (%d)",
				filepath.Join(names...), prog.maxSubDirs))

		return
	}

	f, err := fs.Sub(f, names[len(names)-1])
	if err != nil {
		errs.AddError("Cannot construct the sub-filesystem", err)
		return
	}

	dirEnts, err := fs.ReadDir(f, ".")
	if err != nil {
		errs.AddError("ReadDir", err)
		return
	}

	for _, de := range dirEnts {
		if de.IsDir() {
			prog.readSubDir(f, append(names, de.Name()), snips, errs)
			continue
		}

		err := addSnippet(f, de, names, snips)
		if err != nil {
			errs.AddError("addSnippet", err)
			continue
		}
	}
}

// addSnippet reads the snippet file and adds it to the snippet set. It
// records any erros detected.
func addSnippet(f fs.FS, de fs.DirEntry, names []string, snipSet *sSet) error {
	s, err := readSnippet(f, de)
	if err != nil {
		return err
	}

	s.dirName = filepath.Join(names...)
	s.name = filepath.Join(s.dirName, de.Name())
	snipSet.files[s.name] = s
	snipSet.names = append(snipSet.names, s.name)

	return nil
}
