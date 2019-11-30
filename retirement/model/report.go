package model

import (
	"fmt"
	"os"
	"time"

	"github.com/nickwells/col.mod/v2/col"
	"github.com/nickwells/col.mod/v2/col/colfmt"
	"github.com/nickwells/twrap.mod/twrap"
)

// makeRpt creates the report object
func makeRpt(m M) *col.Report {
	h, err := col.NewHeader()
	if err != nil {
		fmt.Println("Error found while constructing the header:", err)
		os.Exit(1)
	}

	inflColHead := "inflation adjusted"
	yearCol := []*col.Col{
		col.New(&colfmt.Int{}, "Year"),
	}
	portfolioCols := []*col.Col{
		col.New(&colfmt.Float{W: 6}, inflColHead, "Portfolio", "min"),
		col.New(&colfmt.Percent{W: 6, Prec: 2},
			inflColHead, "Portfolio", "shrunk"),
		col.New(&colfmt.Float{W: 6}, inflColHead, "Portfolio", "avg"),
		col.New(&colfmt.Float{W: 6}, inflColHead, "Portfolio", "SD"),
		col.New(&colfmt.Float{W: 6}, inflColHead, "Portfolio", "max"),
	}
	drawingCols := []*col.Col{
		col.New(&colfmt.Float{W: 6}, inflColHead, "Drawing", "min"),
		col.New(&colfmt.Float{W: 6}, inflColHead, "Drawing", "avg"),
		col.New(&colfmt.Float{W: 6}, inflColHead, "Drawing", "SD"),
		col.New(&colfmt.Float{W: 6}, inflColHead, "Drawing", "max"),
	}
	summaryCols := []*col.Col{
		col.New(&colfmt.Percent{W: 7, Prec: 2}, "average", "%age of", "Savings"),
		col.New(&colfmt.Percent{W: 7, Prec: 2}, "average", "nett", "return"),
		col.New(&colfmt.Percent{W: 6, Prec: 2}, "drawing", "covered"),
		col.New(&colfmt.Percent{W: 6, Prec: 2}, "drawing", "minimal"),
		col.New(&colfmt.Percent{W: 8, Prec: 4}, "chance", "of going", "bust"),
	}
	columns := []*col.Col{}
	columns = append(columns, yearCol...)
	columns = append(columns, portfolioCols...)
	columns = append(columns, drawingCols...)
	columns = append(columns, summaryCols...)
	rpt, err := col.NewReport(h, os.Stdout, columns...)
	if err != nil {
		fmt.Println("Error found while constructing the report:", err)
		os.Exit(1)
	}

	return rpt
}

// colVals creates the column values for passing to the report
func colVals(m M, lastPfl float64, r *AggResults) ([]interface{}, float64) {
	minInc, avgInc, sdInc, maxInc, _ := r.income.vals()
	minPfl, avgPfl, sdPfl, maxPfl, _ := r.portfolio.vals()
	vals := []interface{}{
		r.year,
		minPfl,
		float64(r.portfolioDown) / float64(m.trials),
		avgPfl, sdPfl, maxPfl,
		minInc, avgInc, sdInc, maxInc,
		avgInc / avgPfl,
		(avgPfl - lastPfl) / lastPfl,
		float64(r.surplusAvailable) / float64(m.trials),
		float64(r.minimalIncome) / float64(m.trials),
		float64(r.bust) / float64(m.trials),
	}

	return vals, avgPfl
}

// Report prints the results
func (m M) Report(results []*AggResults) {
	if m.showIntroText {
		m.printIntroText()
	}
	if m.showModelParams {
		m.reportModelParams()
	}

	fmt.Println()
	rpt := makeRpt(m)
	lastPfl := m.initialPortfolio
	var vals []interface{}
	for i, r := range results {
		vals, lastPfl = colVals(m, lastPfl, r)
		if i%int(m.yearsToShow) == 0 || i == len(results)-1 {
			err := rpt.PrintRow(vals...)
			if err != nil {
				fmt.Println("Bad row:", err)
				os.Exit(1)
			}
		}
	}
}

