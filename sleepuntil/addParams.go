package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/param.mod/v6/paction"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
	"github.com/nickwells/tempus.mod/tempus"
)

const (
	paramGroupNameActions = "cmd-actions"
	paramGroupNameTime    = "cmd-time"

	paramNameRepeat = "repeat"
)

// addActionParams adds the program parameters to the PSet
func addActionParams(prog *prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.AddGroup(paramGroupNameActions,
			"what to do when the sleeping finishes.")

		var wakeupAction paction.Counter
		wakeupActionAF := wakeupAction.MakeActionFunc()

		ps.Add("message",
			psetter.String[string]{
				Value: &prog.msg,
				Checks: []check.String{
					check.StringLength[string](check.ValGT(0)),
				},
			},
			"print this message when you wake up",
			param.AltNames("msg"),
			param.PostAction(wakeupActionAF),
			param.GroupName(paramGroupNameActions),
		)

		ps.Add("run",
			psetter.String[string]{
				Value: &prog.afterSleepCmd,
				Checks: []check.String{
					check.StringLength[string](check.ValGT(0)),
				},
			},
			"run the command in a subshell when you wake up",
			param.AltNames("do"),
			param.PostAction(wakeupActionAF),
			param.GroupName(paramGroupNameActions),
		)

		ps.Add("show-time", psetter.Bool{Value: &prog.showTime},
			"show the target time when you wake up",
			param.PostAction(wakeupActionAF),
			param.GroupName(paramGroupNameActions),
		)

		ps.Add("format", psetter.String[string]{Value: &prog.showTimeFmt},
			"the format to use when showing the time."+
				" Setting this value forces the show-time flag on.",
			param.PostAction(paction.SetVal(&prog.showTime, true)),
			param.PostAction(wakeupActionAF),
			param.GroupName(paramGroupNameActions),
		)

		ps.AddFinalCheck(func() error {
			if wakeupAction.Count() == 0 && prog.repeat {
				return errors.New(
					"you have chosen to repeat the sleep" +
						" but have not specified any actions")
			}

			return nil
		})

		return nil
	}
}

// addParams adds the program parameters to the PSet
func addParams(prog *prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add(paramNameRepeat, psetter.Bool{Value: &prog.repeat},
			"repeatedly sleep. Sleep and then sleep again and again...",
			param.AltNames("r"),
		)

		ps.Add("repeat-count",
			psetter.Int[int64]{
				Value: &prog.repeatCount, Checks: []check.Int64{
					check.ValGT[int64](0),
				},
			},
			"the number of times to repeat the operation.",
			param.AltNames("times", "rc"),
			param.PostAction(paction.SetVal(&prog.repeat, true)))

		ps.Add("dont-sleep", psetter.Bool{Value: &prog.doSleep, Invert: true},
			"do everything except sleep - useful for testing the behaviour",
			param.Attrs(param.DontShowInStdUsage))

		return nil
	}
}

