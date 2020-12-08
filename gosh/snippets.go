package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
)

const (
	snippetCommentStr   = "snippet:"
	snippetNoteStr      = "Note:"
	snippetImportStr    = "Import:"
	snippetExpectStr    = "Expect:"
	snippetAfterStr     = "ComesAfter:"
	snippetCommentREStr = `^\s*//\s*` + snippetCommentStr
	snippetNoteREStr    = snippetCommentREStr + `\s*` + snippetNoteStr + `\s*`
	snippetImportREStr  = snippetCommentREStr + `\s*` + snippetImportStr + `\s*`
	snippetExpectREStr  = snippetCommentREStr + `\s*` + snippetExpectStr + `\s*`
	snippetAfterREStr   = snippetCommentREStr + `\s*` + snippetAfterStr + `\s*`
)

var snippetCommentRE = regexp.MustCompile(snippetCommentREStr)
var snippetNoteRE = regexp.MustCompile(snippetNoteREStr)
var snippetImportRE = regexp.MustCompile(snippetImportREStr)
var snippetExpectRE = regexp.MustCompile(snippetExpectREStr)
var snippetAfterRE = regexp.MustCompile(snippetAfterREStr)

// snippetPAF generates the Post-Action func that populates the supplied
// script with the contents of the snippet file
func snippetPAF(g *Gosh, sName *string, script *[]string) param.ActionFunc {
	return func(_ location.L, _ *param.ByName, _ []string) error {
		if filepath.IsAbs(*sName) {
			fileFound, err := addSnippet(g, script, *sName, *sName)
			if !fileFound {
				return fmt.Errorf("Can't read the snippet file %q: %w",
					*sName, err)
			}
			return err
		}

		for _, dir := range g.snippetsDirs {
			fName := filepath.Join(dir, *sName)
			fileFound, err := addSnippet(g, script, fName, *sName)
			if fileFound {
				return err // Will be nil unless ComesAfter rule is broken
			}
		}
		return fmt.Errorf(
			"Cannot find the snippet %q:"+
				" in any of the snippet directories: \"%s\"",
			*sName,
			strings.Join(g.snippetsDirs, `", "`))
	}
}

// missingSnippets will check that all the expected snippets have been used
// and will report any that are missing. It returns the count of expected
// snippets that were not used.
func missingSnippets(g *Gosh) int {
	snippetsMissing := 0
	for snippet, expectedBy := range g.snippetsExpectedBy {
		if !g.snippetsUsed[snippet] {
			fmt.Fprintf(os.Stderr,
				"Missing snippet: %q\n", snippet)
			fmt.Fprintf(os.Stderr,
				"\tthis snippet is expected\n\tif snippet %q is used\n",
				expectedBy)
			snippetsMissing++
		}
	}
	return snippetsMissing
}

// addSnippet will try to read the file and if it succeeds it will add the
// lines from content, one at a time into the script. Snippet comment lines
// are not added to the script.
func addSnippet(g *Gosh, script *[]string, fName, sName string) (bool, error) {
	content, err := ioutil.ReadFile(fName)
	if err != nil {
		return false, err
	}

	g.snippetsUsed[sName] = true
	addSnippetComment(script, fName)
	addSnippetComment(script, "BEGIN")

	buf := bytes.NewBuffer(content)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()
		if snippetCommentRE.FindStringIndex(line) != nil {
			if loc := snippetImportRE.FindStringIndex(line); loc != nil {
				importStr := line[loc[1]:]
				if len(importStr) > 0 {
					g.imports = append(g.imports, importStr)
				}
			} else if loc := snippetExpectRE.FindStringIndex(line); loc != nil {
				expectSName := strings.TrimSpace(line[loc[1]:])
				if len(expectSName) > 0 {
					g.snippetsExpectedBy[expectSName] = sName
				}
			} else if loc := snippetAfterRE.FindStringIndex(line); loc != nil {
				mustFollow := strings.TrimSpace(line[loc[1]:])
				if len(mustFollow) > 0 {
					if !g.snippetsUsed[mustFollow] {
						return true, fmt.Errorf(
							"If snippet %q is used"+
								" it must appear after snippet %q",
							sName, mustFollow)
					}
				}
			}
			continue
		}
		*script = append(*script, line)
	}

	addSnippetComment(script, "END")

	return true, nil
}

// addSnippetComment writes the message at the end of a snippet comment
func addSnippetComment(script *[]string, message string) {
	*script = append(*script, "//"+goshCommentIntro+"snippet : "+message)
}

// listSnippets will read all of the snippet directories and show the
// available snippet files
func (g *Gosh) listSnippets() {
	var loc map[string]string = make(map[string]string)

	for _, dir := range g.snippetsDirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Printf(
					"Couldn't read the snippets directory: %q:\n\t%v\n\n",
					dir, err)
			}
			continue
		}
		fmt.Println("in: " + dir)
		for _, f := range files {
			showFile(loc, dir, "", f)
		}
	}
}

// readSubDir descends into the sub directory and reads the content
func readSubDir(loc map[string]string, snippetDir, subDir string) {
	name := filepath.Join(snippetDir, subDir)
	files, err := ioutil.ReadDir(name)
	if err != nil {
		fmt.Printf("\tCouldn't read the sub-directory: %q: %v", subDir, err)
		return
	}
	for _, f := range files {
		showFile(loc, snippetDir, subDir, f)
	}
}

// showFile reports the file if it is a regular file, descends into the sub
// directory if it is a directory and reports it as a problem otherwise
func showFile(loc map[string]string, snippetDir, subDir string, f os.FileInfo) {
	name := f.Name()
	if subDir != "" {
		name = filepath.Join(subDir, name)
	}
	if f.Mode().IsRegular() {
		fmt.Println("\t" + name)

		if otherSD, eclipsed := loc[name]; eclipsed {
			fmt.Printf("\t\teclipsed by the entry in %q\n", otherSD)
		} else {
			loc[name] = snippetDir
		}
		content, err := ioutil.ReadFile(filepath.Join(snippetDir, name))
		if err != nil {
			fmt.Println("\t\t*** Cannot be read ***")
			return
		}
		printText(content, "       Note: ", snippetNoteRE)
		printText(content, "    Imports: ", snippetImportRE)
		printText(content, "     Expect: ", snippetExpectRE, snippetAfterRE)
		printText(content, "Must Follow: ", snippetAfterRE)
	} else if f.IsDir() {
		readSubDir(loc, snippetDir, name)
	} else {
		fmt.Println("\t" + name + " Unexpected file type: " + f.Mode().String())
	}
}

// printText will find those lines in the content that match the re and will
// strip out the re text and print the rest with the first line prefixed by
// the intro
func printText(content []byte, intro string, res ...*regexp.Regexp) {
	blanks := strings.Repeat(" ", len(intro))
	buf := bytes.NewBuffer(content)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()
		for _, re := range res {
			if loc := re.FindStringIndex(line); loc != nil {
				fmt.Println("\t\t" + intro + line[loc[1]:])
				intro = blanks
			}
		}
	}
}
