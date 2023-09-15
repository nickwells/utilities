package main

import (
	"testing"

	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestAllParams(t *testing.T) {
	g := newGosh()
	slp := &snippetListParams{}

	ps := paramset.NewNoHelpNoExitNoErrRptOrPanic(
		paramOptFuncs(g, slp)...)

	skipParams := map[string]bool{
		"version":            true,
		"version-part":       true,
		"version-p":          true,
		"version-part-short": true,
		"version-short":      true,
		"version-s":          true,
	}
	groups := ps.GetGroups()
	for _, g := range groups {
		for _, p := range g.Params() {
			paramNames := p.AltNames()
			s := p.Setter()
			vr := s.ValueReq()

			for _, pName := range paramNames {
				if skipParams[pName] {
					continue
				}
				args := []string{"-" + pName}
				if vr == param.Mandatory {
					args = append(args, "")
				}
				panicked, panicVal := testhelper.PanicSafe(func() {
					localG := newGosh()
					localSLP := &snippetListParams{}
					localPS := paramset.NewNoHelpNoExitNoErrRptOrPanic(
						paramOptFuncs(localG, localSLP)...)
					localPS.Parse(args)
				})

				if panicked {
					t.Log("Panic: ", panicVal)
					t.Errorf("failed to parse param: %q", pName)
				}
			}
		}
	}
}
