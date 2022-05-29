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

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/english.mod/english"
	"github.com/nickwells/errutil.mod/errutil"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/param.mod/v5/param/psetter"
	"github.com/nickwells/twrap.mod/twrap"
	"github.com/nickwells/verbose.mod/verbose"
)

// Created: Wed May 26 22:30:48 2021

const (
	installAction = "install"
	cmpAction     = "cmp"

	dfltMaxSubDirs = 10
)

var (
	fromDir string
	toDir   string
	action  = cmpAction

	maxSubDirs int64 = dfltMaxSubDirs
	noCopy     bool
)

type snippet struct {
	content []byte
	dirName string
	name    string
}

type sSet struct {
	files map[string]snippet
	names []string
}

// installer holds the details needed to install snippets
type installer struct {
	srcSet  sSet
	targSet sSet

	toDir string

	l logger

	timestamp string
}

// newInstaller ...
func newInstaller(source, target fs.FS, toDir string) *installer {
	return &installer{
		srcSet:    getFSContent(source, "Snippet source"),
		targSet:   getFSContent(target, "Snippet target"),
		toDir:     toDir,
		l:         logger{errs: errutil.NewErrMap()},
		timestamp: time.Now().Format(".20060102-150405.000"),
	}
}

// checkSourceSnippets checks that there are some snippets in the source
// set. If not it reports an error and exits.
func (inst installer) checkSourceSnippets() {
	if len(inst.srcSet.names) == 0 {
		actName := "unknown action"
		switch action {
		case installAction:
			actName = "install"
		case cmpAction:
			actName = "compare"
		}
		fmt.Fprintln(os.Stderr, "There are no snippets to "+actName)
		os.Exit(1)
	}
}

// reportSnippetCounts reports the number of snippets in the source and
// target sets if verbose is on.
func (inst installer) reportSnippetCounts() {
	if !verbose.IsOn() {
		return
	}

	fmt.Printf("       snippets to install:%4d\n", len(inst.srcSet.names))
	if len(inst.targSet.names) == 0 {
		return
	}
	fmt.Printf("snippets already installed:%4d\n", len(inst.targSet.names))
}

// logger is used to record the progress of the installation
type logger struct {
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
func (l *logger) handleErr(err error, errCat, snippetName string) bool {
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
func (l logger) report(dir string) {
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

		if noCopy {
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
				8)
		} else {
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
				8)

			if l.timestampedCount > 0 {
				twc.Wrap("\nNote that some files have a timestamped copy"+
					" indicating that there were previous copies kept."+
					" You should consider cleaning up these old copies.", 0)
			}
		}
	}
}

// reportErrors prints any errors
func (l logger) reportErrors() {
	twc := twrap.NewTWConfOrPanic(twrap.SetWriter(os.Stderr))

	if len(l.badInstalls) > 0 {
		twc.Wrap("The following snippets could not be installed", 0)
		twc.List(l.badInstalls, 8)
	}

	if l.errs.HasErrors() {
		l.errs.Report(os.Stderr, "Installing snippets")
	}
}

//go:embed _snippets
var snippetsDir embed.FS

func main() {
	ps := paramset.NewOrDie(
		verbose.AddParams,
		addParams,
		param.SetProgramDescription(
			"This can install the standard collection of useful snippets."+
				" It can also be used to install snippets from a"+
				" directory or to compare two collections of snippets."+
				"\n\n"+
				"The default behaviour is to compare the"+
				" standard collection of snippets with those"+
				" in the given target directory."),
	)
	ps.Parse()

	toFS := createToFS(toDir)
	var fromFS fs.FS
	var err error
	fromFS, err = fs.Sub(snippetsDir, "_snippets")
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Can't make the sub-filesystem for the embedded directory: %v", err)
		os.Exit(1)
	}
	if fromDir != "" {
		fromFS = os.DirFS(fromDir)
	}

	switch action {
	case cmpAction:
		compareSnippets(fromFS, toFS, toDir)
	case installAction:
		installSnippets(fromFS, toFS, toDir)
	}
}

