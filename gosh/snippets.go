package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
)

// snippetPAF generates the Post-Action func that populates the supplied
// script with the contents of the snippet file
func snippetPAF(g *Gosh, sName *string, script *[]string) param.ActionFunc {
	return func(_ location.L, _ *param.ByName, _ []string) error {
		if filepath.IsAbs(*sName) {
			if addSnippet(script, *sName) {
				return nil
			}
			return fmt.Errorf(
				"The snippet file %q doesn't exist or can't be read",
				*sName)
		}

		for _, dir := range g.snippetsDirs {
			fName := filepath.Join(dir, *sName)
			if addSnippet(script, fName) {
				return nil
			}
		}
		return fmt.Errorf(
			"Cannot find the snippet %q:"+
				" in any of the snippet directories: \"%s\"",
			*sName,
			strings.Join(g.snippetsDirs, `", "`))
	}
}

// addSnippet will try to read the file and if it succeeds it will add the
// lines from content, one at a time into the script
func addSnippet(script *[]string, fName string) bool {
	content, err := ioutil.ReadFile(fName)
	if err != nil {
		return false
	}

	addSnippetComment(script, fName)
	addSnippetComment(script, "BEGIN")

	*script = append(*script, string(content[:]))

	addSnippetComment(script, "END")

	return true
}

// addSnippetComment writes the message at the end of a snippet comment
func addSnippetComment(script *[]string, message string) {
	*script = append(*script, "//"+goshCommentIntro+"snippet : "+message)
}
