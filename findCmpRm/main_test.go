package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/cli.mod/cli/responder"
	"github.com/nickwells/dirsearch.mod/v2/dirsearch"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
	"github.com/nickwells/twrap.mod/twrap"
)

// fileInfo holds the details of a file to create
type fileInfo struct {
	isDir         bool
	isNotReadable bool
	contents      string
}

// filePairInfo holds details needed to create a file-pair
type filePairInfo struct {
	name           string
	origDetails    *fileInfo
	nonOrigDetails *fileInfo
}

// setProg tests the setProg func and if it is not nil it will call it on the
// prog variable. If the set function returns an error it is reported as
// a fatal error
func setProg(t *testing.T, name string, prog *prog, setP func(*prog) error) {
	t.Helper()

	if setP != nil {
		err := setP(prog)
		if err != nil {
			t.Log(name)
			t.Fatal("\t: unexpected setProg Error: ", err)
		}
	}
}

// makeFile creates the file (or directory) with the given name. If fd is nil
// it returns with a nil error
func (fi *fileInfo) makeFile(name string) error {
	if fi == nil {
		return nil
	}

	var perm fs.FileMode = 0o700
	if fi.isNotReadable {
		perm = 0o300
	}

	if fi.isDir {
		return os.MkdirAll(name, perm)
	}

	f, err := os.Create(name) //nolint:gosec
	if err != nil {
		return err
	}

	defer func() { f.Close() }()

	err = f.Chmod(perm)
	if err != nil {
		return err
	}

	_, err = f.WriteString(fi.contents)
	if err != nil {
		return err
	}

	return nil
}

// makeFilePair creates a pair of files, one with the name and one with the
// name plus the extension
func (fpi filePairInfo) makeFilePair(dir, extension string) error {
	baseName := filepath.Join(dir, fpi.name)
	if err := fpi.origDetails.makeFile(baseName + extension); err != nil {
		return err
	}

	return fpi.nonOrigDetails.makeFile(baseName)
}

// makeTestDir makes all the given file pairs in the given directory
func makeTestDir(dir, extension string, fpInfo ...filePairInfo) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	for _, fpi := range fpInfo {
		if err := fpi.makeFilePair(dir, extension); err != nil {
			return err
		}
	}

	return nil
}

// getFilesTestCleanup checks that the necessary function is present and runs
// it, checking for errors. Any problem results in a Fatal test.
func getFilesTestCleanup(t *testing.T, name string,
	prog *prog, post func(*prog) error,
) {
	t.Helper()

	if post != nil {
		if err := post(prog); err != nil {
			t.Log(name)
			t.Fatal("test cleanup failed:", err)
		}
	}

	if err := os.RemoveAll(prog.searchDir); err != nil &&
		!errors.Is(err, os.ErrNotExist) {
		t.Log(name)
		t.Fatal("\t: cannot remove the temp dir after the test: ", err)
	}
}

// getFilesTestSetup checks that the necessary functions are present and runs
// them, checking for errors. Any problem results in a Fatal test.
func getFilesTestSetup(t *testing.T, name string,
	prog *prog, pre func(*prog) error,
) {
	t.Helper()

	if pre != nil {
		if err := pre(prog); err != nil {
			t.Log(name)
			t.Fatal("test setup failed:", err)
		}
	}
}

// setupFakeIO constructs a FakeIO and reports any errors
func setupFakeIO(t *testing.T, prog *prog, testname, testpart, input string,
) *testhelper.FakeIO {
	t.Helper()

	fakeIO, err := testhelper.NewStdioFromString(input)
	if err != nil {
		t.Log(testname)
		t.Log("\t:", "creating FakeIO")

		if testpart != "" {
			t.Log("\t:", "for the "+testpart+" files")
		}

		t.Fatal(err)
	}

	if err := twrap.SetWriter(os.Stdout)(prog.twc); err != nil {
		t.Log(testname)
		t.Log("\t:", "setting twrap writer")

		if testpart != "" {
			t.Log("\t:", "for the "+testpart+" files")
		}

		t.Fatal(err)
	}

	return fakeIO
}