// createToFS will check that the toDir either exists in which case it must
// be a directory or else it does not exist in which case it will be created.
// Any failure to create the directory or the existence as a non-directory
// will be reported and the program will exit.
func createToFS(toDir string) fs.FS {
	exists := filecheck.Provisos{Existence: filecheck.MustExist}

	if exists.StatusCheck(toDir) == nil {
		if filecheck.DirExists().StatusCheck(toDir) != nil {
			fmt.Fprintf(os.Stderr,
				"The target exists but is not a directory: %q\n", toDir)
			os.Exit(1)
		}
		return os.DirFS(toDir)
	}

	verbose.Println("creating the target directory: ", toDir)
	err := os.MkdirAll(toDir, 0o777)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Failed to create the target directory (%q): %v\n", toDir, err)
		os.Exit(1)
	}
	return os.DirFS(toDir)
}

// compareSnippets compares the snippets in the from directory with those in
// the to directory reporting any differences.
func compareSnippets(source, target fs.FS, toDir string) {
	verbose.Println("comparing snippets")

	inst := newInstaller(source, target, toDir)
	inst.checkSourceSnippets()
	inst.reportSnippetCounts()

	for _, name := range inst.srcSet.names {
		fromS := inst.srcSet.files[name]
		if toS, ok := inst.targSet.files[name]; ok {
			if string(toS.content) == string(fromS.content) {
				fmt.Println("Duplicate: ", name)
			} else {
				fmt.Println("  Differs: ", name)
			}
		} else {
			fmt.Println("      New: ", name)
		}
	}
	for _, name := range inst.targSet.names {
		if _, ok := inst.srcSet.files[name]; !ok {
			fmt.Println("    Extra: ", name)
		}
	}
}

// installSnippets installs the snippets from the source directory into
// the target directory, reporting any differences.
func installSnippets(source, target fs.FS, toDir string) {
	verbose.Println("Installing snippets into ", toDir)

	inst := newInstaller(source, target, toDir)
	inst.checkSourceSnippets()
	inst.reportSnippetCounts()

	var err error
	for _, snippetName := range inst.srcSet.names {
		verbose.Println("\tinstalling ", snippetName)
		fromS := inst.srcSet.files[snippetName]
		toS, toFileExists := inst.targSet.files[snippetName]

		fileName := filepath.Join(inst.toDir, snippetName)

		if toFileExists {
			if string(toS.content) == string(fromS.content) {
				// duplicate snippet
				inst.l.dupCount++
				continue
			}
			// changed snippet
			inst.l.diffCount++
			if clearFile(snippetName, fileName, inst) {
				err = writeSnippet(fromS, fileName)
				inst.l.handleErr(err, "Write failure", snippetName)
			}
			continue
		}
		// new snippet
		inst.l.newCount++
		err = inst.makeSubDir(fromS)
		if inst.l.handleErr(err, "Mkdir failure", snippetName) {
			continue
		}
		err = writeSnippet(fromS, fileName)
		if inst.l.handleErr(err, "Write failure", snippetName) {
			continue
		}
	}

	inst.l.report(inst.toDir)
	inst.l.reportErrors()
}

// makeSubDir creates the snippet's corresponding sub-directory in the target
// directory if necessary.
func (inst *installer) makeSubDir(s snippet) error {
	if s.dirName == "" {
		return nil
	}

	subDirName := filepath.Join(inst.toDir, s.dirName)
	if filecheck.DirExists().StatusCheck(subDirName) == nil {
		return nil
	}

	err := os.MkdirAll(subDirName, 0o777)
	if err == nil {
		return nil
	}

	name := subDirName
	exists := filecheck.Provisos{Existence: filecheck.MustExist}
	for exists.StatusCheck(name) != nil &&
		name != inst.toDir {
		name = filepath.Dir(name)
	}

	if name == inst.toDir {
		return err
	}

	if clearFile(s.name, name, inst) {
		return os.MkdirAll(subDirName, 0o777)
	}

	return errors.New("Cannot clear the blocking non-dir: " + name)
}

