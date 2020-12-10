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

// snippet records the details of the snippet file
type snippet struct {
	name    string
	path    string
	text    []string
	docs    []string
	expects []string
	imports []string
	follows []string
}

// cacheSnippet finds the file for the snippet, parses a snippet object from
// the file, adds any imports and caches the snippet.  It returns an error if
// the snippet cannot be created. If the snippet is already in the cache it
// will skip these steps and return a nil error (the snippet is only cached
// if no errors have been found).
func cacheSnippet(g *Gosh, sName string) error {
	_, ok := g.snippetCache[sName]
	if ok {
		return nil
	}

	var err error
	var s *snippet

	defer func() {
		if err == nil {
			g.imports = append(g.imports, s.imports...)
			g.snippetCache[sName] = s
		}
	}()

	if filepath.IsAbs(sName) {
		s, err = parseSnippet(sName, sName)
		if s == nil {
			return fmt.Errorf("Can't read the snippet file %q: %w",
				sName, err)
		}

		return err
	}

	for _, dir := range g.snippetsDirs {
		fName := filepath.Join(dir, sName)
		s, err = parseSnippet(fName, sName)
		if err == nil {
			return nil
		}
	}

	err = fmt.Errorf("snippet %q: is not in any snippet directory: \"%s\"",
		sName,
		strings.Join(g.snippetsDirs, `", "`))

	return err
}

// snippetExpand finds the snippet and returns the contents. It returns and
// error if the snippet is invalid.
func snippetExpand(g *Gosh, sName string) ([]string, error) {
	s, ok := g.snippetCache[sName]
	if !ok {
		return nil,
			fmt.Errorf("snippet: %q not found in the snippet cache", sName)
	}

	g.snippetUsed[sName] = true
	for _, shouldBeUsed := range s.follows {
		if !g.snippetUsed[shouldBeUsed] {
			g.addError("Snippet out of order",
				fmt.Errorf("snippet %q should appear before snippet %q",
					shouldBeUsed, sName))
		}
	}
	return s.text, nil
}

// parseSnippet will try to read the file and if it succeeds it will
// construct the snippet from content
func parseSnippet(fName, sName string) (*snippet, error) {
	content, err := ioutil.ReadFile(fName)
	if err != nil {
		return nil, err
	}

	s := &snippet{
		name: sName,
		path: fName,
	}

	addSnippetComment(&s.text, fName)
	addSnippetComment(&s.text, "BEGIN")

	buf := bytes.NewBuffer(content)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		l := scanner.Text()
		if snippetCommentRE.FindStringIndex(l) != nil {
			if addMatchToSlices(l, snippetImportRE, &s.imports) {
				continue
			}
			if addMatchToSlices(l, snippetExpectRE, &s.expects) {
				continue
			}
			if addMatchToSlices(l, snippetAfterRE, &s.expects, &s.follows) {
				continue
			}
			if addWholeMatchToSlice(l, snippetNoteRE, &s.docs) {
				continue
			}
		} else {
			s.text = append(s.text, l)
		}
	}

	addSnippetComment(&s.text, "END")

	return s, nil
}

// addMatchToSlices tests the string for a match against the regexp. If it
// matches then the remainder of the string after the matched portion is
// trimmed of white space. If the resulting string is non-empty it is added
// to the slices. It returns true if the string matched the regex and false
// otherwise.
func addMatchToSlices(s string, re *regexp.Regexp, slcs ...*[]string) bool {
	if loc := re.FindStringIndex(s); loc != nil {
		text := strings.TrimSpace(s[loc[1]:])
		if len(text) > 0 {
			for _, slc := range slcs {
				*slc = append(*slc, text)
			}
		}
		return true
	}
	return false
}

// addWholeMatchToSlice behaves as per addMatchToSlices but doesn't trim
// the line or ignore empty lines
func addWholeMatchToSlice(s string, re *regexp.Regexp, slc *[]string) bool {
	if loc := re.FindStringIndex(s); loc != nil {
		text := s[loc[1]:]
		*slc = append(*slc, text)
		return true
	}
	return false
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
		fmt.Printf("\tCouldn't read the sub-directory: %q:\n\t%v\n\n",
			subDir, err)
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
