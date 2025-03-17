package main

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v6/psetter"
)

const moduleMapSeparator = "=>"

// ModuleMapSetter is a specialised setter for setting entries in a map from
// module name to directory. The resulting entries will be used to set
// replace entries in a go.mod file.
type ModuleMapSetter struct {
	psetter.ValueReqMandatory
	Value *map[string]string
}

// SetWithVal (called when a value follows the parameter) splits the value
// using the Separator. It then checks that the second part of the value is a
// valid directory and returns an error if not.
func (s ModuleMapSetter) SetWithVal(_ string, paramVal string) error {
	mod, dir, ok := strings.Cut(paramVal, moduleMapSeparator)
	if !ok {
		return fmt.Errorf(
			"bad value: %q, should be in two parts with %q in between",
			paramVal, moduleMapSeparator)
	}

	if dir == "" {
		return errors.New("the replacement directory is empty")
	}

	var err error

	dir, err = filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("bad module replacement directory %q: %w", dir, err)
	}

	if err := filecheck.DirExists().StatusCheck(dir); err != nil {
		return fmt.Errorf("bad module replacement directory: %w", err)
	}

	(*s.Value)[mod] = dir

	return nil
}

// AllowedValues returns a string listing the allowed values
func (s ModuleMapSetter) AllowedValues() string {
	return "a module name and a replacement directory separated by " +
		moduleMapSeparator
}

// ValDescribe returns a brief description of the value
func (s ModuleMapSetter) ValDescribe() string {
	return "mod" + moduleMapSeparator + "path"
}

// CurrentValue returns the current setting of the parameter value
func (s ModuleMapSetter) CurrentValue() string {
	cv := ""
	keys := slices.Sorted(maps.Keys(*s.Value))
	sep := ""

	for _, k := range keys {
		cv += sep + fmt.Sprintf("%s%s%v", k, moduleMapSeparator, (*s.Value)[k])
		sep = "\n"
	}

	return cv
}

// CheckSetter panics if the setter has not been properly created - if the
// Value is nil.
func (s ModuleMapSetter) CheckSetter(name string) {
	if s.Value == nil {
		panic(psetter.NilValueMessage(name, "gosh.ModuleMapSetter"))
	}

	if *s.Value == nil {
		*s.Value = make(map[string]string)
	}
}
