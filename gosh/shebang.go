package main

import (
	"os"
	"regexp"
)

// shebangFileContents this will read the contents of the file removing any
// initial line starting with '#!'. It returns the edited content and any
// error. If the error is not nil the returned string should not be used.
func shebangFileContents(fileName string) (string, error) {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile("^#![^\n]*(\n|$)")
	content = re.ReplaceAll(content, []byte{})

	return string(content), nil
}
