package main

import (
	"os/exec"
	"strings"

	"github.com/nickwells/english.mod/english"
	"github.com/nickwells/verbose.mod/verbose"
)

// Importer holds the name of a program that can generate import statements
// for a Go file.
type Importer struct {
	name string
	args []string
}

// importers is a list of default commands that can be used to
// populate the import statement.
var importers = []Importer{
	{name: "gopls", args: []string{"imports", "-w"}},
	{name: "goimports", args: []string{"-w"}},
}

// importerCmds will return a string describing the available commands
func importerCmds() string {
	fCmds := make([]string, 0, len(importers))

	for _, f := range importers {
		cmd := "'" + f.name + " " + strings.Join(f.args, " ") + "'"
		fCmds = append(fCmds, cmd)
	}

	return english.Join(fCmds, ", ", " or ")
}

// findImporter will find the first available importer in the PATH. it
// returns the importer, the full path and true if the importer is found,
// empty values and false otherwise.
func findImporter(g *Gosh) (Importer, string, bool) {
	defer g.dbgStack.Start("findImporter",
		"Finding the import generating command")()
	intro := g.dbgStack.Tag()

	for _, f := range importers {
		if path, err := exec.LookPath(f.name); err == nil {
			verbose.Println(intro, " Using the default importer: ", f.name)
			verbose.Println(intro, "                   pathname: ", path)
			verbose.Println(intro, "                  arguments: ",
				strings.Join(f.args, " "))
			return f, path, true
		}
	}
	return Importer{}, "", false
}
