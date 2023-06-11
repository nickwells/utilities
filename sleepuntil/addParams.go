package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

const (
	paramGroupNameActions = "cmd-actions"
	paramGroupNameTime    = "cmd-time"

	paramNameRepeat = "repeat"
)

var (
	wakeupActionParamCounter paction.Counter
	timeParamCounter         paction.Counter
)

// addActionParams adds the program parameters to the PSet
func addActionParams(prog *Prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.AddGroup(paramGroupNameActions,
			"what to do when the sleeping finishes.")
		countParams := wakeupActionParamCounter.MakeActionFunc()

		ps.Add("message",
			psetter.String{
				Value: &prog.msg,
				Checks: []check.String{
					check.StringLength[string](check.ValGT[int](0)),
				},
			},
			"print this message when you wake up",
			param.AltNames("msg"),
			param.PostAction(countParams),
			param.GroupName(paramGroupNameActions),
		)

		ps.Add("run",
			psetter.String{
				Value: &prog.afterSleepCmd,
				Checks: []check.String{
					check.StringLength[string](check.ValGT[int](0)),
				},
			},
			"run the command in a subshell when you wake up",
			param.AltNames("do"),
			param.PostAction(countParams),
			param.GroupName(paramGroupNameActions),
		)

		ps.Add("show-time", psetter.Bool{Value: &prog.showTime},
			"show the target time when you wake up",
			param.PostAction(countParams),
			param.GroupName(paramGroupNameActions),
		)

		ps.Add("format", psetter.String{Value: &prog.showTimeFmt},
			"the format to use when showing the time."+
				" Setting this value forces the show-time flag on.",
			param.PostAction(paction.SetBool(&prog.showTime, true)),
			param.PostAction(countParams),
			param.GroupName(paramGroupNameActions),
		)

		ps.AddFinalCheck(checkActionParams(prog))

		return nil
	}
}

// addParams adds the program parameters to the PSet
func addParams(prog *Prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add(paramNameRepeat, psetter.Bool{Value: &prog.repeat},
			"repeatedly sleep. Sleep and then sleep again and again...",
			param.AltNames("r"),
		)

		ps.Add("repeat-count",
			psetter.Int64{
				Value: &prog.repeatCount, Checks: []check.Int64{
					check.ValGT[int64](0),
				},
			},
			"the number of times to repeat the operation.",
			param.AltNames("times", "rc"),
			param.PostAction(paction.SetBool(&prog.repeat, true)))

		ps.Add("dont-sleep", psetter.Bool{Value: &prog.doSleep, Invert: true},
			"do everything except sleep - useful for testing the behaviour",
			param.Attrs(param.DontShowInStdUsage))

		return nil
	}
}

// addTimeParams adds the program parameters relating to specifying how
// long to sleep for to the PSet
func addTimeParams(prog *Prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.AddGroup(paramGroupNameTime, "specify how long to sleep for.")
		countTimeParams := timeParamCounter.MakeActionFunc()

		var (
			needAbsTime bool
			hasAbsTime  bool
		)

		absTimeLocation, _ := time.LoadLocation("Local")

		var absTimeStr string

		const absTimeFormat = "20060102 15:04:05"

		ps.Add("utc", psetter.Bool{Value: &prog.useUTC},
			"use UTC time",
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("timezone", psetter.TimeLocation{Value: &absTimeLocation},
			"the timezone that the time is in. If this is supplied then an"+
				" absolute time must also be given.",
			param.AltNames("tz", "location"),
			param.PostAction(paction.SetBool(&needAbsTime, true)),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("time",
			psetter.String{
				Value: &absTimeStr,
				Checks: []check.String{
					check.StringLength[string](check.ValEQ[int](len(absTimeFormat))),
				},
			},
			"the actual time to sleep until. Format: '"+absTimeFormat+"'."+
				" This cannot be used with the "+paramNameRepeat+" argument",
			param.AltNames("t"),
			param.PostAction(countTimeParams),
			param.PostAction(paction.SetBool(&hasAbsTime, true)),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("minute",
			psetter.Int64{
				Value: &prog.timeMins,
				Checks: []check.Int64{
					check.ValGT[int64](0),
					check.ValDivides[int64](24 * 60),
				},
			},
			"the minute to sleep until",
			param.AltNames("m", "min", "minutes"),
			param.PostAction(countTimeParams),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("second",
			psetter.Int64{
				Value: &prog.timeSecs,
				Checks: []check.Int64{
					check.ValGT[int64](0),
					check.ValDivides[int64](24 * 60 * 60),
				},
			},
			"the second to sleep until",
			param.AltNames("s", "sec", "seconds"),
			param.PostAction(countTimeParams),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("per-day",
			psetter.Int64{
				Value: &prog.perDay,
				Checks: []check.Int64{
					check.ValGT[int64](0),
					check.ValDivides[int64](24 * 60 * 60),
				},
			},
			"the number of parts to split the day into. "+
				"A value of 24 would sleep until the next hour",
			param.PostAction(countTimeParams),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("per-hour",
			psetter.Int64{
				Value: &prog.perHour,
				Checks: []check.Int64{
					check.ValGT[int64](0),
					check.ValDivides[int64](60 * 60),
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
				Value: &prog.perMinute,
				Checks: []check.Int64{
					check.ValGT[int64](0),
					check.ValDivides[int64](60),
				},
			},
			"the number of parts to split the minute into."+
				" A value of 20 would sleep until the next 3-second period.",
			param.PostAction(countTimeParams),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("offset",
			psetter.Int64{
				Value: &prog.offset,
				Checks: []check.Int64{
					check.Not(
						check.ValEQ[int64](0),
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
				Value: &prog.offsetMins,
				Checks: []check.Int64{
					check.Not(
						check.ValEQ[int64](0),
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

			if prog.repeat {
				return errors.New(
					"You cannot give an absolute time and ask that the sleep" +
						" should repeat.")
			}

			var err error
			prog.absTime, err = time.ParseInLocation(
				"20060102 15:04:05",
				absTimeStr,
				absTimeLocation)
			if err != nil {
				return fmt.Errorf("couldn't parse target time: %q: %w",
					absTimeStr, err)
			}
			now := time.Now()
			if prog.useUTC {
				now = now.UTC()
			}
			if prog.absTime.Sub(now) < 0 {
				return fmt.Errorf("the target time %q is in the past", prog.absTime)
			}
			return nil
		})

		ps.AddFinalCheck(checkTimeParams(prog))

		return nil
	}
}

// checkActionParams checks that at most one of the time specification
// parameters has been set
func checkActionParams(prog *Prog) func() error {
	return func() error {
		if wakeupActionParamCounter.Count() == 0 && prog.repeat {
			return errors.New(
				"you have chosen to repeat the sleep" +
					" but have not specified any actions")
		}
		return nil
	}
}

// checkTimeParams checks that at most one of the time specification
// parameters has been set
func checkTimeParams(prog *Prog) func() error {
	return func() error {
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
}