// getFakeIO gets the standard out and error streams from the FakeIO,
// reporting any errors found.
func getFakeIO(t *testing.T, testname, testpart string,
	fakeIO *testhelper.FakeIO,
) (stdout, stderr []byte) {
	t.Helper()

	stdout, stderr, err := fakeIO.Done()
	if err != nil {
		t.Log(testname)
		t.Log("\t:", "getting FakeIO results")

		if testpart != "" {
			t.Log("\t: for the " + testpart + " files")
		}

		t.Fatal(err)
	}

	return stdout, stderr
}

func TestGetFiles(t *testing.T) {
	tempTestDir := filepath.Join("testdata", "tempTestDir")

	f1Orig := filepath.Join(tempTestDir, "f1"+dfltExtension)

	f2Orig := filepath.Join(tempTestDir, "f2"+dfltExtension)

	f3Name := filepath.Join(tempTestDir, "f3")
	f3Orig := filepath.Join(tempTestDir, "f3"+dfltExtension)

	f4Orig := filepath.Join(tempTestDir, "f4"+dfltExtension)

	f5Name := filepath.Join(tempTestDir, "f5")
	f5Orig := filepath.Join(tempTestDir, "f5"+dfltExtension)

	f6Name := filepath.Join(tempTestDir, "f6")
	f6Orig := filepath.Join(tempTestDir, "f6"+dfltExtension)

	d1 := filepath.Join(tempTestDir, "d1")

	f7Orig := filepath.Join(d1, "f7"+dfltExtension)

	f8Orig := filepath.Join(d1, "f8"+dfltExtension)

	testCases := []struct {
		testhelper.ID
		setProg       func(prog *prog) error
		pre, post     func(prog *prog) error
		expFilenames  []string
		expDuplicates []string
		expBadFiles   []badFile
		expErrCount   int

		expComparableStdout string
		expComparableStderr string
		expDuplicateStdout  string
		expDuplicateStderr  string
	}{
		{
			ID: testhelper.MkID("bad search directory"),
			setProg: func(prog *prog) error {
				prog.searchDir = filepath.Join("testdata", "nonesuch")
				return nil
			},
			expErrCount: 1,
		},
		{
			ID: testhelper.MkID("good search directory, with errors and dups"),
			setProg: func(prog *prog) error {
				prog.searchDir = tempTestDir
				return nil
			},
			pre: func(prog *prog) error {
				return makeTestDir(prog.searchDir, prog.fileExtension,
					filePairInfo{
						name:           "f1",
						origDetails:    &fileInfo{contents: "Hello"},
						nonOrigDetails: &fileInfo{contents: "Hello"},
					},
					filePairInfo{
						name:           "f2",
						origDetails:    &fileInfo{contents: "Hello"},
						nonOrigDetails: &fileInfo{contents: "World"},
					},
					filePairInfo{
						name:        "f3",
						origDetails: &fileInfo{contents: "Hello"},
						nonOrigDetails: &fileInfo{
							contents:      "World",
							isNotReadable: true,
						},
					},
					filePairInfo{
						name: "f4",
						origDetails: &fileInfo{
							contents:      "Hello",
							isNotReadable: true,
						},
						nonOrigDetails: &fileInfo{contents: "World"},
					},
					filePairInfo{
						name:        "f5",
						origDetails: &fileInfo{contents: "Hello"},
					},
					filePairInfo{
						name:           "f6",
						origDetails:    &fileInfo{contents: "Hello"},
						nonOrigDetails: &fileInfo{isDir: true},
					},
					filePairInfo{
						name:           "d1",
						nonOrigDetails: &fileInfo{isDir: true},
					},
					filePairInfo{
						name:           filepath.Join("d1", "f7"),
						origDetails:    &fileInfo{contents: "Hello"},
						nonOrigDetails: &fileInfo{contents: "World"},
					},
					filePairInfo{
						name:           filepath.Join("d1", "f8"),
						origDetails:    &fileInfo{contents: "Hello"},
						nonOrigDetails: &fileInfo{contents: "Hello"},
					},
				)
			},
			expFilenames:  []string{f7Orig, f2Orig},
			expDuplicates: []string{f8Orig, f1Orig},
			expBadFiles: []badFile{
				{
					name: f3Orig,
					problem: `cannot read "` +
						f3Name + `": open ` + f3Name + `: permission denied`,
				},
				{
					name: f4Orig,
					problem: `cannot read "` +
						f4Orig + `": open ` + f4Orig + `: permission denied`,
				},
				{
					name:    f5Orig,
					problem: `there is no file named "` + f5Name + `"`,
				},
				{
					name:    f6Orig,
					problem: `"` + f6Name + `" is a directory`,
				},
			},
			expComparableStdout: "2 comparable files found\n" +
				"in testdata/tempTestDir\n" +
				"        - 1: d1/f7.orig\n" +
				"        - 2: f2.orig\n",
			expDuplicateStdout: "2 duplicate files found\n" +
				"in testdata/tempTestDir\n" +
				"        - d1/f8.orig\n" +
				"        - f1.orig\n",
		},
		{
			ID: testhelper.MkID("good search directory, no recursive search"),
			setProg: func(prog *prog) error {
				prog.searchDir = tempTestDir
				prog.searchSubDirs = false
				return nil
			},
			pre: func(prog *prog) error {
				return makeTestDir(prog.searchDir, prog.fileExtension,
					filePairInfo{
						name:           "f1",
						origDetails:    &fileInfo{contents: "Hello"},
						nonOrigDetails: &fileInfo{contents: "Hello"},
					},
					filePairInfo{
						name:           "f2",
						origDetails:    &fileInfo{contents: "Hello"},
						nonOrigDetails: &fileInfo{contents: "World"},
					},
					filePairInfo{
						name:           "d1",
						nonOrigDetails: &fileInfo{isDir: true},
					},
					filePairInfo{
						name:           filepath.Join("d1", "f7"),
						origDetails:    &fileInfo{contents: "Hello"},
						nonOrigDetails: &fileInfo{contents: "World"},
					},
					filePairInfo{
						name:           filepath.Join("d1", "f8"),
						origDetails:    &fileInfo{contents: "Hello"},
						nonOrigDetails: &fileInfo{contents: "Hello"},
					},
				)
			},
			expFilenames:  []string{f2Orig},
			expDuplicates: []string{f1Orig},
			expBadFiles:   []badFile{},
			expComparableStdout: "1 comparable file found\n" +
				"in testdata/tempTestDir\n" +
				"        - 1: f2.orig\n",
			expDuplicateStdout: "1 duplicate file found\n" +
				"in testdata/tempTestDir\n" +
				"        - f1.orig\n",
		},
	}

	for _, tc := range testCases {
		prog := newProg()
		setProg(t, tc.IDStr(), prog, tc.setProg)

		getFilesTestSetup(t, tc.IDStr(), prog, tc.pre)

		comparables, duplicates, badFiles, errs := prog.getFiles()

		testhelper.DiffInt(t, tc.IDStr(), "number of errors",
			len(errs), tc.expErrCount)

		if !testhelper.DiffStringSlice(t, tc.IDStr(), "comparable files",
			comparables, tc.expFilenames) {
			fakeIO := setupFakeIO(t, prog, tc.IDStr(), "comparable", "")

			prog.showComparableFiles(comparables)

			stdout, stderr := getFakeIO(t, tc.IDStr(), "comparable", fakeIO)
			testhelper.DiffString(t,
				tc.IDStr(), "comparable files stdout",
				string(stdout), tc.expComparableStdout)
			testhelper.DiffString(t,
				tc.IDStr(), "comparable files stderr",
				string(stderr), tc.expComparableStderr)
		}

		if !testhelper.DiffStringSlice(t, tc.IDStr(), "duplicate files",
			duplicates, tc.expDuplicates) {
			fakeIO := setupFakeIO(t, prog, tc.IDStr(), "duplicate", "")

			prog.showDuplicateFiles(duplicates)

			stdout, stderr := getFakeIO(t, tc.IDStr(), "duplicate", fakeIO)
			testhelper.DiffString(t,
				tc.IDStr(), "duplicate files stdout",
				string(stdout), tc.expDuplicateStdout)
			testhelper.DiffString(t,
				tc.IDStr(), "duplicate files stderr",
				string(stderr), tc.expDuplicateStderr)
		}

		if testhelper.DiffSlice(t, tc.IDStr(), "bad Files",
			badFiles, tc.expBadFiles) {
			t.Log(tc.IDStr())
			t.Log("\t:", badFiles)
			t.Error("\t: unexpected bad files")
		}

		getFilesTestCleanup(t, tc.IDStr(), prog, tc.post)
	}
}

