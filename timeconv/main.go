package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/tempus.mod/tempus"
)

// Created: Sun Oct 22 11:17:41 2017

const (
	tsNow = iota
	tsDateTimeStr
	tsTimeStr
)

const (
	dfltDateFmt = "20060102"
	dfltTimeFmt = "15:04:05"
	inFormat    = dfltDateFmt + " " + dfltTimeFmt
)

// Prog holds program parameters and status
type Prog struct {
	fromZone *time.Location
	toZone   *time.Location

	outFormat string

	useUSDateOrder bool
	noSecs         bool
	noCentury      bool
	showTimezone   bool
	showAMPM       bool
	showMonthName  bool

	datePartSep string
	dtStr       string
	tStr        string
	timeSource  int

	fromZoneParam *param.ByName
	dtParam       *param.ByName
	tParam        *param.ByName

	tzNames     []string
	listTZNames bool
}

// NewProg returns a new Prog instance with the default values set
func NewProg() *Prog {
	return &Prog{
		fromZone:   time.Local,
		toZone:     time.Local,
		outFormat:  inFormat,
		timeSource: tsNow,
		tzNames:    tempus.TimezoneNames(),
	}
}

// parseTime parses the supplied time string according to the input format in
// the from-timezone. If the first attempt fails it will check that the
// seconds are present and if not it will try again with the seconds set to
// 00.
func (prog *Prog) parseTime(ts string) (time.Time, error) {
	t, err := time.ParseInLocation(inFormat, ts, prog.fromZone)
	if err == nil {
		return t, nil
	}
	if len(ts) == len(inFormat)-3 {
		ts += ":00"
		var err2 error
		t, err2 = time.ParseInLocation(inFormat, ts, prog.fromZone)
		if err2 == nil {
			return t, nil
		}
	}
	return t, err
}

// getTime returns the time according to the parameters given
func (prog *Prog) getTime() time.Time {
	switch prog.timeSource {
	case tsNow:
		return time.Now()
	case tsDateTimeStr:
		tIn, err := prog.parseTime(prog.dtStr)
		if err != nil {
			fmt.Println("Cannot parse the date and time:", err)
			os.Exit(1)
		}
		return tIn
	case tsTimeStr:
		dtStr := time.Now().In(prog.fromZone).Format(dfltDateFmt) +
			" " + prog.tStr
		tIn, err := prog.parseTime(dtStr)
		if err != nil {
			fmt.Println("Cannot parse the time:", err)
			os.Exit(1)
		}
		return tIn
	}

	fmt.Println("Unknown time source:", prog.timeSource)
	os.Exit(1)
	return time.Time{}
}

// listTimezoneNames displays the Timezone names
func listTimezoneNames(prog *Prog) {
	for _, n := range prog.tzNames {
		fmt.Println(n)
	}
	os.Exit(0)
}

func main() {
	prog := NewProg()
	ps := makeParamSet(prog)
	ps.Parse()

	if prog.listTZNames {
		listTimezoneNames(prog)
		os.Exit(0)
	}

	tIn := prog.getTime()

	tOut := tIn.In(prog.toZone)
	fmt.Println(tOut.Format(prog.outFormat))
}
