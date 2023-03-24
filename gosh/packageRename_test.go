package main

import (
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestPackageRename(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		filename   string
		expContent string
		testhelper.ExpErr
	}{
		{
			ID:       testhelper.MkID("package name EQ main"),
			filename: "testdata/packageRename/_pkgEQmain.go",
			expContent: `package main

var a int
`,
		},
		{
			ID:       testhelper.MkID("package name NE main"),
			filename: "testdata/packageRename/_pkgNEmain.go",
			expContent: `package main

var a int
`,
		},
		{
			ID:       testhelper.MkID("leading comments, package name EQ main"),
			filename: "testdata/packageRename/_pkgEQmainWithComments.go",
			expContent: `// comment

package main

var a int
`,
		},
		{
			ID:       testhelper.MkID("leading comments, package name NE main"),
			filename: "testdata/packageRename/_pkgNEmainWithComments.go",
			expContent: `// comment

package main

var a int
`,
		},
		{
			ID:       testhelper.MkID("file has no package"),
			filename: "testdata/packageRename/_hasNoPackage.go",
			ExpErr:   testhelper.MkExpErr("expected 'package', found"),
		},
		{
			ID:       testhelper.MkID("file doesn't exist"),
			filename: "testdata/packageRename/nosuchfile.go",
			ExpErr:   testhelper.MkExpErr("no such file or directory"),
		},
	}

	for _, tc := range testCases {
		content, err := packageRename(tc.filename)
		if testhelper.CheckExpErr(t, err, tc) && err == nil {
			testhelper.DiffString(t, tc.IDStr(), "converted content",
				string(content), tc.expContent)
		}
	}
}
