package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/nickwells/param.mod/v6/psetter"
)

// ContChkSetter is used as a parameter setter for adding new content checks
type ContChkSetter struct {
	psetter.ValueReqMandatory

	Value *checkMap
}

// SetWithVal checks that the parameter value meets is of the form
// name=value. It then checks that the name consists of just letters, digits,
// dashes and underscores and at most a single dot. The following value must
// compile into a valid regular expression. The name (before any dot) is used
// to find an entry in the Value checkMap and if there is no dot in the name
// the entry must not exist. If there is a dot the named entry must
// exist. The part of the name following a dot must be one of the recognised
// sub tags and are used to determine which part of the ContentCheck entry is
// to be set.
func (chk ContChkSetter) SetWithVal(_, paramVal string) error {
	tag, re, hasRE := strings.Cut(paramVal, "=")
	if !hasRE {
		return fmt.Errorf(
			"missing '=': the parameter %q should be of the form: tag=RE",
			paramVal)
	}

	tagName, partName, hasPartName := strings.Cut(tag, ".")
	if !hasPartName {
		partName = dfltCheckerPart
	}

	cc, ok := (*chk.Value)[tagName]
	if !ok {
		if partName != dfltCheckerPart {
			var checkers []string

			for k := range *chk.Value {
				checkers = append(checkers, k)
			}

			if len(checkers) == 0 {
				return errors.New("no checkers have been created yet")
			}

			sort.Strings(checkers)

			return fmt.Errorf("no such checker: %q. Available checkers: %s",
				tagName, strings.Join(checkers, ", "))
		}

		cc = &ContentCheck{name: tagName}
		(*chk.Value)[tagName] = cc
	} else if partName == dfltCheckerPart {
		return fmt.Errorf("the checker %q already exists", tagName)
	}

	cp, ok := checkerParts[partName]
	if !ok {
		return fmt.Errorf(
			"unknown checker part name: %q. Must be one of %s",
			partName, strings.Join(checkerPartNames(), ", "))
	}

	err := cp.setter(cc, re)
	if err != nil {
		return fmt.Errorf("%s: %w", tagName, err)
	}

	return nil
}

// AllowedValues simply returns a description of a well-formed value
func (chk ContChkSetter) AllowedValues() string {
	return "a string of the form tag=RE or tag.part=RE." +
		"\n\n" +
		"The tag can be any value and is simply used to" +
		" add parts to an existing content checker." +
		" Note that the first use of a tag must be without" +
		" any part name attached." +
		"\n\n" +
		"The 'RE' part must compile to a valid" +
		" regular expression." +
		"\n\n" +
		"The 'part' must be one of:" +
		"\n" +
		checkerPartsHelpText()
}

// CurrentValue returns the current setting of the parameter value
func (chk ContChkSetter) CurrentValue() string {
	rval := ""

	var checks []string

	for k := range *chk.Value {
		checks = append(checks, k)
	}

	sort.Strings(checks)

	sep := ""

	for _, k := range checks {
		rval += sep + (*chk.Value)[k].String()
		sep = "\n"
	}

	return rval
}

// CheckSetter panics if the setter has not been properly created - if the
// Value is nil.
func (chk ContChkSetter) CheckSetter(name string) {
	if chk.Value == nil {
		panic(psetter.NilValueMessage(name, "ContChkSetter"))
	}
}

// ValDescribe returns a short string showing what the value should look like.
func (chk ContChkSetter) ValDescribe() string {
	return "tag=RE"
}
