package main

import (
	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/param.mod/v3/param"
	"github.com/nickwells/param.mod/v3/param/psetter"
)

func addParams(ps *param.PSet) error {
	ps.Add("portfolio",
		psetter.Float64{
			Value: &portfolio,
			Checks: []check.Float64{
				check.Float64GT(0.0),
			},
		},
		"set the starting size of your retirement portfolio",
		param.AltName("p"))

	ps.Add("income",
		psetter.Float64{
			Value: &income,
			Checks: []check.Float64{
				check.Float64GT(0.0),
			},
		},
		"set your desired retirement income",
		param.AltName("i"))

	ps.Add("inflation",
		psetter.Float64{
			Value: &inflation,
			Checks: []check.Float64{
				check.Float64GT(0.0),
			},
		},
		"set your expected inflation rate",
		param.AltName("ei"))

	ps.Add("return",
		psetter.Float64{
			Value: &portfolioReturn,
			Checks: []check.Float64{
				check.Float64GT(0.0),
			},
		},
		"set your expected annual return on the portfolio",
		param.AltName("r"))

	ps.Add("years",
		psetter.Int64{
			Value: &years,
			Checks: []check.Int64{
				check.Int64GT(0),
			},
		},
		"set the number of years to simulate over",
		param.AltName("y"))

	ps.Add("defer",
		psetter.Int64{
			Value: &yearsDefered,
			Checks: []check.Int64{
				check.Int64GT(0),
			},
		},
		"set the number of years to defer the start of withdrawing funds",
		param.AltName("d"))

	ps.Add("rand",
		psetter.Float64{
			Value: &randomRange,
			Checks: []check.Float64{
				check.Float64GT(0),
			},
		},
		"set the range of the random variation around the average return")

	ps.Add("crash-interval",
		psetter.Int64{
			Value: &crashInterval,
			Checks: []check.Int64{
				check.Int64GT(0),
			},
		},
		"set the number of years between market crashes. If this value is "+
			"not set then there will be no crashes in the simulation",
		param.AltName("ci"))

	ps.Add("crash-prop",
		psetter.Float64{
			Value: &crashProp,
			Checks: []check.Float64{
				check.Float64GT(0.0),
			},
		},
		"set the proportion of the portfolio to lose in a market crash. "+
			"If the crash interval value is not set then there will be no "+
			"crashes in the simulation",
		param.AltName("cp"))

	ps.Add("keep-safe",
		psetter.Float64{
			Value: &keepSafePerc,
		},
		"If the amount being withdrawn is too much then reduce it to this "+
			"much less than the real rate of return",
		param.AltName("adjust"))

	return nil
}
