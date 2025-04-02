package main

import (
	"bytes"
	"os"
)

const shebangGoshParam = "#gosh.param:"

// shebangFileContents this will read the contents of the file removing any
// initial line starting with '#!'. It returns the edited content and any
// error. If the error is not nil the returned string should not be used.
func shebangFileContents(fileName string) ([]byte, []byte, error) {
	content, err := os.ReadFile(fileName) //nolint:gosec
	if err != nil {
		return []byte{}, []byte{}, err
	}

	script, config := shebangStrip(content)

	return script, config, nil
}

// shebangStrip removes the "#!" from the start of the file and any lines
// starting with # that immediately follow the #! line. If the # is followed
// immediately by "gosh.param:" then the following text up to the end of line
// is copied into the config slice and returned as the second return
// value. The content with any leading lines starting with a # removed is
// returned as the first return value
func shebangStrip(content []byte) ([]byte, []byte) {
	var config []byte

	// strip the leading #! (if any)
	if bytes.HasPrefix(content, []byte("#!")) {
		if i := bytes.IndexByte(content, '\n'); i > 0 {
			content = content[i+1:]
		} else {
			content = content[:0]
		}
	}

	// strip any immediately following lines starting with a #
	//
	// If any of these lines start with #gosh.param: then copy the rest into
	// the config slice for later parsing as parameters to gosh
	for len(content) > 0 && content[0] == '#' {
		end := len(content)
		if i := bytes.IndexByte(content, '\n'); i > 0 {
			end = i + 1
		}

		if bytes.HasPrefix(content, []byte(shebangGoshParam)) {
			config = append(config, content[len(shebangGoshParam):end]...)
		}

		content = content[end:]
	}

	return content, config
}
