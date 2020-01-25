package model

import (
	"fmt"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/param.mod/v3/param"
	"github.com/nickwells/param.mod/v3/param/psetter"
)

// (m *M)MakeAddParamsFunc creates and returns a function that will set the
// parameters on a param set which will update entries in the model
func (m *M) MakeAddParamsFunc() param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add("portfolio",
			psetter.Float64{
				Value: &m.initialPortfolio,
				Checks: []check.Float64{
					check.Float64GT(0.0),
				},
			},
			"set the starting size of your retirement portfolio",
			param.AltName("p"),
			param.Attrs(param.MustBeSet))

		ps.Add("income",
			psetter.Float64{
				Value: &m.targetIncome,
				Checks: []check.Float64{
					check.Float64GT(0.0),
				},
			},
			"set your desired retirement income",
			param.AltName("i"),
			param.Attrs(param.MustBeSet))

		ps.Add("inflation",
			psetter.Float64{
				Value: &m.inflationPct,
				Checks: []check.Float64{
					check.Float64GT(0.0),
				},
			},
			"set your expected percentage inflation rate",
			param.AltName("ei"))

		ps.Add("return",
			psetter.Float64{
				Value: &m.rtnMeanPct,
				Checks: []check.Float64{
					check.Float64GT(0.0),
				},
			},
			"set your expected annual percentage return on the portfolio",
			param.AltName("r"))

		ps.Add("defer",
			psetter.Int64{
				Value: &m.yearsDefered,
				Checks: []check.Int64{
					check.Int64GT(0),
				},
			},
			"set the number of years to defer the start of withdrawing funds",
			param.AltName("d"))

		ps.Add("return-range",
			psetter.Float64{
				Value: &m.rtnSDPct,
				Checks: []check.Float64{
					check.Float64GT(0),
				},
			},
			"set the range of the random variation around the average return."+
				" This should be the standard deviation of the returns",
			param.AltName("sd"))

		ps.Add("crash-interval",
			psetter.Int64{
				Value: &m.crashInterval,
				Checks: []check.Int64{
					check.Int64GT(0),
				},
			},
			"set the number of years between market crashes. If this value is"+
				" not set then there will be no crashes in the simulation,"+
				" otherwise there will, on average, be a crash every this"+
				" many years",
			param.AltName("ci"))

		ps.Add("crash-prop",
			psetter.Float64{
				Value: &m.crashPct,
				Checks: []check.Float64{
					check.Float64GT(0.0),
				},
			},
			"set the percentage by which the portfolio will decline in value"+
				" in a market crash."+
				" If the crash interval value is not set then there will be no"+
				" crashes in the simulation",
			param.AltName("cp"))

		ps.Add("min-return", psetter.Float64{Value: &m.minGrowthPct},
			"this is a desired minimum real rate of growth of the portfolio."+
				" The income taken from the portfolio will be adjusted to"+
				" try to ensure that the portfolio grows by at least this much"+
				" plus inflation each year, subject to the minimum income")

		ps.Add("periods",
			psetter.Int64{
				Value: &m.drawingPeriodsPerYear,
				Checks: []check.Int64{
					check.Int64GT(0),
				},
			},
			"how many periods should the year be split into - are you going to"+
				" take income once a year"+
				" 4 times a year (quarterly)"+
				" 12 times a year (monthly)"+
				" 13 times a year (every 4 weeks)"+
				" or 52 times a year (weekly)",
			param.AltName("drawings-per-year"),
		)

		ps.Add("min-income", psetter.Float64{Value: &m.minIncome},
			"set the lowest income that you can afford to receive")

		ps.Add("years",
			psetter.Int64{
				Value: &m.years,
				Checks: []check.Int64{
					check.Int64GT(0),
				},
			},
			"set the number of years to simulate over",
			param.AltName("y"))

		ps.Add("trials",
			psetter.Int64{
				Value: &m.trials,
				Checks: []check.Int64{
					check.Int64GT(0),
				},
			},
			"set the number of trials per year",
			param.AltName("t"))

		ps.Add("extreme-set-size",
			psetter.Int64{
				Value: &m.extremeSetSize,
				Checks: []check.Int64{
					check.Int64GT(0),
				},
			},
			"the size of the set of extreme values. This is used to"+
				" smooth the maximum and minimum values. The value"+
				" reported is the average of the values in this set. For"+
				" instance setting this to 10 would mean that the minimum"+
				" reported would be the average of the 10 smallest values"+
				" observed.")

		ps.Add("show-every-n-years", psetter.Int64{Value: &m.yearsToShow},
			"only report every nth year (and the last)",
			param.AltName("show-yrs"))

		ps.Add("show-intro", psetter.Bool{Value: &m.showIntroText},
			"print a description of the model before showing the results")

		ps.Add("show-model-params", psetter.Bool{Value: &m.showModelParams},
			"report the parameters to the model before showing the results")

		ps.Add("show-model-metrics", psetter.Bool{Value: &m.showModelMetrics},
			"show various metrics about the model's performance",
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage))

		ps.AddFinalCheck(checkIncomeBounds(m))

		return nil
	}
}

// checkIncomeBounds checks that the target income and the minimum income are
// in the right relation to one another and sets the minIncome value if it
// has not be set
func checkIncomeBounds(m *M) param.FinalCheckFunc {
	return func() error {
		if m.minIncome > m.targetIncome {
			return fmt.Errorf("the minimum income (%.1f)"+
				" must be less than or equal to the target (%.1f)",
				m.minIncome, m.targetIncome)
		}

		return nil
	}
}
