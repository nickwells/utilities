// retirement
package main

import (
	"github.com/nickwells/param.mod/v3/param"
	"github.com/nickwells/param.mod/v3/param/paramset"
	"github.com/nickwells/utilities/retirement/model"
)

// main
func main() {
	m := model.New()
	ps := paramset.NewOrDie(
		m.MakeAddParamsFunc(),
		SetConfigFile,
		param.SetProgramDescription(
			"this will simulate various scenarios for retirement"+
				" allowing you to explore the effect of changes"+
				" in your portfolio, inflation etc"))
	ps.Parse()

	m.Report(m.CalcValues())

	m.ReportModelMetrics()
}
