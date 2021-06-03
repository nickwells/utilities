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
}

type sSet struct {
	files map[string]snippet
	names []string
}

type iLog struct {
	newCount         int
	dupCount         int
	diffCount        int
	timestampedCount int

	movedAsideFiles []string
	failedInstalls  []string
}

//go:embed _snippets
var snippetsDir embed.FS

func main() {
	ps := paramset.NewOrDie(
		verbose.AddParams,
		addParams,
		param.SetProgramDescription(
			"This will install a collection of useful snippets."+
				" It can also be used to copy snippets from one"+
				" directory to another"+
				" or to compare the contents of"+
				" the source and target directories."+
				" The default behaviour is to compare the contents"+
				" of the standard collection of snippets with the"+
				" contents of the supplied target directory."),
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

	fromSnippets, errs := getFSContent(from)
	if errCount, _ := errs.CountErrors(); errCount != 0 {
		errs.Report(os.Stderr, "Snippet source")
		os.Exit(1)
	}
	if len(fromSnippets.names) == 0 {
		fmt.Fprintln(os.Stderr, "There are no snippets in the source directory")
		return
	}

	toSnippets, errs := getFSContent(to)
	if errCount, _ := errs.CountErrors(); errCount != 0 {
		errs.Report(os.Stderr, "Snippet target")
		os.Exit(1)
	}
	if len(toSnippets.names) == 0 {
		fmt.Println("There are no snippets in the target directory")
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

	fromSnippets, errs := getFSContent(from)
	if errCount, _ := errs.CountErrors(); errCount != 0 {
		errs.Report(os.Stderr, "Snippet source")
		os.Exit(1)
	}
	if len(fromSnippets.names) == 0 {
		fmt.Fprintln(os.Stderr, "There are no snippets to install")
		return
	}
	verbose.Println(fmt.Sprintf("%d snippets to install",
		len(fromSnippets.names)))

	toSnippets, errs := getFSContent(to)
	if errCount, _ := errs.CountErrors(); errCount != 0 {
		errs.Report(os.Stderr, "Snippet target")
		os.Exit(1)
	}
	if len(toSnippets.names) > 0 {
		verbose.Println(
			fmt.Sprintf("%d snippets already in the target directory",
				len(toSnippets.names)))
	}

	var stats iLog

	timestamp := time.Now().Format(".20060102-150405.000")
	dirExists := filecheck.DirExists()
	exists := filecheck.Provisos{Existence: filecheck.MustExist}
	var err error
	for _, fName := range fromSnippets.names {
		verbose.Println("\tinstalling ", fName)
		fromS := fromSnippets.files[fName]
		toS, ok := toSnippets.files[fName]

		var (
			dirName       = filepath.Join(toDir, fromS.dirName)
			fullName      = filepath.Join(toDir, fName)
			moveAsideName = fullName + ".orig"
		)

		if ok {
			if string(toS.content) == string(fromS.content) {
				// the exact same snippet already exists
				// - no further action needed
				stats.dupCount++
				continue
			}
			// the snippet exists but it's changed
			// - move the current snippet aside (unless no-copy is set)
			// - write the new snippet
			stats.diffCount++
			if noCopy {
				err = os.Remove(fullName)
				if err != nil {
					errs.AddError("Remove failure", err)
					stats.failedInstalls = append(stats.failedInstalls, fName)
					continue
				}
			} else {
				if exists.StatusCheck(moveAsideName) == nil {
					moveAsideName += timestamp
					stats.timestampedCount++
				}

				stats.movedAsideFiles = append(stats.movedAsideFiles, moveAsideName)
				err = os.Rename(fullName, moveAsideName)
				if err != nil {
					errs.AddError("Rename failure", err)
					stats.failedInstalls = append(stats.failedInstalls, fName)
					continue
				}
			}

			err = writeSnippet(fromS, fullName)
			if err != nil {
				errs.AddError("Write failure", err)
				stats.failedInstalls = append(stats.failedInstalls, fName)
				continue
			}

			continue
		}
		// this is a new snippet
		// - create any necessary directories
		// - move aside any files with the same name as a directory
		// - write the new snippet
		stats.newCount++
		if fromS.dirName != "" {
			if dirExists.StatusCheck(dirName) != nil {
				// TODO: walk back up the dirName (using filepath.Dir) until
				// you get to the toDir which you know exists. We're dealing
				// with the case where you want to create a/b/c/d but a/b/c
				// is a file
				err = os.MkdirAll(dirName, 0777)
				if err != nil {
					errs.AddError("Mkdir failure", err)
					stats.failedInstalls = append(stats.failedInstalls, fName)
					continue
				}
			}
		}
		err = writeSnippet(fromS, fullName)
		if err != nil {
			errs.AddError("Write failure", err)
			stats.failedInstalls = append(stats.failedInstalls, fName)
			continue
		}
	}
	verbose.Println("Snippet installation summary")
	verbose.Println(fmt.Sprintf("\t        New:%4d", stats.newCount))
	verbose.Println(fmt.Sprintf("\t  Duplicate:%4d", stats.dupCount))
	verbose.Println(fmt.Sprintf("\t    Changed:%4d", stats.diffCount))
	verbose.Println(fmt.Sprintf("\tTimestamped:%4d", stats.timestampedCount))
	verbose.Println(fmt.Sprintf("\t   Failures:%4d", len(stats.failedInstalls)))

	twc := twrap.NewTWConfOrPanic()

	if stats.diffCount > 0 {
		if stats.diffCount == 1 {
			fmt.Println("One snippet was changed")
		} else {
			fmt.Printf("%d existing snippets were changed\n", stats.diffCount)
		}
		if !noCopy {
			twc.Wrap("You should check that you are happy with the changes"+
				" and if so, remove the copies of the original snippet"+
				" files. You might find the 'findCmpRm' tool useful for"+
				" this.", 4)
			fmt.Println()
			fmt.Println("The copies of the files are:")
			twc.List(stats.movedAsideFiles, 8)

			if stats.timestampedCount > 0 {
				twc.Wrap("\nNote that some files have a timestamped copy"+
					" indicating that there were previous copies kept."+
					" You should consider cleaning up these old copies.", 4)

			}
		}
	}

	if len(stats.failedInstalls) > 0 {
		twc.Wrap("The following files could not be installed", 0)
		twc.List(stats.failedInstalls, 8)
	}
	if errCount, _ := errs.CountErrors(); errCount != 0 {
		errs.Report(os.Stderr, "Installing snippets")
		os.Exit(1)
	}
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
func getFSContent(f fs.FS) (snips sSet, errs *errutil.ErrMap) {
	errs = errutil.NewErrMap()
	snips = sSet{
		files: map[string]snippet{},
	}

	dirEnts, err := fs.ReadDir(f, ".")
	if err != nil {
		errs.AddError("ReadDir", err)
		return
	}

	for _, de := range dirEnts {
		if de.IsDir() {
			readSubDir(f, []string{de.Name()}, &snips, errs)
			continue
		}
		err := addSnippet(f, de, []string{}, &snips)
		if err != nil {
			errs.AddError("addSnippet", err)
			continue
		}
	}
	return
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
func addSnippet(f fs.FS, de fs.DirEntry, names []string, snips *sSet) error {
	s, err := readSnippet(f, de)
	if err != nil {
		return err
	}

	s.dirName = filepath.Join(names...)
	filename := filepath.Join(s.dirName, de.Name())
	snips.files[filename] = s
	snips.names = append(snips.names, filename)
	return nil
}

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	ps.Add("action",
		psetter.Enum{
			Value: &action,
			AllowedVals: psetter.AllowedVals{
				installAction: "install the default snippets in" +
					" the given directory",
				cmpAction: "compare the default snippets with" +
					" those in the directory",
			},
		},
		"what action should be performed",
		param.AltNames("a"),
		param.Attrs(param.CommandLineOnly),
	)

	ps.Add("install", psetter.Nil{},
		"install the snippets",
		param.PostAction(paction.SetString(&action, installAction)),
		param.Attrs(param.CommandLineOnly),
	)

	ps.Add("to",
		psetter.Pathname{
			Value: &toDir,
			Checks: []check.String{
				check.StringLenGT(0),
			},
		},
		"set the directory where the snippets are to be copied.",
		param.AltNames("to-dir", "target", "t"),
		param.Attrs(param.CommandLineOnly|param.MustBeSet),
	)

	ps.Add("from",
		psetter.Pathname{
			Value:       &fromDir,
			Expectation: filecheck.DirExists(),
		},
		"set the directory where the snippets are to be found."+
			" If this is not set then the default snippet set will be used",
		param.AltNames("from-dir", "source", "f"),
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
		"this will suppress the copying of existing files which have"+
			" changed and are being replaced.",
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

	return nil
}
