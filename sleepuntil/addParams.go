package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

const (
	paramGroupNameActions = "cmd-actions"
	paramGroupNameTime    = "cmd-time"

	paramNameRepeat = "repeat"
)

var wakeupActionParamCounter paction.Counter
var timeParamCounter paction.Counter

// addActionParams adds the program parameters to the PSet
func addActionParams(ps *param.PSet) error {
	ps.AddGroup(paramGroupNameActions,
		"what to do when the sleeping finishes.")
	countParams := wakeupActionParamCounter.MakeActionFunc()

	ps.Add("message",
		psetter.String{
			Value: &msg,
			Checks: []check.String{
				check.StringLenGT(0),
			},
		},
		"print this message when you wake up",
		param.AltName("msg"),
		param.PostAction(countParams),
		param.GroupName(paramGroupNameActions),
	)

	ps.Add("run",
		psetter.String{
			Value: &afterSleepCmd,
			Checks: []check.String{
				check.StringLenGT(0),
			},
		},
		"run the command in a subshell when you wake up",
		param.AltName("do"),
		param.PostAction(countParams),
		param.GroupName(paramGroupNameActions),
	)

	ps.Add("show-time", psetter.Bool{Value: &showTime},
		"show the target time when you wake up",
		param.PostAction(countParams),
		param.GroupName(paramGroupNameActions),
	)

	ps.Add("format", psetter.String{Value: &showTimeFmt},
		"the format to use when showing the time."+
			" Setting this value forces the show-time flag on.",
		param.PostAction(paction.SetBool(&showTime, true)),
		param.PostAction(countParams),
		param.GroupName(paramGroupNameActions),
	)

	ps.AddFinalCheck(checkActionParams)

	return nil
}

// addParams adds the program parameters to the PSet
func addParams(ps *param.PSet) error {
	ps.Add(paramNameRepeat, psetter.Bool{Value: &repeat},
		"repeatedly sleep. Sleep and then sleep again and again...",
		param.AltName("r"),
	)

	ps.Add("repeat-count",
		psetter.Int64{
			Value: &repeatCount, Checks: []check.Int64{
				check.Int64GT(0),
			},
		},
		"the number of times to repeat the operation.",
		param.AltName("times"),
		param.AltName("rc"),
		param.PostAction(paction.SetBool(&repeat, true)))

	ps.Add("dont-sleep", psetter.Bool{Value: &doSleep, Invert: true},
		"do everything except sleep - useful for testing the behaviour",
		param.Attrs(param.DontShowInStdUsage))

	return nil
}

