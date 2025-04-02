package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v6/paction"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
	"github.com/nickwells/tempus.mod/tempus"
)

const (
	paramNameFromZone          = "from-zone"
	paramNameToZone            = "to-zone"
	paramNameListTimezoneNames = "list-timezone-names"

	groupNameTimezone   = param.DfltGroupName + "-timezone"
	groupNameSetting    = param.DfltGroupName + "-setting"
	groupNameFormatting = param.DfltGroupName + "-formatting"
)

// setFormat returns an action func that will set the output format (and, if
// zone is not nil, the toZone)
func setFormat(prog *prog, fmt string, zone *time.Location) param.ActionFunc {
	return func(_ location.L, _ *param.ByName, _ []string) error {
		prog.outFormat = fmt
		if zone != nil {
			prog.toZone = zone
		}

		return nil
	}
}

// addParams adds the parameters for this program
//
//nolint:cyclop
func addParams(prog *prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		var toZoneParam,
			fromZoneParam,
			dtParam,
			tParam *param.ByName

		const timeDesc = "The time must be given in 24-hour form with" +
			" a leading zero and a colon (':') between" +
			" the hours, minutes and seconds," +
			" for instance: '15:10:30'" +
			"\n\n" +
			"If no seconds are given they are taken to be zero"

		// add the setting parameter group
		ps.AddGroup(groupNameSetting, "time-setting parameters\n\n"+
			"These allow you to set the time to be converted."+
			" The default is to use the current time")

		fromZoneParam = ps.Add(paramNameFromZone,
			psetter.TimeLocation{
				Value:     &prog.fromZone,
				Locations: prog.tzNames,
			},
			`the timezone in which to interpret the supplied date and time`,
			param.AltNames("from-timezone", "from-tz"),
			param.GroupName(groupNameSetting))

		dtParam = ps.Add("date-time",
			psetter.String[string]{Value: &prog.dtStr},
			"the date and time to be converted."+
				"\n\n"+
				"The date must be given in the form of the year"+
				" (including the century) the month number and"+
				" the day of the month with leading zeros and no spaces."+
				"\n\n"+
				"Then the time is separated from the date by a single space."+
				"\n\n"+
				timeDesc+
				"\n\n"+
				"For instance: '20190321 15:10:30'",
			param.AltNames("dt"),
			param.GroupName(groupNameSetting),
			param.PostAction(paction.SetVal(
				&prog.timeSource, tsDateTimeStr)),
		)

		tParam = ps.Add("time",
			psetter.String[string]{Value: &prog.tStr},
			"the time to be converted."+
				"\n\n"+
				timeDesc+
				"\n\n"+
				"The date used is the"+
				" current date in the source timezone which could be"+
				" a day before or after the current time in your timezone.",
			param.AltNames("from", "t"),
			param.GroupName(groupNameSetting),
			param.PostAction(paction.SetVal(&prog.timeSource, tsTimeStr)),
		)

		ps.AddFinalCheck(func() error {
			if dtParam.HasBeenSet() &&
				tParam.HasBeenSet() {
				return fmt.Errorf("you may set at most one of %q or %q",
					dtParam.Name(), tParam.Name())
			}

			return nil
		})

		ps.AddFinalCheck(func() error {
			if fromZoneParam.HasBeenSet() {
				if !dtParam.HasBeenSet() &&
					!tParam.HasBeenSet() {
					return fmt.Errorf(
						"if you have specified %q you must give %q or %q",
						fromZoneParam.Name(),
						dtParam.Name(),
						tParam.Name())
				}
			}

			return nil
		})

		// add the timezone parameter group

		ps.AddGroup(groupNameTimezone, "time-zone parameters")

		toZoneParam = ps.Add(paramNameToZone,
			psetter.TimeLocation{Value: &prog.toZone, Locations: prog.tzNames},
			`the timezone in which to present the supplied date and time`,
			param.AltNames("to-timezone", "to-tz"),
			param.GroupName(groupNameTimezone))

		if len(prog.tzNames) > 0 {
			ps.Add(paramNameListTimezoneNames,
				psetter.Bool{Value: &prog.listTZNames},
				`list all the available timezones`,
				param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
				param.AltNames("list-tz-names", "list-timezones"),
				param.GroupName(groupNameTimezone))

			err := param.SeeAlso(paramNameListTimezoneNames)(toZoneParam)
			if err != nil {
				return err
			}

			err = param.SeeAlso(paramNameListTimezoneNames)(fromZoneParam)
			if err != nil {
				return err
			}
		}

		// add the formatting parameter group

		var fmtFlagCounter,
			fmtCounter,
			dateFmtFlagCounter,
			noDateCounter paction.Counter

		fmtFlagCounterAF := (&fmtFlagCounter).MakeActionFunc()
		fmtCounterAF := (&fmtCounter).MakeActionFunc()
		dateFmtFlagCounterAF := (&dateFmtFlagCounter).MakeActionFunc()
		noDateCounterAF := (&noDateCounter).MakeActionFunc()

		ps.AddGroup(groupNameFormatting, "formatting parameters\n\n"+
			"These are used to control how the resulting date and"+
			" time are shown to the user. You can either set the output"+
			" format directly or else give parameters to control the"+
			" appearance of different parts of the formatted time")

		ps.Add("format",
			psetter.String[string]{Value: &prog.outFormat},
			"the format in which to display the resulting date and time."+
				" Note that this format uses the Go programming language"+
				" time formatting specification.\n\n"+
				"You can specify precisely how the time should appear"+
				" as follows:\n\n"+
				"for the year use '06' (or '2006' for the century as well)\n"+
				"for the month use '1', '01', 'Jan' or 'January'\n"+
				"for the day of the month use '2' or '02'\n"+
				"to show the day of the week use 'Mon' or 'Monday'\n"+
				"for the hour use '03' (or '15' for a 24-hour clock)\n"+
				"for the minute and second use '04' and '05'\n"+
				"for fractions of a second add '.' followed by 1-9 zeroes\n"+
				"to show AM or PM use 'PM'\n"+
				"to show the timezone use 'MST'\n\n"+
				"unrecognised strings will appear as given",
			param.AltNames("fmt"),
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtCounterAF),
		)

		ps.Add("format-timestamp",
			psetter.Nil{},
			"set the output format to one suitable for use as a timestamp:"+
				"\n\n"+
				tempus.FormatTimestamp,
			param.AltNames("fmt-ts"),
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtCounterAF),
			param.PostAction(setFormat(prog, tempus.FormatTimestamp, nil)),
		)

		ps.Add("format-iso8601",
			psetter.Nil{},
			"set the output format to that given by ISO 8601:"+
				"\n\n"+
				tempus.FormatISO8601,
			param.AltNames("fmt-iso"),
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtCounterAF),
			param.PostAction(setFormat(prog, tempus.FormatISO8601, nil)),
		)

		ps.Add("format-http",
			psetter.Nil{},
			"set the output format to the preferred HTTP format:"+
				"\n\n"+
				tempus.FormatHTTP+
				"\n\n"+
				"This will also set the output timezone to UTC (GMT)"+
				" but this can be overridden by following parameters"+
				" in which case the format will not be HTTP standard"+
				" compliant. Also, be aware that the GMT at the end"+
				" of the displayed time is a fixed string and will not"+
				" change to reflect any change in timezone.",
			param.AltNames("fmt-http"),
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtCounterAF),
			param.PostAction(setFormat(prog, tempus.FormatHTTP, time.UTC)),
		)

		ps.Add("us-date-order",
			psetter.Bool{Value: &prog.useUSDateOrder},
			`display the date in US format: month day year`,
			param.AltNames("us-date-fmt", "us-format", "us-fmt"),
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtFlagCounterAF),
			param.PostAction(dateFmtFlagCounterAF),
		)

		ps.Add("no-seconds",
			psetter.Bool{Value: &prog.noSecs},
			`display the  time without showing the seconds`,
			param.AltNames("no-secs", "dont-show-seconds", "dont-show-secs"),
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtFlagCounterAF),
		)

		ps.Add("no-century",
			psetter.Bool{Value: &prog.noCentury},
			`display the date without showing the century`,
			param.AltNames("dont-show-century"),
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtFlagCounterAF),
			param.PostAction(dateFmtFlagCounterAF),
		)

		ps.Add("no-date",
			psetter.Bool{Value: &prog.showDate, Invert: true},
			`don't display the date; just show the time`,
			param.AltNames("dont-show-date"),
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtFlagCounterAF),
			param.PostAction(noDateCounterAF),
		)

		ps.Add("show-timezone",
			psetter.Bool{Value: &prog.showTimezone},
			`display the time with the timezone`,
			param.AltNames("show-tz", "show-zone"),
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtFlagCounterAF),
		)

		ps.Add("show-ampm",
			psetter.Bool{Value: &prog.showAMPM},
			`display the  time in AM/PM format not 24 hour`,
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtFlagCounterAF),
		)

		ps.Add("show-month-name",
			psetter.Bool{Value: &prog.showMonthName},
			`display the month name rather than the number`,
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtFlagCounterAF),
			param.PostAction(dateFmtFlagCounterAF),
		)

		ps.Add("date-part-sep",
			psetter.String[string]{Value: &prog.datePartSep},
			`separate the parts of the date with the given value`,
			param.AltNames("date-part-separator"),
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtFlagCounterAF),
			param.PostAction(dateFmtFlagCounterAF),
		)

		ps.Add("date-time-sep",
			psetter.String[string]{Value: &prog.dateTimeSep},
			`separate the date from the time with the given value`,
			param.AltNames("date-time-separator"),
			param.GroupName(groupNameFormatting),
			param.PostAction(fmtFlagCounterAF),
			param.PostAction(dateFmtFlagCounterAF),
		)

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
			if noDateCounter.Count() >= 1 && dateFmtFlagCounter.Count() >= 1 {
				return fmt.Errorf("you've set the date format and"+
					" chosen not to display the date:\n\n%s\n\n%s",
					dateFmtFlagCounter.SetBy(),
					noDateCounter.SetBy(),
				)
			}

			return nil
		})

		ps.AddFinalCheck(func() error {
			if fmtFlagCounter.Count() >= 1 {
				prog.setOutputFormat()
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
