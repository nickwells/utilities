// retirement
package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/nickwells/col.mod/col"
	"github.com/nickwells/col.mod/col/colfmt"
	"github.com/nickwells/param.mod/v2/param"
	"github.com/nickwells/param.mod/v2/param/paramset"
)

var randomRange float64
var portfolio float64 = 1.0e3
var income float64 = 0.1e3
var inflation float64 = 3.0
var portfolioReturn float64 = 7.0
var crashProp float64 = 40.0
var years int64 = 30
var yearsDefered int64
var crashInterval int64
var keepSafePerc float64

// convertPctToProp will convert the values entered on the command line (or
// the defaults) from percentage values into proportions
func convertPctToProp() {
	randomRange /= 100
	inflation /= 100
	portfolioReturn /= 100
	crashProp /= 100
	keepSafePerc /= 100
}

// isACrashYear returns true if the year is one where a crash is expected
func isACrashYear(y int64) bool {
	if crashInterval > 0 && y%crashInterval == 0 {
		return true
	}
	return false
}

// addNote will add the string s to the note and return the resulting
// string. If s is not empty then a separator will be added between them.
func addNote(note, s string) string {
	if s == "" {
		return note
	}
	if note != "" {
		note += " | "
	}
	note += s
	return note
}

func main() {
	ps, _ := paramset.New(addParams,
		param.SetProgramDescription(
			"this will simulate various scenarios for retirement"+
				" allowing you to explore the effect of changes"+
				" in your portfolio, inflation etc"))
	ps.Parse()

	reportParams()
	fmt.Println()

	convertPctToProp()

	realReturn := portfolioReturn - inflation

	h, err := col.NewHeader()
	if err != nil {
		fmt.Println("Error found while constructing the header:", err)
		os.Exit(1)
	}
	rpt, err := col.NewReport(h, os.Stdout,
		col.New(colfmt.Int{}, "Year"),
		col.New(colfmt.Float{W: 6}, "Savings"),
		col.New(colfmt.Float{W: 6}, "Drawing"),
		col.New(colfmt.Float{W: 6, Prec: 1}, "%age", "of Savings"),
		col.New(colfmt.Float{W: 6}, "inflation adjusted", "Savings"),
		col.New(colfmt.Float{W: 6}, "inflation adjusted", "Drawing"),
		col.New(colfmt.Float{W: 6, Prec: 2}, "this year's", "return"),
		col.New(colfmt.String{W: 10}, "notes"))
	if err != nil {
		fmt.Println("Error found while constructing the report:", err)
		os.Exit(1)
	}

	targetIncome := income
	inflationAdjustment := 1.0
	rand.Seed(int64(time.Now().Nanosecond()))
	for y := int64(1); y <= years; y++ {
		randomVar := rand.Float64()*2*randomRange - randomRange

		notes := ""

		if isACrashYear(y) {
			portfolio *= (1 - crashProp)
			notes = addNote(notes, fmt.Sprintf("Crash: %5.2f%%", 100*crashProp))
		} else {
			portfolio *= (1 + portfolioReturn + randomVar)
		}
		inflationAdjustment *= (1 + inflation)
		income *= (1 + inflation)
		targetIncome *= (1 + inflation)

		if y > yearsDefered {
			portfolio -= income
		} else {
			notes = addNote(notes, "(withdrawal defered)")
		}

		proportion := income / portfolio

		if proportion > (realReturn - keepSafePerc) {
			notes = addNote(notes, fmt.Sprintf("drawing too much: %.1f > %.1f",
				proportion*100.0, (realReturn-keepSafePerc)*100.0))
		}

		if keepSafePerc > 0 {
			income = math.Min(
				portfolio*(realReturn-keepSafePerc),
				targetIncome)
		}
		_ = rpt.PrintRow(y,
			portfolio, income, 100*proportion,
			portfolio/inflationAdjustment, income/inflationAdjustment,
			100*(portfolioReturn+randomVar),
			notes)
	}
}

// reportParams will report the settings of various parameters
func reportParams() {
	h, err := col.NewHeader()
	if err != nil {
		fmt.Println("Error found while constructing the header:", err)
		os.Exit(1)
	}

	initVals, err := col.NewReport(h, os.Stdout,
		col.New(colfmt.String{W: 20, StrJust: col.Right}, "parameter").
			SetSep(": "),
		col.New(colfmt.Float{W: 6, Prec: 1}, "value"),
		col.New(colfmt.Percent{W: 5, Prec: 1, IgnoreNil: true}))
	if err != nil {
		fmt.Println("Error found while constructing the initial values table:",
			err)
		os.Exit(1)
	}

	_ = initVals.PrintRow("initial Savings", portfolio, nil)
	_ = initVals.PrintRow("target Drawing", income, income/portfolio)
	_ = initVals.PrintRow("expected inflation", inflation, nil)
	_ = initVals.PrintRow("expected return", portfolioReturn, nil)
	_ = initVals.PrintRow("real return", portfolioReturn-inflation, nil)
}
