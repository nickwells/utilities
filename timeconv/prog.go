package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nickwells/tempus.mod/tempus"
)

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

// prog holds program parameters and status
type prog struct {
	fromZone *time.Location
	toZone   *time.Location

	outFormat string

	useUSDateOrder bool
	noSecs         bool
	noCentury      bool
	showDate       bool
	showTimezone   bool
	showAMPM       bool
	showMonthName  bool

	dateTimeSep string
	datePartSep string
	dtStr       string
	tStr        string
	timeSource  int

	tzNames     []string
	listTZNames bool
}

// newProg returns a new Prog instance with the default values set
func newProg() *prog {
	return &prog{
		showDate:    true,
		dateTimeSep: " ",
		fromZone:    time.Local,
		toZone:      time.Local,
		outFormat:   inFormat,
		timeSource:  tsNow,
		tzNames:     tempus.TimezoneNames(),
	}
}

// listTimezoneNames displays the Timezone names
func (prog prog) listTimezoneNames() {
	for _, n := range prog.tzNames {
		fmt.Println(n)
	}
}

// parseTime parses the supplied time string according to the input format in
// the from-timezone. If the first attempt fails it will check that the
// seconds are present and if not it will try again with the seconds set to
// 00.
func (prog prog) parseTime(ts string) (time.Time, error) {
	const zeroSeconds = ":00"

	lenSeconds := len(zeroSeconds)

	t, err := time.ParseInLocation(inFormat, ts, prog.fromZone)
	if err == nil {
		return t, nil
	}

	if len(ts) == len(inFormat)-lenSeconds {
		ts += zeroSeconds

		var err2 error

		t, err2 = time.ParseInLocation(inFormat, ts, prog.fromZone)
		if err2 == nil {
			return t, nil
		}
	}

	return t, err
}

// getTime returns the time according to the parameters given
func (prog prog) getTime() time.Time {
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

	return time.Time{}
}

// makeTimePart constructs the time part of the output format
func (prog prog) makeTimePart() string {
	hourPart := "15"
	AMPMsuffix := ""

	if prog.showAMPM {
		hourPart = "03"
		AMPMsuffix = " PM"
	}

	secsPart := ":05"

	if prog.noSecs {
		secsPart = ""
	}

	TZPart := ""

	if prog.showTimezone {
		TZPart = " MST"
	}

	return hourPart + ":" + "04" + secsPart + AMPMsuffix + TZPart
}

// makeDatePart makes the datepart of the format string
func (prog prog) makeDatePart() string {
	monthPart := "01"

	if prog.showMonthName {
		monthPart = "Jan"
	}

	yearPart := "2006"

	if prog.noCentury {
		yearPart = "06"
	}

	if prog.useUSDateOrder {
		return monthPart + prog.datePartSep + "02" + prog.datePartSep + yearPart
	}

	return yearPart + prog.datePartSep + monthPart + prog.datePartSep + "02"
}

// setOutputFormat sets the output format in accordance with the format
// specifications
func (prog *prog) setOutputFormat() {
	prog.outFormat = ""

	if prog.showDate {
		prog.outFormat = prog.makeDatePart() + prog.dateTimeSep
	}

	prog.outFormat += prog.makeTimePart()
}