// clearFile either moves the file aside or removes it. It updates the
// installation log and records any errors. It returns true if there were no
// errors, false otherwise.
func clearFile(snippetName, fileName string, inst *installer) bool {
	inst.l.clearCount++

	if noCopy {
		inst.l.removedFiles = append(inst.l.removedFiles, fileName)
		err := os.Remove(fileName)
		return !inst.l.handleErr(err, "Remove failure", snippetName)
	}

	exists := filecheck.Provisos{Existence: filecheck.MustExist}
	copyName := fileName + ".orig"

	if exists.StatusCheck(copyName) == nil {
		copyName += inst.timestamp
		inst.l.timestampedCount++
	}

	inst.l.renamedFiles = append(inst.l.renamedFiles, copyName)
	err := os.Rename(fileName, copyName)
	return !inst.l.handleErr(err, "Rename failure", snippetName)
}

// writeSnippet creates the named file and writes the snippet into it
func writeSnippet(s snippet, name string) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(s.content)

	return err
}

// getFSContent ...
func getFSContent(f fs.FS, name string) sSet {
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
			readSubDir(f, []string{de.Name()}, &snipSet, errs)
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
	defer file.Close()

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
func readSubDir(f fs.FS, names []string, snips *sSet, errs *errutil.ErrMap) {
	if int64(len(names)) > maxSubDirs {
		errs.AddError("Directories too deep - suspected loop",
			fmt.Errorf(
				"The directories at %q exceed the maximum directory depth (%d)",
				filepath.Join(names...), maxSubDirs))
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
			readSubDir(f, append(names, de.Name()), snips, errs)
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

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	const (
		actionParamName = "action"
	)
	ps.Add(actionParamName,
		psetter.Enum{
			Value: &action,
			AllowedVals: psetter.AllowedVals{
				installAction: "install the default snippets in" +
					" the target directory",
				cmpAction: "compare the default snippets with" +
					" those in the target directory",
			},
		},
		"what action should be performed",
		param.AltNames("a"),
		param.Attrs(param.CommandLineOnly),
	)

	ps.Add("install", psetter.Nil{},
		"install the snippets.",
		param.PostAction(paction.SetString(&action, installAction)),
		param.Attrs(param.CommandLineOnly),
		param.SeeAlso(actionParamName),
	)

	ps.Add("target",
		psetter.Pathname{
			Value: &toDir,
			Checks: []check.String{
				check.StringLenGT(0),
			},
		},
		"set the directory where the snippets are to be copied or compared.",
		param.AltNames("to", "to-dir", "t"),
		param.Attrs(param.CommandLineOnly|param.MustBeSet),
	)

	ps.Add("source",
		psetter.Pathname{
			Value:       &fromDir,
			Expectation: filecheck.DirExists(),
		},
		"set the directory where the snippets are to be found."+
			" If this is not set then the standard collection of"+
			" snippets will be used.",
		param.AltNames("from", "from-dir", "f"),
		param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
	)

	ps.Add("max-sub-dirs",
		psetter.Int64{
			Value:  &maxSubDirs,
			Checks: []check.Int64{check.Int64GT(2)},
		},
		"how many levels of sub-directory are allowed before we assume"+
			" there is a loop in the directory path",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("no-copy", psetter.Bool{Value: &noCopy},
		"suppress the copying of existing files which have"+
			" changed and are being replaced."+
			"\n\n"+
			"NOTE: this deletes files from the target directory"+
			" which have the same name as files from the source."+
			" The original files cannot be recovered, no copy is kept.",
		param.AltNames("no-backup"),
		param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
	)

	ps.AddReference("findCmpRm",
		"A program to find files with a given suffix and compare"+
			" them with corresponding files without the suffix."+
			" This can be useful to compare the installed snippets"+
			" with differing versions of the same snippet moved"+
			" aside during the installation. It will prompt the"+
			" user after any differences have been shown to remove"+
			" the copy of the file. It is thus useful for cleaning"+
			" up the snippet directory after installation."+
			"\n\n"+
			"This can be found in the same repository as gosh and"+
			" this command. You can install this with 'go install'"+
			" in the same way as these commands.")

	ps.AddExample(
		`snipDir=$HOME/.config/github.com/nickwells/utilities/gosh/snippets
gosh.snippet -target $snipDir`,
		"This will compare the standard collection of snippets"+
			" with those in the target directory")

	ps.AddExample(
		`snipDir=$HOME/.config/github.com/nickwells/utilities/gosh/snippets
gosh.snippet -target $snipDir -install`,
		"This will install the standard collection of snippets"+
			" into the target directory")

	return nil
}
