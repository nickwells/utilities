package main

// gosh.snippet

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/nickwells/check.mod/check"
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
	action  string = cmpAction

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
	fromSet sSet
	toSet   sSet

	toDir string

	l logger

	timestamp string
}

// logger is used to record the progress of the installation
type logger struct {
	newCount         int
	dupCount         int
	diffCount        int
	timestampedCount int

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

// report prints information about the state of the installation
func (l logger) report() {
	if verbose.IsOn() {
		fmt.Println("Snippet installation summary")
		fmt.Printf("\t        New:%4d\n", l.newCount)
		fmt.Printf("\t  Duplicate:%4d\n", l.dupCount)
		fmt.Printf("\t    Changed:%4d\n", l.diffCount)
		fmt.Printf("\tTimestamped:%4d\n", l.timestampedCount)
		fmt.Printf("\t   Failures:%4d\n", len(l.badInstalls))
	}

	twc := twrap.NewTWConfOrPanic()

	if l.diffCount > 0 {
		if l.diffCount == 1 {
			fmt.Println("One snippet was changed")
		} else {
			fmt.Printf("%d existing snippets were changed\n", l.diffCount)
		}
		if !noCopy {
			twc.Wrap("You should check that you are happy with the changes"+
				" and if so, remove the copies of the original snippet"+
				" files. You might find the 'findCmpRm' tool useful for"+
				" this.", 0)
			fmt.Println()
			fmt.Println("The copies of the files are:")
			twc.List(l.renamedFiles, 8)

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

	if errCount, _ := l.errs.CountErrors(); errCount != 0 {
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

	var toFS fs.FS = createToFS(toDir)
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
		compareSnippets(fromFS, toFS)
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
	err := os.MkdirAll(toDir, 0777)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Failed to create the target directory (%q): %v\n", toDir, err)
		os.Exit(1)
	}
	return os.DirFS(toDir)
}

// compareSnippets compares the snippets in the from directory with those in
// the to directory reporting any differences.
func compareSnippets(from, to fs.FS) {
	verbose.Println("comparing snippets")

	fromSnippets := getFSContent(from, "Snippet source")
	if len(fromSnippets.names) == 0 {
		fmt.Fprintln(os.Stderr, "There are no snippets in the source directory")
		return
	}

	toSnippets := getFSContent(to, "Snippet target")
	if len(toSnippets.names) == 0 {
		fmt.Fprintln(os.Stderr, "There are no snippets in the target directory")
		return
	}

	for _, name := range fromSnippets.names {
		fromS := fromSnippets.files[name]
		if toS, ok := toSnippets.files[name]; ok {
			if string(toS.content) == string(fromS.content) {
				fmt.Println("Duplicate: ", name)
			} else {
				fmt.Println("  Differs: ", name)
			}
		} else {
			fmt.Println("      New: ", name)
		}
	}
	for _, name := range toSnippets.names {
		if _, ok := fromSnippets.files[name]; !ok {
			fmt.Println("    Extra: ", name)
		}
	}
}

// installSnippets installs the snippets in the from directory into
// the to directory reporting any differences.
func installSnippets(from, to fs.FS, toDir string) {
	verbose.Println("Installing snippets into ", toDir)

	inst := &installer{
		fromSet:   getFSContent(from, "Snippet source"),
		toSet:     getFSContent(to, "Snippet target"),
		toDir:     toDir,
		l:         logger{errs: errutil.NewErrMap()},
		timestamp: time.Now().Format(".20060102-150405.000"),
	}
	if len(inst.fromSet.names) == 0 {
		fmt.Fprintln(os.Stderr, "There are no snippets to install")
		return
	}
	verbose.Println(
		fmt.Sprintf("%d snippets to install", len(inst.fromSet.names)))

	if len(inst.toSet.names) > 0 {
		verbose.Println(fmt.Sprintf("snippets in the target directory: %d",
			len(inst.toSet.names)))
	}

	var err error
	for _, snippetName := range inst.fromSet.names {
		verbose.Println("\tinstalling ", snippetName)
		fromS := inst.fromSet.files[snippetName]
		toS, toFileExists := inst.toSet.files[snippetName]

		var (
			dirName  = filepath.Join(toDir, fromS.dirName)
			fileName = filepath.Join(toDir, snippetName)
		)

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
		if fromS.dirName != "" {
			if filecheck.DirExists().StatusCheck(dirName) != nil {
				// TODO: walk back up the dirName (using filepath.Dir) until
				// you get to the toDir which you know exists. We're dealing
				// with the case where you want to create a/b/c/d but a/b/c
				// is a file
				err = os.MkdirAll(dirName, 0777)
				if inst.l.handleErr(err, "Mkdir failure", snippetName) {
					continue
				}
			}
		}
		err = writeSnippet(fromS, fileName)
		if inst.l.handleErr(err, "Write failure", snippetName) {
			continue
		}
	}

	inst.l.report()
	inst.l.reportErrors()
}

// clearFile either moves the file aside or removes it. It updates the
// installation log and records any errors. It returns true if there were no
// errors, false otherwise.
func clearFile(snippetName, fileName string, inst *installer) bool {
	if noCopy {
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
