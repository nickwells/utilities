package main

import (
	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
	"github.com/nickwells/unitsetter.mod/v4/unitsetter"
)

// addParams will add parameters to the passed ParamSet
func addParams(prog *prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add("units",
			unitsetter.UnitSetter{
				Value: &prog.displayUnits,
				F:     prog.dataFamily,
			},
			"set the units in which to display the results")

		ps.Add("no-label",
			psetter.Bool{
				Value: &noLabel,
			},
			"show the results without labels")

		ps.Add("table",
			psetter.Bool{
				Value: &showAsTable,
			},
			"show the results in a table rather than on a line")

		ps.Add("show",
			psetter.EnumList[string]{
				Value:       &fields,
				AllowedVals: prog.allowedFields,
				Checks: []check.StringSlice{
					check.SliceHasNoDups[[]string, string],
					check.SliceLength[[]string](check.ValGT(0)),
				},
			},
			"choose which information to show about the file system")

		err := ps.SetRemHandler(param.NullRemHandler{}) // allow trailing params
		if err != nil {
			return err
		}

		return nil
	}
}
