package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paction"
	"github.com/nickwells/param.mod/v6/psetter"
)

const (
	baseGroupName = param.DfltGroupName

	dateFmt         = "20060102"
	timeFmt         = "15:04:05"
	timestampFormat = "20060102.150405"
	iso8601Format   = "2006-01-02T15:04:05"
	httpFormat      = "Mon, 02 Jan 2006 15:04:05 GMT"
)

// setFormat returns an action func that will set the output format (and, if
// zone is not nil, the toZone)
func setFormat(prog *Prog, fmt string, zone *time.Location) param.ActionFunc {
	return func(_ location.L, _ *param.ByName, _ []string) error {
		prog.outFormat = fmt
		if zone != nil {
			prog.toZone = zone
		}
		return nil
	}
}

// addTimezoneParams will add the parameters for setting timezones
func addTimezoneParams(prog *Prog, fromZoneParam **param.ByName,
) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		const tzGroupname = baseGroupName + "-timezone"

		ps.AddGroup(tzGroupname, "time-zone parameters")

		*fromZoneParam = ps.Add("from-zone",
			psetter.TimeLocation{Value: &prog.fromZone},
			`the timezone in which to interpret the supplied date and time`,
			param.GroupName(tzGroupname))

		ps.Add("to-zone",
			psetter.TimeLocation{Value: &prog.toZone},
			`the timezone in which to present the supplied date and time`,
			param.GroupName(tzGroupname))

		return nil
	}
}

// addTimeSettingParams adds the parameters used to set the time to be
// converted. The default time is the current time
func addTimeSettingParams(prog *Prog, dtParam, tParam **param.ByName,
) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		const timeGroupname = baseGroupName + "-setting"

		ps.AddGroup(timeGroupname, "time-setting parameters\n\n"+
			"These allow you to set the time to be converted."+
			" The default is to use the current time")

		*dtParam = ps.Add("date-time",
			psetter.String[string]{Value: &prog.dtStr},
			"the date and time. Note that the date is in the form of the year,"+
				" including the century, the month number and"+
				" the day of the month with leading zeros and no spaces."+
				" Then the time is separated from the year by a single space"+
				" and is in 24-hour form with a leading zero and"+
				" a colon (':') between the hours, minutes and seconds.\n\n"+
				"For instance: '20190321 15:10:30'",
			param.AltNames("dt"),
			param.GroupName(timeGroupname),
			param.PostAction(paction.SetVal[int](&prog.timeSource, tsDateTimeStr)),
		)

		*tParam = ps.Add("time",
			psetter.String[string]{Value: &prog.tStr},
			"the time to be converted."+
				" Note that the time is in 24-hour form with"+
				" a leading zero and"+
				" a colon (':') between the hours, minutes and seconds.\n\n"+
				"For instance: '15:10:30'\n\n"+
				"When only the time is given the date is taken as the"+
				" current date in the source timezone which could be"+
				" a day before or after the current time",
			param.AltNames("from", "t"),
			param.GroupName(timeGroupname),
			param.PostAction(paction.SetVal[int](&prog.timeSource, tsTimeStr)),
		)

		ps.AddFinalCheck(func() error {
			if (*dtParam).HasBeenSet() &&
				(*tParam).HasBeenSet() {
				return fmt.Errorf("you may set at most one of %q or %q",
					(*dtParam).Name(), (*tParam).Name())
			}
			return nil
		})

		return nil
	}
}

