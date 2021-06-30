package main

import (
	"os"
	"testing"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/testhelper.mod/testhelper"
)

func TestCd(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		dir string
		testhelper.ExpErr
	}{
		{
			ID:  testhelper.MkID("good-dir"),
			dir: "testdata/gopkg",
		},
		{
			ID:     testhelper.MkID("bad-dir"),
			dir:    "testdata/nonesuch",
			ExpErr: testhelper.MkExpErr(),
		},
	}

	preTestWD, err := os.Getwd()
	if err != nil {
		t.Fatal("cannot get the current directory:", err)
		return
	}
	for _, tc := range testCases {
		err := testCd(tc.dir)
		testhelper.CheckExpErr(t, err, tc)
		postTestWD, err := os.Getwd()
		if err != nil {
			t.Log(tc.IDStr())
			t.Fatal("cannot get the current directory:", err)
			return
		}
		if preTestWD != postTestWD {
			t.Log(tc.IDStr())
			t.Log("\t:  pre-test working dir:", preTestWD)
			t.Log("\t: post-test working dir:", postTestWD)
			t.Errorf("\t: cd failed\n")
		}
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

func TestPkgMatches(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		dir       string
		pkgNames  []string
		expResult bool
	}{
		{
			ID:        testhelper.MkID("noPkg-OK"),
			dir:       "testdata/gopkg",
			pkgNames:  []string{},
			expResult: true,
		},
		{
			ID:       testhelper.MkID("noPkg-NotOK"),
			dir:      "testdata/notgopkg",
			pkgNames: []string{},
		},
		{
			ID:        testhelper.MkID("singlePkg-OK"),
			dir:       "testdata/gopkg",
			pkgNames:  []string{"gopkg"},
			expResult: true,
		},
		{
			ID:       testhelper.MkID("singlePkg-NotOK"),
			dir:      "testdata/gopkg",
			pkgNames: []string{"notgopkg"},
		},
		{
			ID:        testhelper.MkID("multiPkg-OK"),
			dir:       "testdata/gopkg",
			pkgNames:  []string{"notgopkg", "gopkg"},
			expResult: true,
		},
		{
			ID:       testhelper.MkID("multiPkg-NotOK"),
			dir:      "testdata/gopkg",
			pkgNames: []string{"notgopkg", "othernotgopkg"},
		},
	}

	for _, tc := range testCases {
		if checkPkg(t, tc.dir, tc.pkgNames) != tc.expResult {
			t.Log(tc.IDStr())
			t.Errorf("\t: unexpected result\n")
		}
	}
}

// checkPkg will check that the directory is a Go package and that it's name
// matches one of the listed names
func checkPkg(t *testing.T, dir string, pkgNames []string) bool {
	t.Helper()

	undo, err := cd(dir)
	if err != nil {
		t.Fatal("couldn't cd into", dir)
		return false
	}
	defer undo()

	pkg, err := gogen.GetPackage()
	if err != nil { // it's not a package directory
		return false
	}

	fgd := NewFindGoDirs()
	fgd.pkgNames = pkgNames
	return fgd.pkgMatches(pkg)
}
