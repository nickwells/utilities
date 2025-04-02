package main

import (
	"go/parser"
	"go/token"
	"os"
)

// packageRename this will read the contents of the file replacing the
// package name with 'main' if it is not already called 'main'.
//
// It returns the edited content and any error. If the error is
// not nil the returned string should not be used.
func packageRename(fileName string) ([]byte, error) {
	content, err := os.ReadFile(fileName) //nolint:gosec
	if err != nil {
		return []byte{}, err
	}

	fset := token.NewFileSet()

	f, err := parser.ParseFile(
		fset, fileName, string(content), parser.PackageClauseOnly)
	if err != nil {
		return []byte{}, err
	}

	if f.Name.Name == "main" {
		return content, nil
	}

	start := fset.Position(f.Package).Offset
	end := f.Name.End()

	pkgMain := []byte("package main\n")
	rval := content[:start]
	rval = append(rval, pkgMain...)
	rval = append(rval, content[end:]...)

	return rval, nil
}
