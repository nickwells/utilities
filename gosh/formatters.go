package main

import (
	"os/exec"
	"strings"

	"github.com/nickwells/english.mod/english"
	"github.com/nickwells/verbose.mod/verbose"
)

// Formatter holds the name of a program that can format the generated
// program according to the Go standard and any associated arguments.
type Formatter struct {
	name string
	args []string
}

// formatters is a list of default commands that can be used to 'format' the
// code. Note that we are only interested in the side effect, namely that of
// populating the import statement. These two independed functions should be
// split into two separate tasks.
var formatters = []Formatter{
	{name: "gofmt", args: []string{"-w"}},
	{name: "gopls", args: []string{"format", "-w"}},
	{name: "gofumpt", args: []string{"-w"}},
	{name: "goimports", args: []string{"-w"}},
}

// formatterCmds will return a string describing the available commands
func formatterCmds() string {
	fCmds := make([]string, 0, len(formatters))

	for _, f := range formatters {
		cmd := "'" + f.name + " " + strings.Join(f.args, " ") + "'"
		fCmds = append(fCmds, cmd)
	}

	return english.Join(fCmds, ", ", " or ")
}

// findFormatter will find the first available formatter in the PATH. it
// returns the formatter, the full path and true if the formatter is found,
// empty values and false otherwise.
func findFormatter(g *gosh) (Formatter, string, bool) {
	defer g.dbgStack.Start("findFormatter", "Finding the formatting command")()
	intro := g.dbgStack.Tag()

	for _, f := range formatters {
		if path, err := exec.LookPath(f.name); err == nil {
			verbose.Println(intro, " Using the default formatter: ", f.name)
			verbose.Println(intro, "                    pathname: ", path)
			verbose.Println(intro, "                   arguments: ",
				strings.Join(f.args, " "))

			return f, path, true
		}
	}

	return Formatter{}, "", false
}