// addTimeParams adds the program parameters relating to specifying how
// long to sleep for to the PSet
func addTimeParams(ps *param.PSet) error {
	ps.AddGroup(paramGroupNameTime, "specify how long to sleep for.")
	countTimeParams := timeParamCounter.MakeActionFunc()

	var needAbsTime bool
	var hasAbsTime bool
	var absTimeLocation, _ = time.LoadLocation("Local")
	var absTimeStr string
	const absTimeFormat = "20060102 15:04:05"

	ps.Add("utc", psetter.Bool{Value: &useUTC},
		"use UTC time",
		param.GroupName(paramGroupNameTime),
	)

	ps.Add("timezone", psetter.TimeLocation{Value: &absTimeLocation},
		"the timezone that the time is in. If this is supplied then an"+
			" absolute time must also be given.",
		param.AltName("tz"),
		param.AltName("location"),
		param.PostAction(paction.SetBool(&needAbsTime, true)),
		param.GroupName(paramGroupNameTime),
	)

	ps.Add("time",
		psetter.String{
			Value: &absTimeStr,
			Checks: []check.String{
				check.StringLenEQ(len(absTimeFormat)),
			},
		},
		"the actual time to sleep until. Format: '"+absTimeFormat+"'."+
			" This cannot be used with the "+paramNameRepeat+" argument",
		param.AltName("t"),
		param.PostAction(countTimeParams),
		param.PostAction(paction.SetBool(&hasAbsTime, true)),
		param.GroupName(paramGroupNameTime),
	)

	ps.Add("minute",
		psetter.Int64{
			Value: &timeMins,
			Checks: []check.Int64{
				check.Int64GT(0),
				check.Int64Divides(24 * 60),
			},
		},
		"the minute to sleep until",
		param.AltName("m"),
		param.AltName("min"),
		param.AltName("minutes"),
		param.PostAction(countTimeParams),
		param.GroupName(paramGroupNameTime),
	)

	ps.Add("second",
		psetter.Int64{
			Value: &timeSecs,
			Checks: []check.Int64{
				check.Int64GT(0),
				check.Int64Divides(24 * 60 * 60),
			},
		},
		"the second to sleep until",
		param.AltName("s"),
		param.AltName("sec"),
		param.AltName("seconds"),
		param.PostAction(countTimeParams),
		param.GroupName(paramGroupNameTime),
	)

	ps.Add("per-day",
		psetter.Int64{
			Value: &perDay,
			Checks: []check.Int64{
				check.Int64GT(0),
				check.Int64Divides(24 * 60 * 60),
			},
		},
		"the number of parts to split the day into. "+
			"A value of 24 would sleep until the next hour",
		param.PostAction(countTimeParams),
		param.GroupName(paramGroupNameTime),
	)

	ps.Add("per-hour",
		psetter.Int64{
			Value: &perHour,
			Checks: []check.Int64{
				check.Int64GT(0),
				check.Int64Divides(60 * 60),
			},
		},
		"the number of parts to split the hour into."+
			" A value of 20 would sleep until the next 3-minute period."+
			" This would be minute 00, 03, 06, 09, 12 etc.",
		param.PostAction(countTimeParams),
		param.GroupName(paramGroupNameTime),
	)

	ps.Add("per-min",
		psetter.Int64{
			Value: &perMinute,
			Checks: []check.Int64{
				check.Int64GT(0),
				check.Int64Divides(60),
			},
		},
		"the number of parts to split the minute into."+
			" A value of 20 would sleep until the next 3-second period.",
		param.PostAction(countTimeParams),
		param.GroupName(paramGroupNameTime),
	)

	ps.Add("offset",
		psetter.Int64{
			Value: &offset,
			Checks: []check.Int64{
				check.Int64Not(
					check.Int64EQ(0),
					"The offset must not be zero"),
			},
		},
		"set an offset to the calculated time (in seconds)."+
			"\n\n"+
			" The two offsets are added together to give a combined offset.",
		param.GroupName(paramGroupNameTime),
	)

	ps.Add("offset-mins",
		psetter.Int64{
			Value: &offsetMins,
			Checks: []check.Int64{
				check.Int64Not(
					check.Int64EQ(0),
					"The offset must not be zero"),
			},
		},
		"set an offset to the calculated time (in minutes)."+
			"\n\n"+
			" The two offsets are added together to give a combined offset.",
		param.GroupName(paramGroupNameTime),
	)

	ps.AddFinalCheck(func() error {
		if needAbsTime && !hasAbsTime {
			return errors.New("A location (timezone) has been specified" +
				" for a time to be interpreted in but no time has been" +
				" given.")
		}
		if !hasAbsTime {
			return nil
		}

		if repeat {
			return errors.New(
				"You cannot give an absolute time and ask that the sleep" +
					" should repeat.")
		}

		var err error
		absTime, err = time.ParseInLocation(
			"20060102 15:04:05",
			absTimeStr,
			absTimeLocation)
		if err != nil {
			return fmt.Errorf("couldn't parse target time: %q: %w",
				absTimeStr, err)
		}
		now := time.Now()
		if useUTC {
			now = now.UTC()
		}
		if absTime.Sub(now) < 0 {
			return fmt.Errorf("the target time %q is in the past", absTime)
		}
		return nil
	})

	ps.AddFinalCheck(checkTimeParams)

	return nil
}

// checkActionParams checks that at most one of the time specification
// parameters has been set
func checkActionParams() error {
	if wakeupActionParamCounter.Count() == 0 && repeat {
		return errors.New(
			"you have chosen to repeat the sleep" +
				" but have not specified any actions")
	}
	return nil
}

// checkTimeParams checks that at most one of the time specification
// parameters has been set
func checkTimeParams() error {
	if timeParamCounter.Count() == 0 {
		return errors.New("no time specification has been set")
	}
	if timeParamCounter.Count() > 1 {
		return fmt.Errorf(
			"the time specification has been set more than once: %s",
			timeParamCounter.SetBy())
	}
	return nil
}