func TestShortNames(t *testing.T) {
	var (
		searchDir   = "testdata"
		f1shortPath = "f1"
		f1path      = filepath.Join(searchDir, f1shortPath)
		f2shortPath = filepath.Join("d1", "f2")
		f2path      = filepath.Join(searchDir, f2shortPath)
	)

	testCases := []struct {
		testhelper.ID
		searchDir     string
		names         []string
		expShortNames []string
		expMaxLen     int
	}{
		{
			ID:            testhelper.MkID("only"),
			searchDir:     "testdata",
			names:         []string{f1path, f2path},
			expMaxLen:     len(f2shortPath),
			expShortNames: []string{f1shortPath, f2shortPath},
		},
	}

	for _, tc := range testCases {
		prog := newProg()
		prog.searchDir = tc.searchDir
		shortnames, maxLen := prog.shortNames(tc.names)
		testhelper.DiffInt(t, tc.IDStr(), "maxLen", maxLen, tc.expMaxLen)
		testhelper.DiffStringSlice(t, tc.IDStr(), "short-names",
			shortnames, tc.expShortNames)
	}
}

func TestProcessDuplicateFiles(t *testing.T) {
	tempTestDir := filepath.Join("testdata", "tempTestDir")
	f1 := filepath.Join(tempTestDir, "f1"+dfltExtension)
	f2 := filepath.Join(tempTestDir, "f2"+dfltExtension)
	f3 := filepath.Join(tempTestDir, "f3"+dfltExtension)

	dupFiles := []string{f1, f2, f3}

	testCases := []struct {
		testhelper.ID
		response     rune
		expFileCount int
		expStdout    string
		expStderr    string
		pre, post    func(prog *prog) error
		setProg      func(prog *prog) error
	}{
		{
			ID:           testhelper.MkID("do not delete dups"),
			response:     'n',
			expFileCount: len(dupFiles),
			expStdout:    "\n\n",
		},
		{
			ID:       testhelper.MkID("delete dups"),
			response: 'y',
			setProg: func(prog *prog) error {
				prog.status.dupFile.deleted = len(dupFiles)
				return nil
			},
			expStdout: "\n3 duplicate files deleted\n\n",
		},
		{
			ID:       testhelper.MkID("delete dups (fail)"),
			response: 'y',
			setProg: func(prog *prog) error {
				prog.status.dupFile.delErrs = len(dupFiles)
				return nil
			},
			expFileCount: len(dupFiles),
			expStdout: "\n" +
				"Couldn't delete the file:" +
				" remove " + f1 + ": permission denied\n" +
				"Couldn't delete the file:" +
				" remove " + f2 + ": permission denied\n" +
				"Couldn't delete the file:" +
				" remove " + f3 + ": permission denied\n" +
				"3 duplicate files could not be deleted\n\n",
			pre: func(_ *prog) error {
				return os.Chmod(tempTestDir, 0o500) //nolint:gosec
			},
			post: func(_ *prog) error {
				return os.Chmod(tempTestDir, 0o700) //nolint:gosec
			},
		},
	}

	for _, tc := range testCases {
		err := makeTestDir(tempTestDir, dfltExtension,
			filePairInfo{
				name:           "f1",
				origDetails:    &fileInfo{contents: "Hello"},
				nonOrigDetails: &fileInfo{contents: "Hello"},
			},
			filePairInfo{
				name:           "f2",
				origDetails:    &fileInfo{contents: "Hello"},
				nonOrigDetails: &fileInfo{contents: "Hello"},
			},
			filePairInfo{
				name:           "f3",
				origDetails:    &fileInfo{contents: "Hello"},
				nonOrigDetails: &fileInfo{contents: "Hello"},
			},
		)
		if err != nil {
			t.Log(tc.IDStr())
			t.Fatal("\t: unexpected makeTestDir error: ", err)
		}

		prog := newProg()
		prog.searchDir = tempTestDir
		prog.deleteDupR = responder.FixedResponse{Response: tc.response}
		prog.status.dupFile.total = len(dupFiles)

		expProg := newProg()
		expProg.searchDir = tempTestDir
		expProg.status.dupFile.total = len(dupFiles)
		setProg(t, tc.IDStr(), expProg, tc.setProg)

		getFilesTestSetup(t, tc.IDStr(), prog, tc.pre)

		fakeIO := setupFakeIO(t, prog, tc.IDStr(), "duplicate", "")

		prog.processDuplicateFiles(dupFiles)

		stdout, stderr := getFakeIO(t, tc.IDStr(), "duplicate", fakeIO)

		testhelper.DiffString(t, tc.IDStr(), "stdout",
			string(stdout), tc.expStdout)
		testhelper.DiffString(t, tc.IDStr(), "stderr",
			string(stderr), tc.expStderr)

		err = testhelper.DiffVals(*prog, *expProg,
			[]string{"deleteDupR"},
			[]string{"twc"})
		if err != nil {
			t.Log(tc.IDStr())
			t.Log("\t: resulting prog values differ: ", err)
			t.Error("\t: unexpected Prog value")
		}

		count, errs := dirsearch.CountRecurse(tempTestDir,
			check.FileInfoIsRegular,
			check.FileInfoName(
				check.StringHasSuffix[string](prog.fileExtension)))
		if len(errs) != 0 {
			t.Log(tc.IDStr())
			t.Log("\t: errors: ", errs)
			t.Error("\t: Unexpected errors counting the files in the dir")

			continue
		}

		testhelper.DiffInt(t, tc.IDStr(), "file count", count, tc.expFileCount)

		getFilesTestCleanup(t, tc.IDStr(), prog, tc.post)
	}
}

