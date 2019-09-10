package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v3/param"
	"github.com/nickwells/param.mod/v3/param/paction"
	"github.com/nickwells/param.mod/v3/param/psetter"
)

const baseGroupName = param.DfltGroupName

var fromZone = time.Local
var toZone = time.Local
var fromZoneParam *param.ByName

const dateFmt = "20060102"

var inFormat = dateFmt + " 15:04:05"
var outFormat = inFormat

const timestampFormat = "20060102.150405"
const iso8601Format = "2006-01-02T15:04:05"
const httpFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

func setFormatToTimestamp(_ location.L, _ *param.ByName, _ []string) error {
	outFormat = timestampFormat
	return nil
}

func setFormatToISO8601(_ location.L, _ *param.ByName, _ []string) error {
	outFormat = iso8601Format
	return nil
}

func setFormatToHTTP(_ location.L, _ *param.ByName, _ []string) error {
	outFormat = httpFormat
	toZone = time.UTC
	return nil
}

var fmtCounter paction.Counter

var useUSDateOrder bool
var noSecs bool
var noCentury bool
var showTimezone bool
var showAMPM bool
var showMonthName bool
var datePartSep = ""
var fmtFlagCounter paction.Counter

var dtStr string
var dtParam *param.ByName
var tStr string
var tParam *param.ByName
var dtCounter paction.Counter

// addParams will add the parameters for the timeconv program to the set
// of params
func addParams(ps *param.PSet) error {
	if err := addTimezoneParams(ps); err != nil {
		return err
	}

	if err := addTimeSettingParams(ps); err != nil {
		return err
	}

	if err := addTimeFormattingParams(ps); err != nil {
		return err
	}

	ps.AddFinalCheck(paramChecker)

	return nil
}

// addTimezoneParams will add the parameters for setting timezones
func addTimezoneParams(ps *param.PSet) error {
	const tzGroupname = baseGroupName + "-timezone"

	ps.AddGroup(tzGroupname, "time-zone parameters")

	fromZoneParam = ps.Add("from-zone",
		psetter.TimeLocation{Value: &fromZone},
		`the timezone in which to interpret the supplied date and time`,
		param.GroupName(tzGroupname))

	ps.Add("to-zone",
		psetter.TimeLocation{Value: &toZone},
		`the timezone in which to present the supplied date and time`,
		param.GroupName(tzGroupname))

	return nil
}

// addTimeSettingParams adds the parameters used to set the time to be
// converted. The default time is the current time
func addTimeSettingParams(ps *param.PSet) error {
	dtCounterAF := (&dtCounter).MakeActionFunc()
	const timeGroupname = baseGroupName + "-setting"

	ps.AddGroup(timeGroupname, "time-setting parameters\n\n"+
		"These allow you to set the time to be converted."+
		" The default is to use the current time")

	dtParam = ps.Add("date-time",
		psetter.String{Value: &dtStr},
		"the date and time. Note that the date is in the form of the year,"+
			" including the century, the month number and"+
			" the day of the month with leading zeros and no spaces."+
			" Then the time is separated from the year by a single space"+
			" and is in 24-hour form with a leading zero and"+
			" a colon (':') between the hours, minutes and seconds.\n\n"+
			"For instance: '20190321 15:10:30'",
		param.AltName("dt"),
		param.GroupName(timeGroupname),
		param.PostAction(dtCounterAF))

	tParam = ps.Add("time",
		psetter.String{Value: &tStr},
		"the time to be converted."+
			" Note that the time is in 24-hour form with a leading zero and"+
			" a colon (':') between the hours, minutes and seconds.\n\n"+
			"For instance: '15:10:30'\n\n"+
			"When only the time is given the date is taken as the"+
			" current date in the source timezone which could be"+
			" a day before or after the current time",
		param.AltName("t"),
		param.GroupName(timeGroupname),
		param.PostAction(dtCounterAF))

	return nil
}

