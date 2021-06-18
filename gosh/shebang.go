package main

import (
	"bytes"
	"os"
)

// shebangFileContents this will read the contents of the file removing any
// initial line starting with '#!'. It returns the edited content and any
// error. If the error is not nil the returned string should not be used.
func shebangFileContents(fileName string) (string, error) {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}

	return string(shebangRemove(content)), nil
}

func shebangRemove(content []byte) []byte {
	if !bytes.HasPrefix(content, []byte("#!")) {
		return content
	}
	if i := bytes.IndexByte(content, '\n'); i > 1 {
		return content[i+1:]
	}
	return content[:0]
}