// printIntroText prints the introductory text which explains the model
func (m M) printIntroText() {
	twc := twrap.NewTWConfOrPanic()
	twc.Wrap("This report shows the expected behaviour of your portfolio."+
		"\n\nThe behaviour is modelled over a number of trials and the"+
		" aggregate results are shown. The model starts by calculating the"+
		" income to be drawn from the portfolio; the first year this is"+
		" the target income. Then at the end of each simulated year it will:",
		0)
	twc.Wrap2Indent("- look back at the return from the year just passed"+
		"\n- this is then reduced by the target minimum growth plus inflation"+
		"\n- the resulting figure is taken to be the available income"+
		"\n- if this amount is greater than the target income then the"+
		" next year's drawing is set to the target income and the number"+
		" of times a surplus was available is incremented."+
		"\n- if the amount is less than the minimum income then the next"+
		" year's drawing is set to the minimum income and the number of"+
		" times we had to take the minimum income is incremented"+
		"- if the available amount lies between the two figures then that"+
		" is taken as the next year's drawing",
		2, 4)
	twc.Wrap("The report shows the proportion of time that the drawing is"+
		" fully covered by the income received, the proportion of time that"+
		" the minimal income was taken and the cumulative proportion of"+
		" times that all the money is spent (that you go bust)",
		0)
	twc.Wrap("figures are all shown adjusted for inflation - that is they"+
		" are shown in today's pounds/dollars/etc",
		0)
}

// reportModelParams will report the model parameters
func (m M) reportModelParams() {
	h, err := col.NewHeader()
	if err != nil {
		fmt.Println("Error found while constructing the header for params:",
			err)
		os.Exit(1)
	}

	rpt, err := col.NewReport(h, os.Stdout,
		col.New(&colfmt.Percent{W: 6, Prec: 2}, "Inflation"),
		col.New(&colfmt.Float{W: 6}, "Initial", "Portfolio"),
		col.New(&colfmt.Percent{W: 6, Prec: 2}, "Growth", "", "Mean"),
		col.New(&colfmt.Percent{W: 6, Prec: 2}, "Growth", "", "SD"),
		col.New(&colfmt.Percent{W: 6, Prec: 2}, "Growth", "Target", "Min"),
		col.New(&colfmt.Float{W: 6}, "Income", "", "Target"),
		col.New(&colfmt.Float{W: 6}, "Income", "", "Min"),
		col.New(colfmt.Int{W: 6}, "Income", "drawings", "per yr"),
		col.New(colfmt.Int{W: 6}, "Income", "years", "defered"),
		col.New(colfmt.Int{W: 6}, "Crash", "interval"),
		col.New(&colfmt.Percent{W: 6, Prec: 2}, "Crash", "%age"),
		col.New(colfmt.Int{W: 6}, "Model", "years"),
		col.New(colfmt.Int{W: 7}, "Model", "trials p/a"),
		col.New(colfmt.Int{W: 6}, "Model", "yrs shown"),
	)
	if err != nil {
		fmt.Println("Error found while constructing the report for params:",
			err)
		os.Exit(1)
	}

	fmt.Println()
	err = rpt.PrintRow(
		m.inflationPct/100,
		m.initialPortfolio,
		m.rtnMeanPct/100, m.rtnSDPct/100, m.minGrowthPct/100,
		m.targetIncome, m.minIncome, m.drawingPeriodsPerYear, m.yearsDefered,
		m.crashInterval, m.crashPct/100,
		m.years, m.trials, m.yearsToShow)
	if err != nil {
		fmt.Println("Couldn't print the model parameters:", err)
	}
}

// ReportModelMetrics reports the metrics on the model performance
func (m M) ReportModelMetrics() {
	if !m.showModelMetrics {
		return
	}

	h, err := col.NewHeader()
	if err != nil {
		fmt.Println("Error found while constructing the header for metrics:",
			err)
		os.Exit(1)
	}

	rpt, err := col.NewReport(h, os.Stdout,
		col.New(colfmt.Int{W: 6}, "threads"),
		col.New(colfmt.Int{W: 8}, "time taken (µs)", "overall"),
	)
	if err != nil {
		fmt.Println("Error found while constructing the report for metrics:",
			err)
		os.Exit(1)
	}
	fmt.Println()
	err = rpt.PrintRow(
		m.modelMetrics.threadCount,
		m.modelMetrics.durCalcValues.D.Nanoseconds()/int64(time.Microsecond))
	if err != nil {
		fmt.Println("Couldn't print the model parameters:", err)
	}
}
