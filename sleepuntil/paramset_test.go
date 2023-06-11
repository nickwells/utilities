package main

import (
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestMakeParamSet(t *testing.T) {
	prog := NewProg()
	panicked, panicVal := testhelper.PanicSafe(func() {
		_ = makeParamSet(prog)
	})
	testhelper.PanicCheckError(t, "makeParamSet",
		panicked, false,
		panicVal, []string{})
}
