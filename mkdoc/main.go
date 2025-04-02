package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
)

// Created: Wed Jun 10 11:29:28 2020

const (
	docSuffix = ".DOC.md"

	snippetFile = "_snippet.md"
)

// prog holds program parameters and status
type prog struct {
	parts []partParams

	// parameters
	snippetModSkip []string
	snippetModPfx  []string

	buildArgs []string
}

// newProg returns an initialised prog structure
func newProg() *prog {
	var (
		mainPart = partParams{
			partName: "intro",
			headFile: "_headDoc.md",
			tailFile: "_tailDoc.md",
			suffix:   docSuffix,
		}
		examplesPart = partParams{
			partName: "examples",
			headFile: "_headExamples.md",
			tailFile: "_tailExamples.md",
			suffix:   ".EXAMPLES.md",
			subTitle: "Examples",
			desc:     "examples",
		}
		refsPart = partParams{
			partName: "refs",
			headFile: "_headReferences.md",
			tailFile: "_tailReferences.md",
			suffix:   ".REFERENCES.md",
			subTitle: "See Also",
			desc:     "external references",
		}
		notesPart = partParams{
			partName: "notes",
			headFile: "_headNotes.md",
			tailFile: "_tailNotes.md",
			suffix:   ".NOTES.md",
			subTitle: "Notes",
			desc:     "additional notes",
		}
	)

	return &prog{
		parts: []partParams{
			mainPart,
			examplesPart,
			refsPart,
			notesPart,
		},
		snippetModPfx: []string{
			"github.com/nickwells/",
		},
	}
}

// partParams holds the details for generating the different Markdown files.
type partParams struct {
	headFile   string
	partName   string
	extraFiles []string
	tailFile   string
	suffix     string
	subTitle   string
	desc       string
}

const (
	paramSnippetModPfx  = "snippet-mod-prefix"
	paramSnippetModSkip = "snippet-mod-skip"
)

func main() {
	prog := newProg()
	ps := makeParamSet(prog)

	ps.Parse()

	packageIsMainOrExit()

	cmd := prog.buildCmd(commandName())
	defer os.RemoveAll(filepath.Dir(cmd)) //nolint:errcheck
	prog.parts[0].extraFiles = prog.getModuleSnippets(cmd)

	var docText string
	for _, pp := range prog.parts {
		docText += pp.generate(cmd)
	}

	if docText == "" {
		fmt.Println("No Documentation!")
		return
	}

	filename := prog.parts[0].filename(cmd)
	if makeFile(filename, docText) {
		fmt.Println("Add the following lines to the README.md file" +
			" (if not already present).")
		fmt.Printf("## %s\n\n", filepath.Base(cmd))
		fmt.Printf("[See here](%s/%s)\n", filepath.Base(cmd), filename)
	}
}

// packageIsMainOrExit checks that the package directory we are in is one for
// generating a command
func packageIsMainOrExit() {
	if pkgName := gogen.GetPackageOrDie(); pkgName != "main" {
		fmt.Fprintf(os.Stderr,
			"the package (%q) does not build a command\n", pkgName)
		os.Exit(1)
	}
}

// buildCmd builds the temporary executable instance of the program and
// returns the full pathname. The file should be removed after the last
// use of the program.
func (prog *prog) buildCmd(cmdName string) string {
	dirName, err := os.MkdirTemp("", "mkdoc_"+cmdName+"_*")
	if err != nil {
		fmt.Fprintln(os.Stderr,
			"cannot create the temporary directory for the build:", err)
		os.Exit(1)
	}

	cmd := filepath.Join(dirName, cmdName)
	buildCmd := []string{"build", "-o", cmd}
	buildCmd = append(buildCmd, prog.buildArgs...)
	gogen.ExecGoCmd(gogen.NoCmdIO, buildCmd...)

	return cmd
}

