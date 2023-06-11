package main

import (
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

// addParams will add parameters to the passed ParamSet
func addParams(prog *Prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add("twitter-account", psetter.String{Value: &prog.twitterAC},
			"The name of an associated twitter account",
			param.AltNames("twitter-ac", "twitter"),
		)

		ps.Add("no-comment", psetter.Bool{Value: &prog.noComment},
			"suppress the printing of the markdown comments",
		)

		return nil
	}
}
