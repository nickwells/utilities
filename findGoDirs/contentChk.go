package main

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/nickwells/location.mod/location"
)

// ContentCheck contains the parameters used to check the contents of a
// file.
type ContentCheck struct {
	// A tag used to update the ContentCheck.
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
}

// String returns a string describing the ContentCheck
func (cc ContentCheck) String() string {
	rval := cc.name + ":\n"
	partPrefix := strings.Repeat(" ", len(cc.name)+1)
	valPrefix := partPrefix + "    "

	rval += partPrefix + "A file has a line matching pattern:\n"
	rval += valPrefix + cc.matchPattern.String()

	if cc.skipPattern != nil {
		rval += "\n" +
			partPrefix + "Skip if line also matches pattern:\n" +
			valPrefix + cc.skipPattern.String()
	}

	if cc.stopPattern != nil {
		rval += "\n" +
			partPrefix + "Stop looking after a line matching pattern:\n" +
			valPrefix + cc.stopPattern.String()
	}

	if cc.filenamePattern != nil {
		rval += "\n" +
			partPrefix + "Only search files matching pattern:\n" +
			valPrefix + cc.filenamePattern.String()
	}

	return rval
}

// FileNameOK returns true if the filename matches the filenamePattern (if any)
func (cc ContentCheck) FileNameOK(fn string) bool {
	if cc.filenamePattern == nil {
		return true
	}

	return cc.filenamePattern.MatchString(fn)
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

// setMatchPattern sets the matchPattern in the supplied ContentCheck value. It
// will return an error if the ContentCheck pointer is nil, if the
// matchPattern is already set or if the string doesn't compile into a valid
// regular expression.
func setMatchPattern(chk *ContentCheck, s string) error {
	errPfx := "Could not set the match pattern"
	if chk == nil {
		return errors.New(errPfx + " - the ContentCheck value is nil")
	}

	errPfx += " for " + chk.name

	if chk.matchPattern != nil {
		return errors.New(errPfx + " - the match pattern is already set")
	}

	re, err := regexp.Compile(s)
	if err != nil {
		return fmt.Errorf("%s - bad pattern: %w", errPfx, err)
	}

	chk.matchPattern = re

	return nil
}

// setSkipPattern sets the skipPattern in the supplied ContentCheck value. It
// will return an error if the ContentCheck pointer is nil, if the
// skipPattern is already set or if the string doesn't compile into a valid
// regular expression.
func setSkipPattern(chk *ContentCheck, s string) error {
	errPfx := "Could not set the skip pattern"
	if chk == nil {
		return errors.New(errPfx + " - the ContentCheck value is nil")
	}

	errPfx += " for " + chk.name

	if chk.skipPattern != nil {
		return errors.New(errPfx + " - the skip pattern is already set")
	}

	re, err := regexp.Compile(s)
	if err != nil {
		return fmt.Errorf("%s - bad pattern: %w", errPfx, err)
	}

	chk.skipPattern = re

	return nil
}

// setStopPattern sets the stopPattern in the supplied ContentCheck value. It
// will return an error if the ContentCheck pointer is nil, if the
// stopPattern is already set or if the string doesn't compile into a valid
// regular expression.
func setStopPattern(chk *ContentCheck, s string) error {
	errPfx := "Could not set the stop pattern"
	if chk == nil {
		return errors.New(errPfx + " - the ContentCheck value is nil")
	}

	errPfx += " for " + chk.name

	if chk.stopPattern != nil {
		return errors.New(errPfx + " - the stop pattern is already set")
	}

	re, err := regexp.Compile(s)
	if err != nil {
		return fmt.Errorf("%s - bad pattern: %w", errPfx, err)
	}

	chk.stopPattern = re

	return nil
}

// setFilenamePattern sets the filenamePattern in the supplied ContentCheck
// value. It will return an error if the ContentCheck pointer is nil, if the
// filenamePattern is already set or if the string doesn't compile into a
// valid regular expression.
func setFilenamePattern(chk *ContentCheck, s string) error {
	errPfx := "Could not set the filename pattern"
	if chk == nil {
		return errors.New(errPfx + " - the ContentCheck value is nil")
	}

	errPfx += " for " + chk.name

	if chk.filenamePattern != nil {
		return errors.New(errPfx + " - the filename pattern is already set")
	}

	re, err := regexp.Compile(s)
	if err != nil {
		return fmt.Errorf("%s - bad pattern: %w", errPfx, err)
	}

	chk.filenamePattern = re

	return nil
}

// setterFunc is the type of a func for setting the value of some part of a
// ContentCheck object
type setterFunc func(*ContentCheck, string) error

// checkerPart records the settable details of a ContentCheck along with some
// descriptive text and a function to set just that part.
type checkerPart struct {
	desc   string
	setter setterFunc
}

const dfltCheckerPart = "match"

var checkerParts = map[string]checkerPart{
	dfltCheckerPart: {
		desc:   "match file content",
		setter: setMatchPattern,
	},
	"stop": {
		desc: "stop further checking." +
			" Once a line is found matching this pattern" +
			" no more lines in the file will be checked" +
			" by this checker.",
		setter: setStopPattern,
	},
	"skip": {
		desc: "skip otherwise matching lines." +
			" If a line matches the match pattern" +
			" but also matches this skip pattern then" +
			" it is taken as not being a matching line.",
		setter: setSkipPattern,
	},
	"filename": {
		desc: "limit the files to check. This check will" +
			" only be applied to files with names matching this pattern.",
		setter: setFilenamePattern,
	},
}

// checkerPartNames returns a sorted list of the named parts of a checker
// excluding the default part.
func checkerPartNames() []string {
	var partNames []string

	for k := range checkerParts {
		if k != dfltCheckerPart {
			partNames = append(partNames, k)
		}
	}

	sort.Strings(partNames)

	return partNames
}

// checkerPartsHelpText returns a formatted string describing the named parts
// of a checker. Note that the default, 'match', is excluded.
func checkerPartsHelpText() string {
	maxNameLen := 0

	partNames := checkerPartNames()
	for _, k := range partNames {
		if len(k) > maxNameLen {
			maxNameLen = len(k)
		}
	}

	rval := ""
	sep := "  "

	for _, k := range partNames {
		rval += fmt.Sprintf("%s%-*s: %s",
			sep, maxNameLen, k, checkerParts[k].desc)
		sep = "\n  "
	}

	return rval
}

// StatusCheck contains a check and an associated status. it is used to
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
	buildTagChecks = &ContentCheck{
		name:            "build-tag",
		matchPattern:    regexp.MustCompile(`^//\s*\+build\s+`),
		stopPattern:     regexp.MustCompile(`^\s*package\s+`),
		filenamePattern: regexp.MustCompile(`.*\.go`),
	}

	gogenChecks = &ContentCheck{
		name:            "go-gen",
		matchPattern:    regexp.MustCompile(`^//go:generate\s+`),
		filenamePattern: regexp.MustCompile(`.*\.go`),
	}
)

type (
	contentMap map[string][]location.L
	checkMap   map[string]*ContentCheck
)
