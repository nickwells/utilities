package stdparams

import (
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
	"github.com/nickwells/utilities/internal/callstack"
)

// AddTiming adds the common show-timing parameter used to set the
// ShowTimings field in a callstack.Stack struct
func AddTiming(
	ps *param.PSet,
	cs *callstack.Stack,
	opt ...param.OptFunc,
) *param.ByName {
	opt = append(opt,
		param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
		param.AltNames("show-timing", "show-time", "show-times"))

	return ps.Add("show-timings", psetter.Bool{Value: &cs.ShowTimings},
		"report the time taken for various parts of this program to complete.",
		opt...)
}
