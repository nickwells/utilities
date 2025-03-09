package main

import (
	"testing"

	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// TestAllParams will repeatedly parse the parameters each in turn to check
// that there are no panics caused by the parsing. The version parameters are
// all excluded as the parsing results in a call to os.Exit.
func TestAllParams(t *testing.T) {
	g := newGosh()
	slp := &snippetListParams{}

	ps := paramset.NewNoHelpNoExitNoErrRptOrPanic(
		paramOptFuncs(g, slp)...)

	skipParams := map[string]bool{}
	skipGroups := map[string]bool{
		versionparams.GroupName: true,
	}
	groups := ps.GetGroups()

	for _, g := range groups {
		if skipGroups[g.Name()] {
			continue
		}

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