// commandName returns the name of the command to be documented - it is
// derived from the directory name rather than the go.mod file
func commandName() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr,
			"cannot retrieve the current directory name:", err)
		os.Exit(1)
	}

	return filepath.Base(cwd)
}

// skipModule returns true if the module should be skipped, false
// otherwise. A module should be skipped if either it does not have a prefix
// in the list of valid module prefixes or else it is explicitly excluded.
func (prog *prog) skipModule(modName string) bool {
	skip := true

	for _, pfx := range prog.snippetModPfx {
		if strings.HasPrefix(modName, pfx) {
			skip = false
			break
		}
	}

	if skip {
		return true
	}

	if slices.Contains(prog.snippetModSkip, modName) {
		return true
	}

	return false
}

// getModuleSnippets finds all the dependent modules of the command and if
// any of them have a '_snippet.md' file in the module directory then the
// pathname is added to the list of snippet files to return
func (prog *prog) getModuleSnippets(cmd string) []string {
	gopath := gogen.GetGopath()
	if gopath == "" {
		return []string{}
	}

	modVerCmd := []string{"version", "-m", cmd}
	buf := new(bytes.Buffer)

	gogen.ExecGoCmdCaptureOutput(buf, modVerCmd...)

	snippetFiles := []string{}

	s := bufio.NewScanner(buf)
	for s.Scan() {
		parts := strings.Fields(s.Text())
		if len(parts) != 4 && parts[0] != "dep" {
			continue
		}

		modName := parts[1]
		vsn := parts[2]

		if prog.skipModule(modName) {
			continue
		}

		filename := filepath.Join(gopath,
			"pkg",
			"mod",
			modName+"@"+vsn,
			snippetFile)

		if _, err := os.Stat(filename); err == nil {
			snippetFiles = append(snippetFiles, filename)
		}
	}

	return snippetFiles
}

// filename returns the appropriate filename for the given part of the
// command documentation.
func (pp partParams) filename(cmd string) string {
	return "_" + filepath.Base(cmd) + pp.suffix
}

// generate constructs the text of part of the documentation. It operates as
// follows:
//
// It starts with the contents of the head file (if any).
//
// Then it runs the command to get the text of the named part of the help
// text (if any).
//
// If there are any extras given then their contents will be added to the
// text.
//
// Lastly, the contents of the tail file (if any) are added.
//
// Having generated the text, if it is not empty and there is a subTitle then
// it will generate the file, write the generated text into it and return a
// fragment of Markdown referencing this subsidiary file. Otherwise it
// returns the text generated.
func (pp partParams) generate(cmd string) string {
	var text string
	text += getText(pp.headFile)
	text += getDocPart(cmd, pp.partName)

	for _, extraFile := range pp.extraFiles {
		if extraText := getText(extraFile); extraText != "" {
			text += "\n\n" + extraText
		}
	}

	text += getText(pp.tailFile)
	if text == "" {
		_ = os.Remove(pp.filename(cmd))
		return ""
	}

	if pp.subTitle == "" {
		return text
	}

	filename := pp.filename(cmd)
	if makeFile(filename, text) {
		return "\n\n" +
			"## " + pp.subTitle +
			"\n" +
			"For " + pp.desc + " [see here](" + filename + ")\n"
	}

	return ""
}

// makeFile creates the file and populates it. It returns true if the file
// was successfully created and populated, false otherwise
func makeFile(filename, contents string) bool {
	f, err := os.Create(filename) //nolint:gosec
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not create the file:", err)
		return false
	}

	defer f.Close()

	_, err = f.WriteString("<!-- Created by mkdoc DO NOT EDIT. -->\n\n")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not write to the file:", err)
		return false
	}

	_, err = f.WriteString(contents)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not write to the file:", err)
		return false
	}

	return true
}

// getText reads the text from the file. If err is not nil and isn't
// os.ErrNotExist then the error will be reported and the program will
// exit. Otherwise the (possibly empty) string read from the file will be
// returned.
func getText(filename string) string {
	if filename == "" {
		return ""
	}

	text, err := os.ReadFile(filename) //nolint:gosec
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "there was a problem reading the file:", err)
		os.Exit(1)
	}

	return string(text)
}

