package main

import (
	"strings"

	"github.com/nickwells/english.mod/english"
)

// Formatter holds the name of a program that can "format" the generated
// program and any associated arguments. Note that the notion of formatting
// is more to do with generating the necessary import statements.
type Formatter struct {
	name string
	args []string
}

var formatters = []Formatter{
	{name: "gopls", args: []string{"imports", "-w"}},
	{name: "goimports", args: []string{"-w"}},
	{name: "gofmt", args: []string{"-w"}},
}

func formatterCmds() string {
	fCmds := make([]string, 0, len(formatters))

	for _, f := range formatters {
		cmd := "'" + f.name + " " + strings.Join(f.args, " ") + "'"
		fCmds = append(fCmds, cmd)
	}

	return english.Join(fCmds, ", ", " or ")
}
