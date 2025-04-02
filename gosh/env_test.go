package main

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestPopulateEnv(t *testing.T) {
	const (
		goshDir  = "somewhere"
		execName = "G"
	)

	underscore := "_=" + filepath.Join(goshDir, execName)
	aPre := "a=1"
	aPost := "a=A"
	bPre := "b=1"
	cPre := "c=1"
	newD := "d=D"

	testCases := []struct {
		testhelper.ID
		env    []string
		g      *gosh
		expEnv []string
	}{
		{
			ID: testhelper.MkID("a,b,c ; change a, clear=true"),
			env: []string{
				aPre,
				bPre,
				cPre,
			},
			g: &gosh{
				env:      []string{aPost},
				clearEnv: true,
				goshDir:  "somewhere",
				execName: "G",
			},
			expEnv: []string{
				underscore,
				aPost,
			},
		},
		{
			ID: testhelper.MkID("a,b,c ; change a, clear=false"),
			env: []string{
				aPre,
				bPre,
				cPre,
			},
			g: &gosh{
				env:      []string{aPost},
				goshDir:  "somewhere",
				execName: "G",
			},
			expEnv: []string{
				underscore,
				aPost,
				bPre,
				cPre,
			},
		},
		{
			ID: testhelper.MkID("a,b,c ; no change, clear=true"),
			env: []string{
				aPre,
				bPre,
				cPre,
			},
			g: &gosh{
				clearEnv: true,
				goshDir:  "somewhere",
				execName: "G",
			},
			expEnv: []string{
				underscore,
			},
		},
		{
			ID: testhelper.MkID("a,b,c ; no change, clear=false"),
			env: []string{
				aPre,
				bPre,
				cPre,
			},
			g: &gosh{
				goshDir:  "somewhere",
				execName: "G",
			},
			expEnv: []string{
				underscore,
				aPre,
				bPre,
				cPre,
			},
		},
		{
			ID: testhelper.MkID("a,b,c ; change d, clear=true"),
			env: []string{
				aPre,
				bPre,
				cPre,
			},
			g: &gosh{
				env:      []string{newD},
				clearEnv: true,
				goshDir:  "somewhere",
				execName: "G",
			},
			expEnv: []string{
				underscore,
				newD,
			},
		},
		{
			ID: testhelper.MkID("a,b,c ; change d, clear=false"),
			env: []string{
				aPre,
				bPre,
				cPre,
			},
			g: &gosh{
				env:      []string{newD},
				goshDir:  "somewhere",
				execName: "G",
			},
			expEnv: []string{
				underscore,
				aPre,
				bPre,
				cPre,
				newD,
			},
		},
	}

	for _, tc := range testCases {
		var env []string

		if tc.g.clearEnv {
			env = tc.g.populateEnv([]string{})
		} else {
			env = tc.g.populateEnv(tc.env)
		}

		sort.Strings(env)
		sort.Strings(tc.expEnv)

		if testhelper.DiffSlice(t, tc.IDStr(), "", env, tc.expEnv) {
			t.Log(tc.IDStr())
			t.Errorf("\t: failed\n")
		}
	}
}
