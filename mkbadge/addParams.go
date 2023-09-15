package main

import (
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
)

// addParams will add parameters to the passed ParamSet
func addParams(prog *Prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add("twitter-account", psetter.String[string]{
			Value: &prog.twitterAC,
		},
			"The name of an associated X (Twitter) account",
			param.AltNames("twitter-ac", "twitter"),
		)

		ps.Add("no-comment", psetter.Bool{Value: &prog.noComment},
			"suppress the printing of the markdown comments",
		)

		return nil
	}
}