// getDocPart this will run the command passing it the standard help
// parameters. It will capture the output and return it. If the command fails
// then the program exits.
//
// Note that errors are not shown and the program will not exit with a
// non-zero exit status if there are any errors. This is because if any
// arguments are required then an error will be generated and go generate
// will abort the command.
func getDocPart(cmdPath, part string) string {
	args := []string{
		"-help-format", "markdown",
		"-help-show", part,
		"-params-dont-show-errors",
		"-params-dont-exit-on-errors",
	}

	cmd := exec.Command(cmdPath, args...) //nolint:gosec
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr

	err := cmd.Run()
	if err != nil {
		cmdLine := []string{cmdPath}
		cmdLine = append(cmdLine, args...)
		cmdLineStr := strings.Join(cmdLine, " ")

		fmt.Fprintln(os.Stderr, "Couldn't exec the command")
		fmt.Fprintln(os.Stderr, "\t"+cmdLineStr)
		fmt.Fprintln(os.Stderr, "\tError Out:", stdErr.String())
		fmt.Fprintln(os.Stderr, "\tError:", err)
		os.Exit(1)
	}

	return stdOut.String()
}

// addParams will add parameters to the passed ParamSet
func addParams(prog *prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add("build-args",
			psetter.StrListAppender[string]{Value: &prog.buildArgs},
			"arguments to be passed to go build when building the program",
			param.AltNames("build-arg", "build-param"),
		)

		ps.Add(paramSnippetModPfx,
			psetter.StrListAppender[string]{Value: &prog.snippetModPfx},
			"add the prefix of Go module names to be searched for"+
				" Markdown snippet files ("+snippetFile+")",
			param.AltNames("sm-pfx"),
		)

		ps.Add(paramSnippetModSkip,
			psetter.StrListAppender[string]{Value: &prog.snippetModSkip},
			"add the name of Go modules to be skipped when searching for"+
				" Markdown snippet files ("+snippetFile+")",
			param.AltNames("sm-skip"),
		)

		return nil
	}
}

// addNotes will add any Notes to the passed Param Set
func addNotes(prog *prog) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		ps.AddNote("Files generated",
			"Each of the generated Markdown files will have a"+
				" name starting with an underscore followed by"+
				" the name of the program itself. The files to"+
				" be generated are as follows:"+
				"\n\n"+makePartsNote(prog.parts))

		ps.AddNote("Markdown snippets",
			"This program will discover any modules that the program"+
				" being documented uses. Having found these packages"+
				" it will find any whose name starts with one of the"+
				" standard prefixes"+
				" (by default: '"+strings.Join(prog.snippetModPfx, "', '")+"')"+
				" and if the package's module directory contains a"+
				" file called '"+snippetFile+"' then the contents of"+
				" that file will be added to the end of the main"+
				" documentary Markdown file (ending '"+docSuffix+"')"+
				"\n\n"+
				"Note that you can add to the standard prefixes by"+
				" passing the '"+paramSnippetModPfx+"' parameter."+
				" Similarly, you can exclude specific modules by"+
				" passing the '"+paramSnippetModSkip+"' parameter.")

		return nil
	}
}

// makePartsNote generates the text describing the extra text files available
func makePartsNote(parts []partParams) string {
	var text string

	sep := ""

	for _, pp := range parts {
		text += sep
		sep = "\n\n"
		text += fmt.Sprintf("The text from the %q section of", pp.partName)
		text += fmt.Sprintf(" the help message is written to a file ending %q.",
			pp.suffix)
		text += fmt.Sprintf(" Text to come before this is in a file called %q",
			pp.headFile)
		text += fmt.Sprintf(" and any text to come after in a file called %q.",
			pp.tailFile)
	}

	return text
}