// addTimeFormattingParams will add the parameters used to control the output
// of the program
func addTimeFormattingParams(prog *Prog) param.PSetOptFunc {
	var (
		fmtFlagCounter paction.Counter
		fmtCounter     paction.Counter
	)
	return func(ps *param.PSet) error {
		fmtCounterAF := (&fmtCounter).MakeActionFunc()
		fmtFlagCounterAF := (&fmtFlagCounter).MakeActionFunc()

		const fmtGroupname = baseGroupName + "-formatting"

		ps.AddGroup(fmtGroupname, "formatting parameters\n\n"+
			"These are used to control how the resulting date and"+
			" time are shown to the user. You can either set the output"+
			" format directly or else give parameters to control the"+
			" appearance of different parts of the formatted time")

		ps.Add("format",
			psetter.String[string]{Value: &prog.outFormat},
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
			param.AltNames("fmt"),
			param.GroupName(fmtGroupname),
			param.PostAction(fmtCounterAF))

		ps.Add("format-timestamp",
			psetter.Nil{},
			"set the output format to one suitable for use as a timestamp:"+
				"\n\n"+
				timestampFormat,
			param.AltNames("fmt-ts"),
			param.GroupName(fmtGroupname),
			param.PostAction(fmtCounterAF),
			param.PostAction(setFormat(prog, timestampFormat, nil)),
		)

		ps.Add("format-iso8601",
			psetter.Nil{},
			"set the output format to that given by ISO 8601:"+
				"\n\n"+
				iso8601Format,
			param.AltNames("fmt-iso"),
			param.GroupName(fmtGroupname),
			param.PostAction(fmtCounterAF),
			param.PostAction(setFormat(prog, iso8601Format, nil)),
		)

		ps.Add("format-http",
			psetter.Nil{},
			"set the output format to the preferred HTTP format:"+
				"\n\n"+
				httpFormat+
				"\n\n"+
				"This will also set the output timezone to UTC (GMT)"+
				" but this can be overridden by following parameters"+
				" in which case the format will not be HTTP standard"+
				" compliant. Also, be aware that the GMT at the end"+
				" of the displayed time is a fixed string and will not"+
				" change to reflect any change in timezone.",
			param.AltNames("fmt-http"),
			param.GroupName(fmtGroupname),
			param.PostAction(fmtCounterAF),
			param.PostAction(setFormat(prog, httpFormat, time.UTC)),
		)

		ps.Add("us-date-order",
			psetter.Bool{Value: &prog.useUSDateOrder},
			`display the date in US format: month day year`,
			param.AltNames("us-date-fmt"),
			param.GroupName(fmtGroupname),
			param.PostAction(fmtFlagCounterAF))

		ps.Add("no-seconds",
			psetter.Bool{Value: &prog.noSecs},
			`display the  time without showing the seconds`,
			param.AltNames("no-secs"),
			param.GroupName(fmtGroupname),
			param.PostAction(fmtFlagCounterAF))

		ps.Add("no-century",
			psetter.Bool{Value: &prog.noCentury},
			`display the date without showing the century`,
			param.GroupName(fmtGroupname),
			param.PostAction(fmtFlagCounterAF))

		ps.Add("show-timezone",
			psetter.Bool{Value: &prog.showTimezone},
			`display the time with the timezone`,
			param.GroupName(fmtGroupname),
			param.PostAction(fmtFlagCounterAF))

		ps.Add("show-ampm",
			psetter.Bool{Value: &prog.showAMPM},
			`display the  time in AM/PM format not 24 hour`,
			param.GroupName(fmtGroupname),
			param.PostAction(fmtFlagCounterAF))

		ps.Add("show-month-name",
			psetter.Bool{Value: &prog.showMonthName},
			`display the month name rather than the number`,
			param.GroupName(fmtGroupname),
			param.PostAction(fmtFlagCounterAF))

		ps.Add("date-part-sep",
			psetter.String[string]{Value: &prog.datePartSep},
			`separate the parts of the date with the given value`,
			param.GroupName(fmtGroupname),
			param.PostAction(fmtFlagCounterAF))

		// Final checks
		ps.AddFinalCheck(func() error {
			if fmtFlagCounter.Count() >= 1 && fmtCounter.Count() >= 1 {
				return fmt.Errorf(
					"the output format has been set (%s)"+
						" and so have the format flags (%s)",
					fmtCounter.SetBy(), fmtFlagCounter.SetBy())
			}

			return nil
		})

		ps.AddFinalCheck(func() error {
			if fmtFlagCounter.Count() >= 1 {
				timeFmt := prog.makeTimePart()
				dateFmt := prog.makeDatePart()

				prog.outFormat = dateFmt + " " + timeFmt
			}
			return nil
		})

		ps.AddFinalCheck(func() error {
			if fmtCounter.Count() > 1 {
				return errors.New(
					"the output format has been set multiple times: " +
						fmtCounter.SetBy())
			}
			return nil
		})

		return nil
	}
}

// addTimeFormattingParams will add the parameters used to control the output
// of the program
func addParamChecks(fromZoneParam, dtParam, tParam *param.ByName,
) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.AddFinalCheck(func() error {
			if fromZoneParam.HasBeenSet() {
				if !dtParam.HasBeenSet() &&
					!tParam.HasBeenSet() {
					return fmt.Errorf(
						"if you have specified %q you must give %q or %q",
						fromZoneParam.Name(), dtParam.Name(), tParam.Name())
				}
			}
			return nil
		})
		return nil
	}
}

// makeTimePart constructs the time part of the output format
func (prog *Prog) makeTimePart() string {
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
func (prog *Prog) makeDatePart() string {
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
