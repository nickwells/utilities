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

var snippetFileCommentIntro = `\s*//\s*snippet:`

var snippetFileCommentIntroRE = regexp.MustCompile(
	snippetFileCommentIntro)
var snippetFileNoteRE = regexp.MustCompile(
	snippetFileCommentIntro + `\s*Note:\s*`)

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
// lines from content, one at a time into the script. Snippet comment lines
// are not added to the script.
func addSnippet(script *[]string, fName string) bool {
	content, err := ioutil.ReadFile(fName)
	if err != nil {
		return false
	}

	addSnippetComment(script, fName)
	addSnippetComment(script, "BEGIN")

	buf := bytes.NewBuffer(content)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		if snippetFileCommentIntroRE.FindStringIndex(scanner.Text()) != nil {
			continue
		}
		*script = append(*script, scanner.Text())
	}

	addSnippetComment(script, "END")

	return true
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
				fmt.Printf("Couldn't read the snippets directory: %q: %v\n",
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
		noteIntro := "Note: "
		buf := bytes.NewBuffer(content)
		scanner := bufio.NewScanner(buf)
		for scanner.Scan() {
			line := scanner.Text()
			if loc := snippetFileNoteRE.FindStringIndex(line); loc != nil {
				fmt.Println("\t\t" + noteIntro + line[loc[1]:])
				noteIntro = strings.Repeat(" ", len(noteIntro))
			}
		}
	} else if f.IsDir() {
		readSubDir(loc, snippetDir, name)
	} else {
		fmt.Println("\t" + name + " Unexpected file type: " + f.Mode().String())
	}
}
