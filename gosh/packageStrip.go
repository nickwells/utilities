package main

import (
	"go/parser"
	"go/token"
	"os"
)

// packageFileContents this will read the contents of the file removing any
// initial lines that are blank, comments, package statements or import
// statements. This leaves the content suitable to be inserted into a gosh
// file and simplifies pulling in files from other packages.
//
// It returns the edited content and any error. If the error is
// not nil the returned string should not be used.
func packageFileContents(fileName string) (string, error) {
	content, err := os.ReadFile(fileName) //nolint:gosec
	if err != nil {
		return "", err
	}

	fset := token.NewFileSet()

	f, err := parser.ParseFile(
		fset, fileName, string(content), parser.ImportsOnly)
	if err != nil {
		return "", err
	}

	end := f.Name.End()

	if impLen := len(f.Imports); impLen > 0 {
		end = f.Imports[impLen-1].End()
		if int(end) < len(content) && content[end] == ')' {
			end++
		}
	}

	return string(content[end:]), nil
}
