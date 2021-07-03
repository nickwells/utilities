package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/nickwells/param.mod/v5/param/psetter"
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
	parts := strings.SplitN(paramVal, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf(
			"Missing '=': the parameter %q should be of the form: tag=RE",
			paramVal)
	}
	tagParts := strings.SplitN(parts[0], ".", 2)
	tagName := tagParts[0]
	partName := dfltCheckerPart
	if len(tagParts) > 1 {
		partName = tagParts[1]
	}
	cc, ok := (*chk.Value)[tagName]
	if !ok {
		if partName != dfltCheckerPart {
			var checkers []string
			for k := range *chk.Value {
				checkers = append(checkers, k)
			}
			if len(checkers) == 0 {
				return errors.New("No checkers have been created yet")
			}
			sort.Strings(checkers)
			return fmt.Errorf("No such checker: %q. Available checkers: %s",
				tagName, strings.Join(checkers, ", "))
		}
		cc = &ContentCheck{name: tagName}
	} else if partName == dfltCheckerPart {
		return fmt.Errorf("The checker %q already exists", tagName)
	}
	cp, ok := checkerParts[partName]
	if !ok {
		return fmt.Errorf(
			"Unknown checker part name: %q. Must be one of %s",
			partName, strings.Join(checkerPartNames(), ", "))
	}
	err := cp.setter(cc, parts[1])
	if err != nil {
		return fmt.Errorf("%s: %w", tagName, err)
	}
	return nil
}

// AllowedValues simply returns a description of a well-formed value
func (chk ContChkSetter) AllowedValues() string {
	return "a string of the form tag=RE or tag.part=RE." +
		" Note that 'RE' must compile to a regular expression" +
		"\n" +
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
