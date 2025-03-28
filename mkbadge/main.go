package main

// mkbadge

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nickwells/gogen.mod/gogen"
)

// Created: Fri Sep 25 18:29:06 2020

// Prog holds program parameters and status
type Prog struct {
	// parameters
	twitterAC string

	noComment bool
}

// NewProg returns an intialised Prog
func NewProg() *Prog {
	return &Prog{}
}

func main() {
	prog := NewProg()
	ps := makeParamSet(prog)

	ps.Parse()

	const githubPfx = "github.com/"

	module := gogen.GetModuleOrDie()

	trimSemver := regexp.MustCompile(`/v([2-9]|[1-9][0-9]+)$`)
	repo := trimSemver.ReplaceAllLiteralString(module, "")

	prog.comment("START")

	fmt.Println("[" +
		"![go.dev reference]" +
		"(https://img.shields.io/badge/go.dev-reference-green?logo=go)" +
		"]" +
		"(https://pkg.go.dev/mod/" + module + ")")

	fmt.Println("[" +
		"![Go Report Card]" +
		"(https://goreportcard.com/badge/" + module + ")]" +
		"(https://goreportcard.com/report/" + module + ")")

	if strings.HasPrefix(repo, githubPfx) {
		fmt.Println("![GitHub License]" +
			"(https://img.shields.io/github/license/" +
			strings.TrimPrefix(repo, githubPfx) +
			")")
	}

	if prog.twitterAC != "" {
		fmt.Println("![Twitter Follow]" +
			"(https://img.shields.io/twitter/follow/" +
			prog.twitterAC +
			"?style=social)")
	}

	prog.comment("END")
}

// comment prints a comment line with the passed string at the end (if it
// isn't empty)
func (prog *Prog) comment(s string) {
	if prog.noComment {
		return
	}

	fmt.Println("<!-- Code generated by mkbadge; DO NOT EDIT. " + s + " -->")
}
