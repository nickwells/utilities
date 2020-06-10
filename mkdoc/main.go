// mkdoc
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
)

// Created: Wed Jun 10 11:29:28 2020

const (
	prefix = "_"

	examplesSuffix = ".EXAMPLES.md"
	docSuffix      = ".DOC.md"
	refsSuffix     = ".REFERENCES.md"

	examplesTailFile = "_tailExamples.md"
	docTailFile      = "_tailDoc.md"
	refsTailFile     = "_tailReferences.md"
)

func main() {
	ps := paramset.NewOrDie(addParams,
		param.SetProgramDescription(
			"This creates markdown documentation for any Go program which"+
				" uses the param package"+
				" (github.com/nickwells/param.mod/*/param). It will"+
				" generate a markdown file containing examples if the"+
				" program has examples and it will generare a file"+
				" containing references if the program has references. It"+
				" will generate a main doc file which will have links to"+
				" the examples and references files if they exist. This"+
				" main doc file should then be linked to from the"+
				" README.md file"),
	)

	ps.Parse()

	if gogen.GetPackageOrDie() != "main" {
		fmt.Fprintln(os.Stderr, "the package does not build a command")
		os.Exit(1)
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr,
			"cannot retrieve the current directory name:", err)
		os.Exit(1)
	}
	cmdName := filepath.Base(cwd)
	cmd := filepath.Join(cwd, cmdName)

	gogen.ExecGoCmd(gogen.NoCmdIO, "build")

	docFileName := prefix + cmdName + docSuffix
	examplesFileName := prefix + cmdName + examplesSuffix
	refsFileName := prefix + cmdName + refsSuffix

	docText := getDocPart(cmd, "intro")
	docExtras := getText(docTailFile)

	refsText := getDocPart(cmd, "refs") + getText(refsTailFile)

	examplesText := getDocPart(cmd, "examples") + getText(examplesTailFile)
	if examplesText != "" {
		docText += "For examples [see here](" + examplesFileName + ")\n"
		makeFile(examplesFileName, examplesText)
	}
	if refsText != "" {
		docText += "For external references [see here](" + refsFileName + ")\n"
		makeFile(refsFileName, refsText)
	}
	docText += docExtras
	makeFile(docFileName, docText)
}

// makeFile creates the file and populates it
func makeFile(filename, contents string) {
	f, err := os.Create(filename)
	defer f.Close()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not create the file:", err)
		os.Exit(1)
	}
	_, err = f.WriteString("<!-- Created by mkdoc DO NOT EDIT. -->\n\n")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not write to the file:", err)
		os.Exit(1)
	}
	_, err = f.WriteString(contents)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not write to the file:", err)
		os.Exit(1)
	}

}

// getText reads the text from the file. If err is not nil and isn't
// os.ErrNotExist then the error will be reported and the program will
// exit. Otherwise the (possibly empty) string read from the file will be
// returned.
func getText(filename string) string {
	extraText, err := ioutil.ReadFile(filename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "there was a problem reading the file:", err)
		os.Exit(1)
	}
	return string(extraText)
}

// getDocPart this will run the command passing it the standard help
// parameters. It will capture the output and return it. If the command fails
// then the program exits.
func getDocPart(cmdPath, part string) string {
	args := []string{
		"-help-format", "markdown",
		"-help-show", part,
	}

	cmd := exec.Command(cmdPath, args...)
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
func addParams(ps *param.PSet) error {
	return nil
}
