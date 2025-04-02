package main

import (
	"os/exec"
	"strings"

	"github.com/nickwells/english.mod/english"
)

// Importer holds the details of a program that can generate import
// statements for a Go file.
type Importer struct {
	name       string
	args       []string
	installCmd string
}

// importers is a list of default commands that can be used to populate the
// import statement. They are attempted in order with the first one found use
// to populate the import statement.
var importers = []Importer{
	{
		name:       "gopls",
		args:       []string{"imports", "-w"},
		installCmd: "go install golang.org/x/tools/gopls@latest",
	},
	{
		name:       "goimports",
		args:       []string{"-w"},
		installCmd: "go install golang.org/x/tools/cmd/goimports@latest",
	},
}

// importerCmds will return a string describing the available commands
func importerCmds() string {
	iCmds := make([]string, 0, len(importers))

	for _, f := range importers {
		cmd := "'" + f.name + " " + strings.Join(f.args, " ") + "'"
		iCmds = append(iCmds, cmd)
	}

	return english.Join(iCmds, ", ", " or ")
}

// importerPrograms will return a string giving the name of the importers
// that will be used if they are available and no importer has been set
// explicitly
func importerPrograms() string {
	iProgs := make([]string, 0, len(importers))

	for _, f := range importers {
		iProgs = append(iProgs, "'"+f.name+"'")
	}

	return english.Join(iProgs, ", ", " or ")
}

// findImporter will find the first available importer in the PATH. it
// returns the importer, the full path and true if the importer is found,
// empty values and false otherwise.
func findImporter(g *gosh) (Importer, string, bool) {
	defer g.dbgStack.Start("findImporter",
		"Finding the import generating command")()

	for _, f := range importers {
		if path, err := exec.LookPath(f.name); err == nil {
			return f, path, true
		}
	}

	return Importer{}, "", false
}
