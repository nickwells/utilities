package main

import (
	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
)

// addParams will add parameters to the passed ParamSet
func addParams(prog *prog) param.PSetOptFunc {
	checkStringNotEmpty := check.StringLength[string](check.ValGT(0))

	return func(ps *param.PSet) error {
		ps.Add("group",
			psetter.String[string]{
				Value:  &prog.groupName,
				Checks: []check.String{param.GroupNameCheck},
			},
			"sets the name of the group of parameters for which we are"+
				" building the functions. If this is not given then only"+
				" common config file functions will be generated. If a"+
				" group name is given then only the group-specific config"+
				" file functions will be generated. Additionally, unless"+
				" the output file name has been changed from the default,"+
				" the output file name will be adjusted to reflect the"+
				" group name.",
			param.AltNames("g"),
			param.PostAction(
				func(_ location.L, _ *param.ByName, _ []string) error {
					if prog.outputFileName == dfltFileName {
						prog.outputFileName = groupFileNameBase +
							prog.makeGroupSuffix() + ".go"
					}

					return nil
				}),
		)

		ps.Add("must-exist",
			psetter.Bool{Value: &prog.mustExist},
			"the config file will be checked to ensure that it does exist and"+
				" it will be an error if it doesn't",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("must-exist-personal",
			psetter.Bool{
				Value: &prog.mustExistPersonal,
			},
			"the personal config file will be checked to ensure that it"+
				" does exist and it will be an error if it doesn't",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("must-exist-global",
			psetter.Bool{Value: &prog.mustExistGlobal},
			"the global config file will be checked to ensure that it"+
				" does exist and it will be an error if it doesn't",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("base-dir-personal",
			psetter.String[string]{
				Value:  &prog.baseDirPersonal,
				Checks: []check.String{checkStringNotEmpty},
			},
			"set the base directory in which the parameter file will"+
				" be found. This value will be used in place of the"+
				" XDG config directory for personal config files."+
				" The sub-directories (derived from the import path)"+
				" will still be used",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("base-dir-global",
			psetter.String[string]{
				Value:  &prog.baseDirGlobal,
				Checks: []check.String{checkStringNotEmpty},
			},
			"set the base directory in which the parameter file will"+
				" be found. This value will be used in place of the"+
				" XDG config directory for global config files. The"+
				" sub-directories (derived from the import path)"+
				" will still be used",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("funcs",
			psetter.Enum[string]{
				Value: &prog.whichFuncs,
				AllowedVals: psetter.AllowedVals[string]{
					"all": "create all functions",
					"personalOnly": "create just the personal config file" +
						" setter function",
					"globalOnly": "create just the global config file" +
						" setter function",
				},
			},
			"specify which of the two functions (the global or the personal)"+
				" should be created",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("private", psetter.Bool{Value: &prog.privateFunc},
			"this will generate private (non-global) function names",
		)

		return nil
	}
}
