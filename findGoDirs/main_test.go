package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestCd(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		dir string
		testhelper.ExpErr
		expStdout string
		expStderr string
	}{
		{
			ID:  testhelper.MkID("good-dir"),
			dir: "testdata/gopkg",
		},
		{
			ID:  testhelper.MkID("bad-dir"),
			dir: "testdata/nonesuch",
			ExpErr: testhelper.MkExpErr(
				"chdir testdata/nonesuch: no such file or directory"),
			expStderr: `Cannot chdir to "testdata/nonesuch":` +
				` chdir testdata/nonesuch: no such file or directory` + "\n",
		},
	}

	preTestWD, err := os.Getwd()
	if err != nil {
		t.Fatal("cannot get the current directory (before testing):", err)
		return
	}

	for _, tc := range testCases {
		fakeIO, err := testhelper.NewStdioFromString("")
		if err != nil {
			t.Fatal("Cannot make the fakeIO: ", err)
		}

		err = testCd(tc.dir)
		testhelper.CheckExpErr(t, err, tc)

		postTestWD, err := os.Getwd()
		if err != nil {
			t.Log(tc.IDStr())
			t.Fatal("cannot get the current directory (after testing):", err)
		}

		if preTestWD != postTestWD {
			t.Log(tc.IDStr())
			t.Log("\t:  pre-test working dir:", preTestWD)
			t.Log("\t: post-test working dir:", postTestWD)
			t.Errorf("\t: cd failed\n")
		}

		stdout, stderr, err := fakeIO.Done()
		if err != nil {
			t.Log(tc.IDStr())
			t.Fatal("cannot get the std IO buffers:", err)
		}

		testhelper.DiffString(t, tc.IDStr(), "stdout",
			string(stdout), tc.expStdout)
		testhelper.DiffString(t, tc.IDStr(), "stderr",
			string(stderr), tc.expStderr)
	}
}

// testCd will change directory to the given dir and change back
// afterwards. After this has been called the current directory should be as
// it was before this was called.
func testCd(dir string) error {
	undo, err := cd(dir)
	if err != nil {
		return err
	}

	defer undo()

	return nil
}

func TestHasFiles(t *testing.T) {
	const testdir = "testdata/gopkg"

	undo, err := cd(testdir)
	if err != nil {
		t.Fatal("couldn't cd into", testdir)
		return
	}

	defer undo()

	testCases := []struct {
		testhelper.ID
		filesWanted []string
		expResult   bool
	}{
		{
			ID:          testhelper.MkID("singleFile-present"),
			filesWanted: []string{"doesExist1.go"},
			expResult:   true,
		},
		{
			ID:          testhelper.MkID("multiFile-all-present"),
			filesWanted: []string{"doesExist1.go", "doesExist2.go"},
			expResult:   true,
		},
		{
			ID:          testhelper.MkID("singleFile-not-present"),
			filesWanted: []string{"doesNotExist1.go"},
		},
		{
			ID:          testhelper.MkID("multiFile-none-present"),
			filesWanted: []string{"doesNotExist1.go", "doesNotExist2.go"},
		},
		{
			ID:          testhelper.MkID("multiFile-some-present"),
			filesWanted: []string{"doesExist1.go", "doesNotExist2.go"},
		},
	}

	for _, tc := range testCases {
		if hasEntries(tc.filesWanted) != tc.expResult {
			t.Log(tc.IDStr())
			t.Errorf("\t: unexpected result\n")
		}
	}
}

// copyDirFromTo copies the contents of "from" into "to". It will call itself
// recursively for subdirectories and return the first error encountered.
// Both "from" and "to" directories should exist before it is called.
func copyDirFromTo(from, to string) error {
	d, err := os.Open(from) //nolint:gosec
	if err != nil {
		return err
	}
	defer d.Close()

	toBeCopied, err := d.Readdir(0)
	if err != nil {
		return err
	}

	for _, fi := range toBeCopied {
		if fi.IsDir() {
			var (
				newFromDir = filepath.Join(from, fi.Name())
				newToDir   = filepath.Join(to, fi.Name())
			)

			err = os.Mkdir(newToDir, fi.Mode()&fs.ModePerm)
			if err != nil {
				return err
			}

			err = copyDirFromTo(newFromDir, newToDir)
			if err != nil {
				return err
			}
		} else if fi.Mode().IsRegular() {
			var (
				fromFile = filepath.Join(from, fi.Name())
				toFile   = filepath.Join(to, fi.Name())
			)

			fromBytes, err := os.ReadFile(fromFile) //nolint:gosec
			if err != nil {
				return err
			}

			err = os.WriteFile(toFile, fromBytes, fi.Mode()&fs.ModePerm)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf(
				"only dirs & regular files can be copied, %q is neither",
				fi.Name())
		}
	}

	return nil
}

// copyDir makes a temporary directory and copies the contents of the passed
// directory into the temporary directory. It returns the name of the
// temporary directory, a cleanup func and any errors encountered.
func copyDir(fromDir string) (string, func() error, error) {
	tmpDir, err := os.MkdirTemp("", "testdir.")
	if err != nil {
		return "", nil, err
	}

	err = copyDirFromTo(fromDir, tmpDir)

	return tmpDir, func() error { return os.RemoveAll(tmpDir) }, err
}

func TestPkgMatches(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		dir      string
		pkgNames []string
	}{
		{
			ID:       testhelper.MkID("noPkg-OK"),
			dir:      "testdata/gopkg",
			pkgNames: []string{},
		},
		{
			ID: testhelper.MkID("noPkg-NotOK"),
			ExpErr: testhelper.MkExpErr(
				"gogen.GetPackage error: exit status 1"),
			dir:      "testdata/notgopkg",
			pkgNames: []string{},
		},
		{
			ID:       testhelper.MkID("singlePkg-OK"),
			dir:      "testdata/gopkg",
			pkgNames: []string{"gopkg"},
		},
		{
			ID:       testhelper.MkID("singlePkg-NotOK"),
			ExpErr:   testhelper.MkExpErr("no packages match"),
			dir:      "testdata/gopkg",
			pkgNames: []string{"notgopkg"},
		},
		{
			ID:       testhelper.MkID("multiPkg-OK"),
			dir:      "testdata/gopkg",
			pkgNames: []string{"notgopkg", "gopkg"},
		},
		{
			ID:       testhelper.MkID("multiPkg-NotOK"),
			ExpErr:   testhelper.MkExpErr("no packages match"),
			dir:      "testdata/gopkg",
			pkgNames: []string{"notgopkg", "othernotgopkg"},
		},
	}

	for _, tc := range testCases {
		dir, cleanup, err := copyDir(tc.dir)
		if err != nil {
			t.Log("copyDir failed")
			t.Fatal(err)
		}

		err = checkPkg(dir, tc.pkgNames)
		testhelper.CheckExpErr(t, err, tc)

		if cleanup != nil {
			if err := cleanup(); err != nil {
				t.Log("cleanup failed")
				t.Fatal(err)
			}
		}
	}
}

// checkPkg will check that the directory is a Go package and that it's name
// matches one of the listed names
func checkPkg(dir string, pkgNames []string) error {
	undo, err := cd(dir)
	if err != nil {
		return fmt.Errorf("cd error: %w", err)
	}
	defer undo()

	pkg, err := gogen.GetPackage()
	if err != nil { // it's not a package directory
		return fmt.Errorf("gogen.GetPackage error: %w", err)
	}

	fgd := newProg()
	fgd.pkgNames = pkgNames

	if !fgd.pkgMatches(pkg) {
		return errors.New("no packages match")
	}

	return nil
}
