package main

import (
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestComment(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		addComments bool
		text        string
		expComment  string
	}{
		{
			ID:          testhelper.MkID("with comments"),
			addComments: true,
			text:        "Text",
			expComment:  "\t// gosh : Text",
		},
		{
			ID:          testhelper.MkID("without comments"),
			addComments: false,
			text:        "Text",
			expComment:  "",
		},
	}

	g := &Gosh{}
	for _, tc := range testCases {
		g.addComments = tc.addComments
		comment := g.comment(tc.text)
		testhelper.DiffString(t, tc.IDStr(), "comment", comment, tc.expComment)
	}
}

func TestIndent(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		funcList  []func(*Gosh)
		expIndent string
	}{
		{
			ID:        testhelper.MkID("<nil>"),
			funcList:  []func(*Gosh){},
			expIndent: "",
		},
		{
			ID:        testhelper.MkID("in"),
			funcList:  []func(*Gosh){(*Gosh).in},
			expIndent: "\t",
		},
		{
			ID:        testhelper.MkID("in,out"),
			funcList:  []func(*Gosh){(*Gosh).in, (*Gosh).out},
			expIndent: "",
		},
		{
			ID:        testhelper.MkID("in,in,out"),
			funcList:  []func(*Gosh){(*Gosh).in, (*Gosh).in, (*Gosh).out},
			expIndent: "\t",
		},
	}

	for _, tc := range testCases {
		g := &Gosh{}
		for _, f := range tc.funcList {
			f(g)
		}
		indent := g.indentStr()
		testhelper.DiffString(t, tc.IDStr(), "indent", indent, tc.expIndent)
	}
}