func TestProcessComparableFiles(t *testing.T) {
	nosuchDiff := filepath.Join("testdata", "nosuchDiff")

	tempTestDir := filepath.Join("testdata", "tempTestDir")
	f1 := filepath.Join(tempTestDir, "f1"+dfltExtension)
	f2 := filepath.Join(tempTestDir, "f2"+dfltExtension)
	f3 := filepath.Join(tempTestDir, "f3"+dfltExtension)

	cmpFiles := []string{f1, f2, f3}

	testCases := []struct {
		testhelper.ID
		response     rune
		expProg      *prog
		expFileCount int
		expStdout    string
		expStderr    string
		pre, post    func(prog *prog) error
		setProg      func(prog *prog) error
	}{
		{
			ID:       testhelper.MkID("do not compare files"),
			response: 'n',
			expProg: func() *prog {
				prog := newProg()
				prog.status.cmpFile.total = len(cmpFiles)
				return prog
			}(),
			expFileCount: len(cmpFiles),
			expStdout: "    (1 / 3) f1.orig: \n" +
				"    (2 / 3) f2.orig: \n" +
				"    (3 / 3) f3.orig: \n\n",
		},
		{
			ID:       testhelper.MkID("delete all files"),
			response: 'd',
			expProg: func() *prog {
				prog := newProg()
				prog.status.cmpFile.total = len(cmpFiles)
				prog.status.cmpFile.deleted = len(cmpFiles)
				prog.cmpAction = caDeleteAll
				return prog
			}(),
			expStdout: "    (1 / 3) f1.orig: \n\n",
		},
		{
			ID:       testhelper.MkID("revert all files"),
			response: 'r',
			expProg: func() *prog {
				prog := newProg()
				prog.status.cmpFile.total = len(cmpFiles)
				prog.status.cmpFile.reverted = len(cmpFiles)
				prog.cmpAction = caRevertAll
				return prog
			}(),
			expStdout: "    (1 / 3) f1.orig: \n\n",
		},
		{
			ID:       testhelper.MkID("keep all files"),
			response: 'q',
			expProg: func() *prog {
				prog := newProg()
				prog.status.cmpFile.total = len(cmpFiles)
				prog.cmpAction = caKeepAll
				return prog
			}(),
			expFileCount: len(cmpFiles),
			expStdout: "    (1 / 3) f1.orig: \n" +
				"3 comparable files kept\n\n",
		},
		{
			ID:       testhelper.MkID("diff all files (failing)"),
			response: 'y',
			setProg: func(prog *prog) error {
				prog.diff.name = nosuchDiff
				return nil
			},
			expProg: func() *prog {
				prog := newProg()
				prog.status.cmpFile.total = len(cmpFiles)
				prog.status.cmpFile.cmpErrs = len(cmpFiles)
				prog.diff.name = nosuchDiff
				return prog
			}(),
			expFileCount: len(cmpFiles),
			expStdout: "    (1 / 3) f1.orig: \n" +
				"                     Error:" +
				" the diff command (\"testdata/nosuchDiff\n" +
				"                     testdata/tempTestDir/f1.orig" +
				" testdata/tempTestDir/f1\")\n" +
				"                     could not be started:" +
				" fork/exec testdata/nosuchDiff: no\n" +
				"                     such file or directory\n" +
				"    (2 / 3) f2.orig: \n" +
				"                     Error:" +
				" the diff command (\"testdata/nosuchDiff\n" +
				"                     testdata/tempTestDir/f2.orig" +
				" testdata/tempTestDir/f2\")\n" +
				"                     could not be started:" +
				" fork/exec testdata/nosuchDiff: no\n" +
				"                     such file or directory\n" +
				"    (3 / 3) f3.orig: \n" +
				"                     Error:" +
				" the diff command (\"testdata/nosuchDiff\n" +
				"                     testdata/tempTestDir/f3.orig" +
				" testdata/tempTestDir/f3\")\n" +
				"                     could not be started:" +
				" fork/exec testdata/nosuchDiff: no\n" +
				"                     such file or directory\n\n",
		},
	}

	for _, tc := range testCases {
		err := makeTestDir(tempTestDir, dfltExtension,
			filePairInfo{
				name:           "f1",
				origDetails:    &fileInfo{contents: "Hello"},
				nonOrigDetails: &fileInfo{contents: "World"},
			},
			filePairInfo{
				name:           "f2",
				origDetails:    &fileInfo{contents: "Hello"},
				nonOrigDetails: &fileInfo{contents: "World"},
			},
			filePairInfo{
				name:           "f3",
				origDetails:    &fileInfo{contents: "Hello"},
				nonOrigDetails: &fileInfo{contents: "World"},
			},
		)
		if err != nil {
			t.Log(tc.IDStr())
			t.Fatal("\t: unexpected makeTestDir error: ", err)
		}

		prog := newProg()
		prog.searchDir = tempTestDir
		prog.showDiffR = responder.FixedResponse{Response: tc.response}
		prog.status.cmpFile.total = len(cmpFiles)
		setProg(t, tc.IDStr(), prog, tc.setProg)

		getFilesTestSetup(t, tc.IDStr(), prog, tc.pre)

		fakeIO := setupFakeIO(t, prog, tc.IDStr(), "comparable", "")

		prog.processComparableFiles(cmpFiles)

		stdout, stderr := getFakeIO(t, tc.IDStr(), "comparable", fakeIO)

		testhelper.DiffString(t, tc.IDStr(), "stdout",
			string(stdout), tc.expStdout)
		testhelper.DiffString(t, tc.IDStr(), "stderr",
			string(stderr), tc.expStderr)

		err = testhelper.DiffVals(*prog, *tc.expProg,
			[]string{"searchDir"},
			[]string{"showDiffR"},
			[]string{"indent"},
			[]string{"twc"})
		if err != nil {
			t.Log(tc.IDStr())
			t.Log("\t: resulting prog values differ: ", err)
			t.Error("\t: unexpected Prog value")
		}

		count, errs := dirsearch.CountRecurse(tempTestDir,
			check.FileInfoIsRegular,
			check.FileInfoName(
				check.StringHasSuffix[string](prog.fileExtension)))
		if len(errs) != 0 {
			t.Log(tc.IDStr())
			t.Log("\t: errors: ", errs)
			t.Error("\t: Unexpected errors counting the files in the dir")

			continue
		}

		testhelper.DiffInt(t, tc.IDStr(), "file count", count, tc.expFileCount)

		getFilesTestCleanup(t, tc.IDStr(), prog, tc.post)
	}
}
