package main

import (
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestMakeParamSet(t *testing.T) {
	g := newGosh()
	slp := &snippetListParams{}
	panicked, panicVal := testhelper.PanicSafe(func() {
		_ = makeParamSet(g, slp)
	})
	testhelper.PanicCheckError(t, "makeParamSet",
		panicked, false,
		panicVal, []string{})
}
