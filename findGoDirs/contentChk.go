package main

import (
	"regexp"

	"github.com/nickwells/location.mod/location"
)

// ContentCheck contains the parameters used to check the contents of a
// file. These are: a descriptive name, a pattern that must be matched for
// success and a pattern which when matched will stop the operation of this
// ContentCheck.
type ContentCheck struct {
	name         string
	matchPattern *regexp.Regexp
	stopPattern  *regexp.Regexp
}

// StatusCheck contains a check and an associated status. it is used to
// record whether the stopPattern has been matched
type StatusCheck struct {
	chk     *ContentCheck
	stopped bool
}

var buildTagChecks = &ContentCheck{
	name:         "build tags",
	matchPattern: regexp.MustCompile(`^//\s*\+build\s+`),
	stopPattern:  regexp.MustCompile(`^\s*package\s+`),
}

var gogenChecks = &ContentCheck{
	name:         "go generate comment",
	matchPattern: regexp.MustCompile(`^//go:generate\s+`),
}

type (
	contentMap map[string][]location.L
	checkMap   map[string]*ContentCheck
)
