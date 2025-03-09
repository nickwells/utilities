package main

import (
	"errors"
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestPackageStrip(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		filename   string
		expContent string
		expErr     error
	}{
		{
			ID:       testhelper.MkID("with package name, no imports"),
			filename: "testdata/packageFiles/noImports.go",
			expContent: `
func f() string { return "Hello" }
`,
		},
		{
			ID:       testhelper.MkID("with package name, 1 import"),
			filename: "testdata/packageFiles/withImport.go",
			expContent: `
func f() { fmt.Println("Hello") }
`,
		},
		{
			ID:       testhelper.MkID("with package name, 2 import"),
			filename: "testdata/packageFiles/withMultiImport.go",
			expContent: `

func f() { fmt.Println(strings.ToLower("Hello")) }
`,
		},
		{
			ID:       testhelper.MkID("with package name, 2 imports, no code"),
			filename: "testdata/packageFiles/withMultiImportAndNoCode.go",
			expContent: `
`,
		},
		{
			ID:       testhelper.MkID("with comments before package name"),
			filename: "testdata/packageFiles/withCommentsBefore.go",
			expContent: `
func f() string { return "Hello" }
`,
		},
		{
			ID: testhelper.MkID(
				"with comments between package name and import"),
			filename: "testdata/packageFiles/withCommentsBetween.go",
			expContent: `
func f() { fmt.Println("Hello") }
`,
		},
		{
			ID:       testhelper.MkID("bad file - no package name"),
			filename: "testdata/packageFiles/noPackage.go",
			expErr: errors.New(
				"testdata/packageFiles/noPackage.go:1:1: expected 'package', found 'func'"),
		},
	}

	for _, tc := range testCases {
		content, err := packageFileContents(tc.filename)

		if testhelper.DiffErr(
			t, tc.IDStr(), "error", err, tc.expErr) {
			continue
		}

		testhelper.DiffString(
			t, tc.IDStr(), "package file content", content, tc.expContent)
	}
}
