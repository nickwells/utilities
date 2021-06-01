package main

// gosh.snippet

import (
	"embed"
	"fmt"
	"os"

	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

// Created: Wed May 26 22:30:48 2021

const (
	installAction = "install"
	cmpAction     = "cmp"
)

var (
	fromDir string
	toDir   string
	action  string = cmpAction
)

//go:embed _snippets
var snippetsDir embed.FS

func main() {
	ps := paramset.NewOrDie(addParams,
		param.SetProgramDescription(
			""),
	)

	ps.Parse()

	if fromDir != "" {
		snippetsDir = os.DirFS(fromDir)
	}

	switch action {
	case cmpAction:
		compareSnippets()
	case installAction:
		installSnippets()
	}
}

// compareSnippets compares the snippets in the from directory with those in
// the to directory reporting any differences.
func compareSnippets() {
	fmt.Println("comparing")
}

// installSnippets installs the snippets in the from directory into
// the to directory reporting any differences.
func installSnippets() {
	fmt.Println("installing")
}

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	ps.Add("action",
		psetter.Enum{
			Value: &action,
			AllowedVals: psetter.AllowedVals{
				installAction: "install the default snippets in" +
					" the given directory",
				cmpAction: "compare the default snippets with" +
					" those in the directory",
			},
		},
		"what action should be performed",
		param.AltNames("a"),
		param.Attrs(param.CommandLineOnly),
	)

	ps.Add("install", psetter.Nil{},
		"install the snippets",
		param.PostAction(paction.SetString(&action, installAction)),
		param.Attrs(param.CommandLineOnly),
	)

	ps.Add("to",
		psetter.Pathname{
			Value:       &toDir,
			Expectation: filecheck.DirExists(),
		},
		"set the directory where the snippets are to be copied.",
		param.AltNames("to-dir", "t"),
		param.Attrs(param.CommandLineOnly|param.MustBeSet),
	)

	ps.Add("from",
		psetter.Pathname{
			Value:       &fromDir,
			Expectation: filecheck.DirExists(),
		},
		"set the directory where the snippets are to be found."+
			" If this is not set then the default snippet set will be used",
		param.AltNames("from-dir", "f"),
		param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
	)

	return nil
}
