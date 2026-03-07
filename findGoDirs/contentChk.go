package main

import (
	"regexp"
	"strings"

	"github.com/nickwells/location.mod/location"
)

// ContentCheck contains the parameters used to check the contents of a
// file.
type ContentCheck struct {
	// an optional tag to identify the check that has matched the content
	name string
	// A pattern that must be matched for success.
	matchPattern *regexp.Regexp
	// A pattern that is applied to matching lines and will cause them to be
	// skipped if the skip pattern matches.
	skipPattern *regexp.Regexp
	// A pattern which when matched will stop the operation of this
	// ContentCheck.
	stopPattern *regexp.Regexp
	// The ContentCheck will only be applied to files matching this pattern.
	filenamePattern *regexp.Regexp
	// The ContentCheck will only be applied to files not matching this pattern.
	filenameSkipPattern *regexp.Regexp
}

// String returns a string describing the ContentCheck
func (cc ContentCheck) String() string {
	var (
		rval       string
		valPrefix  = "    "
		namePrefix = ""
	)
	if cc.name != "" {
		rval = cc.name + ": "
		namePrefix = strings.Repeat(" ", len(rval))
		valPrefix += namePrefix
	}

	rval += "\n" +
		namePrefix + "A file has a line matching the pattern:\n"
	rval += valPrefix + cc.matchPattern.String()

	if cc.skipPattern != nil {
		rval += "\n" +
			namePrefix + "Skip if line also matches the pattern:\n" +
			valPrefix + cc.skipPattern.String()
	}

	if cc.stopPattern != nil {
		rval += "\n" +
			namePrefix + "Stop looking after a line matching the pattern:\n" +
			valPrefix + cc.stopPattern.String()
	}

	if cc.filenamePattern != nil {
		rval += "\n" +
			namePrefix + "Only search files matching the pattern:\n" +
			valPrefix + cc.filenamePattern.String()
	}

	if cc.filenameSkipPattern != nil {
		rval += "\n" +
			namePrefix + "Don't search files matching the pattern:\n" +
			valPrefix + cc.filenameSkipPattern.String()
	}

	return rval
}

// FileNameOK returns true if the filename matches the filenamePattern (if any)
func (cc ContentCheck) FileNameOK(fn string) bool {
	if cc.filenamePattern != nil {
		if !cc.filenamePattern.MatchString(fn) {
			return false
		}
	}

	if cc.filenameSkipPattern != nil {
		if cc.filenameSkipPattern.MatchString(fn) {
			return false
		}
	}

	return true
}

// Stop returns true if the line matches the stopPattern (if any)
func (cc ContentCheck) Stop(s string) bool {
	if cc.stopPattern == nil {
		return false
	}

	return cc.stopPattern.MatchString(s)
}

// Match returns true if the line matches the matchPattern and not the
// skipPattern (if any)
func (cc ContentCheck) Match(s string) bool {
	if cc.matchPattern == nil {
		return false
	}

	if !cc.matchPattern.MatchString(s) {
		return false
	}

	if cc.skipPattern == nil {
		return true
	}

	return !cc.skipPattern.MatchString(s)
}

// StatusCheck contains a check and an associated status. It is used to
// record whether the stopPattern has been matched
type StatusCheck struct {
	chk     *ContentCheck
	stopped bool
}

// CheckLine applies the checks to the supplied string, setting the stopped
// flag appropriately and excluding skipped values
func (sc *StatusCheck) CheckLine(s string) bool {
	if sc.stopped {
		return false
	}

	if sc.chk.Stop(s) {
		sc.stopped = true
		return false
	}

	return sc.chk.Match(s)
}

var (
	buildTagChecks = ContentCheck{
		name:            "build-tag",
		matchPattern:    regexp.MustCompile(`^//(\s*\+build\b|go:build\b)`),
		stopPattern:     regexp.MustCompile(`^\s*package\s+`),
		filenamePattern: regexp.MustCompile(`.*\.go`),
	}

	gogenChecks = ContentCheck{
		name:            "go-generate",
		matchPattern:    regexp.MustCompile(`^//go:generate\s+`),
		filenamePattern: regexp.MustCompile(`.*\.go`),
	}
)

type (
	contentMap map[string][]location.L
)