// addTimeFormattingParams will add the parameters used to control the output
// of the program
func addTimeFormattingParams(ps *param.PSet) error {
	fmtCounterAF := (&fmtCounter).MakeActionFunc()
	fmtFlagCounterAF := (&fmtFlagCounter).MakeActionFunc()

	const fmtGroupname = baseGroupName + "-formatting"

	ps.AddGroup(fmtGroupname, "formatting parameters\n\n"+
		"These are used to control how the resulting date and"+
		" time are shown to the user. You can either set the output"+
		" format directly or else give parameters to control the"+
		" appearance of different parts of the formatted time")

	ps.Add("format",
		psetter.String{Value: &outFormat},
		"the format in which to display the resulting date and time."+
			" Note that this format uses the Go programming language"+
			" format specification.\n\n"+
			"You can specify precisely how the time should appear"+
			" as follows:\n\n"+
			"for the year use '06' (or '2006' for the century as well)\n"+
			"for the month use '1', '01', 'Jan' or 'January'\n"+
			"for the day of the month use '2' or '02'\n"+
			"to show the day of the week use 'Mon' or 'Monday'\n"+
			"for the hour use '03' (or '15' for a 24-hour clock)\n"+
			"for the minute and second use '04' and '05'\n"+
			"for fractions of a second add '.000'\n"+
			"to show AM or PM use 'PM'\n"+
			"to show the timezone use 'MST'\n\n"+
			"unrecognised strings will appear as given",
		param.AltName("fmt"),
		param.GroupName(fmtGroupname),
		param.PostAction(fmtCounterAF))

	ps.Add("format-timestamp",
		psetter.Nil{},
		`set the output format to one suitable for use as a timestamp:

`+timestampFormat,
		param.AltName("fmt-ts"),
		param.GroupName(fmtGroupname),
		param.PostAction(fmtCounterAF),
		param.PostAction(setFormatToTimestamp))

	ps.Add("format-iso8601",
		psetter.Nil{},
		`set the output format to that given by ISO 8601:

`+iso8601Format,
		param.AltName("fmt-iso"),
		param.GroupName(fmtGroupname),
		param.PostAction(fmtCounterAF),
		param.PostAction(setFormatToISO8601))

	ps.Add("format-http",
		psetter.Nil{},
		`set the output format to the preferred HTTP format:

`+httpFormat+`

This will also set the output timezone to UTC (GMT) but this can be overridden by following parameters in which case the format will not be HTTP standard compliant. Also, be aware that the GMT at the end of the displayed time is a fixed string and will not change to reflect any change in timezone.`,
		param.AltName("fmt-http"),
		param.GroupName(fmtGroupname),
		param.PostAction(fmtCounterAF),
		param.PostAction(setFormatToHTTP))

	ps.Add("us-date-order",
		psetter.Bool{Value: &useUSDateOrder},
		`display the date in US format: month day year`,
		param.AltName("us-date-fmt"),
		param.GroupName(fmtGroupname),
		param.PostAction(fmtFlagCounterAF))

	ps.Add("no-seconds",
		psetter.Bool{Value: &noSecs},
		`display the  time without showing the seconds`,
		param.AltName("no-secs"),
		param.GroupName(fmtGroupname),
		param.PostAction(fmtFlagCounterAF))

	ps.Add("no-century",
		psetter.Bool{Value: &noCentury},
		`display the date without showing the century`,
		param.GroupName(fmtGroupname),
		param.PostAction(fmtFlagCounterAF))

	ps.Add("show-timezone",
		psetter.Bool{Value: &showTimezone},
		`display the time with the timezone`,
		param.GroupName(fmtGroupname),
		param.PostAction(fmtFlagCounterAF))

	ps.Add("show-ampm",
		psetter.Bool{Value: &showAMPM},
		`display the  time in AM/PM format not 24 hour`,
		param.GroupName(fmtGroupname),
		param.PostAction(fmtFlagCounterAF))

	ps.Add("show-month-name",
		psetter.Bool{Value: &showMonthName},
		`display the month name rather than the number`,
		param.GroupName(fmtGroupname),
		param.PostAction(fmtFlagCounterAF))

	ps.Add("date-part-sep",
		psetter.String{Value: &datePartSep},
		`separate the parts of the date with the given value`,
		param.GroupName(fmtGroupname),
		param.PostAction(fmtFlagCounterAF))

	return nil
}

// paramChecker will check that the parameters make sense
func paramChecker() error {
	// First check the sense of the parameters used to set the time to be
	// converted
	if fromZoneParam.HasBeenSet() {
		if !dtParam.HasBeenSet() &&
			!tParam.HasBeenSet() {
			return fmt.Errorf(
				"if you have specified '%s' you must give '%s' or '%s'",
				fromZoneParam.Name(), dtParam.Name(), tParam.Name())
		}
	}

	if dtCounter.Count() > 1 {
		return fmt.Errorf("you may set at most one of '%s' or '%s'",
			dtParam.Name(), tParam.Name())
	}

	// Now check the sense of the parameters used to control the output
	// of the converted time
	if fmtFlagCounter.Count() >= 1 {
		if fmtCounter.Count() >= 1 {
			return fmt.Errorf(
				"the output format has been set (%s)"+
					" and so have the format flags (%s)",
				fmtCounter.SetBy(), fmtFlagCounter.SetBy())
		}

		timePart := makeTimePart()
		datePart := makeDatePart()

		outFormat = datePart + " " + timePart
	}

	if fmtCounter.Count() > 1 {
		return errors.New("the output format has been set multiple times: " +
			fmtCounter.SetBy())
	}

	return nil
}

// makeTimePart constructs the time part of the output format
func makeTimePart() string {
	hourPart := "15"
	AMPMsuffix := ""
	if showAMPM {
		hourPart = "03"
		AMPMsuffix = " PM"
	}
	secsPart := ":05"
	if noSecs {
		secsPart = ""
	}
	TZPart := ""
	if showTimezone {
		TZPart = " MST"
	}
	return hourPart + ":" + "04" + secsPart + AMPMsuffix + TZPart
}

// makeDatePart makes the datepart of the format string
func makeDatePart() string {
	monthPart := "01"
	if showMonthName {
		monthPart = "Jan"
	}
	yearPart := "2006"
	if noCentury {
		yearPart = "06"
	}
	datePart := yearPart + datePartSep + monthPart + datePartSep + "02"
	if useUSDateOrder {
		datePart = monthPart + datePartSep + "02" + datePartSep + yearPart
	}

	return datePart
}