// addTimeParams adds the program parameters relating to specifying how
// long to sleep for to the PSet
func addTimeParams(prog *prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.AddGroup(paramGroupNameTime, "specify how long to sleep for.")

		var timeVal paction.Counter
		timeValAF := timeVal.MakeActionFunc()

		var (
			needAbsTime bool
			hasAbsTime  bool
		)

		const absTimeFormat = "20060102 15:04:05"

		var (
			absTimeLocation, _ = time.LoadLocation("Local")
			absTimeStr         string
			absTimeLen         = len(absTimeFormat)
		)

		ps.Add("utc", psetter.Bool{Value: &prog.useUTC},
			"use UTC time",
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("timezone", psetter.TimeLocation{Value: &absTimeLocation},
			"the timezone that the time is in. If this is supplied then an"+
				" absolute time must also be given.",
			param.AltNames("tz", "location"),
			param.PostAction(paction.SetVal(&needAbsTime, true)),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("time",
			psetter.String[string]{
				Value: &absTimeStr,
				Checks: []check.String{
					check.StringLength[string](check.ValEQ(absTimeLen)),
				},
			},
			"the actual time to sleep until. Format: '"+absTimeFormat+"'."+
				" This cannot be used with the "+paramNameRepeat+" argument",
			param.AltNames("t"),
			param.PostAction(timeValAF),
			param.PostAction(paction.SetVal(&hasAbsTime, true)),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("minute",
			psetter.Int[int64]{
				Value: &prog.timeMins,
				Checks: []check.Int64{
					check.ValGT[int64](0),
					check.ValDivides[int64](tempus.MinutesPerDay),
				},
			},
			"the minute to sleep until",
			param.AltNames("m", "min", "minutes"),
			param.PostAction(timeValAF),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("second",
			psetter.Int[int64]{
				Value: &prog.timeSecs,
				Checks: []check.Int64{
					check.ValGT[int64](0),
					check.ValDivides[int64](tempus.SecondsPerDay),
				},
			},
			"the second to sleep until",
			param.AltNames("s", "sec", "seconds"),
			param.PostAction(timeValAF),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("per-day",
			psetter.Int[int64]{
				Value: &prog.perDay,
				Checks: []check.Int64{
					check.ValGT[int64](0),
					check.ValDivides[int64](tempus.SecondsPerDay),
				},
			},
			"the number of parts to split the day into. "+
				"A value of 24 would sleep until the next hour",
			param.PostAction(timeValAF),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("per-hour",
			psetter.Int[int64]{
				Value: &prog.perHour,
				Checks: []check.Int64{
					check.ValGT[int64](0),
					check.ValDivides[int64](tempus.SecondsPerHour),
				},
			},
			"the number of parts to split the hour into."+
				" A value of 20 would sleep until the next 3-minute period."+
				" This would be minute 00, 03, 06, 09, 12 etc.",
			param.PostAction(timeValAF),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("per-min",
			psetter.Int[int64]{
				Value: &prog.perMinute,
				Checks: []check.Int64{
					check.ValGT[int64](0),
					check.ValDivides[int64](tempus.SecondsPerMinute),
				},
			},
			"the number of parts to split the minute into."+
				" A value of 20 would sleep until the next 3-second period.",
			param.PostAction(timeValAF),
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("offset",
			psetter.Int[int64]{
				Value: &prog.offset,
				Checks: []check.Int64{
					check.Not(
						check.ValEQ[int64](0),
						"The offset must not be zero"),
				},
			},
			"set an offset to the calculated time (in seconds)."+
				"\n\n"+
				" The two offsets are added together to give"+
				" a combined offset.",
			param.GroupName(paramGroupNameTime),
		)

		ps.Add("offset-mins",
			psetter.Int[int64]{
				Value: &prog.offsetMins,
				Checks: []check.Int64{
					check.Not(
						check.ValEQ[int64](0),
						"The offset must not be zero"),
				},
			},
			"set an offset to the calculated time (in minutes)."+
				"\n\n"+
				" The two offsets are added together to give"+
				" a combined offset.",
			param.GroupName(paramGroupNameTime),
		)

		ps.AddFinalCheck(func() error {
			if needAbsTime && !hasAbsTime {
				return errors.New("a location (timezone) has been specified" +
					" for a time to be interpreted in but no time has been" +
					" given")
			}

			if !hasAbsTime {
				return nil
			}

			if prog.repeat {
				return errors.New(
					"you cannot give an absolute time and ask that the sleep" +
						" should repeat")
			}

			var err error

			prog.absTime, err = time.ParseInLocation(
				absTimeFormat,
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
				return fmt.Errorf("the target time %q is in the past",
					prog.absTime)
			}

			return nil
		})

		ps.AddFinalCheck(func() error {
			if timeVal.Count() == 0 {
				return errors.New("no time specification has been set")
			}

			if timeVal.Count() > 1 {
				return fmt.Errorf(
					"the time specification has been set more than once: %s",
					timeVal.SetBy())
			}

			return nil
		})

		return nil
	}
}
