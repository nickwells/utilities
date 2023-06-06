package main

import (
	"fmt"
	"strings"

	"github.com/nickwells/english.mod/english"
)

// Counts records the number of files and actions etc
type Counts struct {
	name         string
	isComparable bool
	total        int

	compared int
	cmpErrs  int

	deleted int
	delErrs int

	reverted int
	revErrs  int
}

// Status holds counts of various operations on and problems with the files
type Status struct {
	cmpFile Counts
	dupFile Counts
	badFile Counts
}

// InitStatus returns a properly initialised Status
func InitStatus() Status {
	return Status{
		cmpFile: Counts{name: "comparable", isComparable: true},
		dupFile: Counts{name: "duplicate"},
		badFile: Counts{name: "problem"},
	}
}

// reportVal reports the value if it is greater than zero
func reportVal(n int, name string, indent int) {
	if n <= 0 {
		return
	}
	fmt.Printf("%s%3d %s\n", strings.Repeat(" ", indent), n, name)
}

// reportFile reports the status of the named file
func (c Counts) reportFile() {
	if c.total == 0 {
		return
	}

	reportVal(c.total, c.name+" "+english.Plural("file", c.total), 4)

	if c.isComparable {
		reportVal(c.compared, "compared", 8)
		reportVal(c.cmpErrs,
			"comparison "+english.Plural("error", c.cmpErrs), 8)
		reportVal(c.total-c.cmpErrs-c.compared, "skipped", 8)
		fmt.Println()
	}

	reportVal(c.deleted, "deleted", 8)
	reportVal(c.delErrs,
		"deletion "+english.Plural("error", c.delErrs), 8)

	reportVal(c.reverted, "reverted", 8)
	reportVal(c.revErrs,
		"revert "+english.Plural("error", c.revErrs), 8)

	reportVal(c.total-c.deleted-c.reverted, "kept", 8)
}

// Report will print out the Status structure
func (s Status) Report() {
	fmt.Println("Summary")
	allFileCount := s.cmpFile.total +
		s.dupFile.total +
		s.badFile.total

	if allFileCount == 0 {
		fmt.Println("No files found")
		return
	}
	s.badFile.reportFile()
	s.dupFile.reportFile()
	s.cmpFile.reportFile()
}
