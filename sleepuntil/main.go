package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

var showTime bool

const dfltShowTimeFmt = "20060102.150405"

var showTimeFmt = dfltShowTimeFmt

var useUTC bool
var doSleep = true

var repeat bool
var repeatCount int64 = -1

var afterSleepCmd string
var afterSleepCmdParam *param.ByName

var absTimeLocation, _ = time.LoadLocation("Local")
var absTime string
var absTimeParam *param.ByName

var timeMins int64
var timeMinsParam *param.ByName

var timeSecs int64
var timeSecsParam *param.ByName

var perDay int64
var perDayParam *param.ByName

var perHour int64
var perHourParam *param.ByName

var perMinute int64
var perMinuteParam *param.ByName

var msg string
var msgParam *param.ByName

var wakeupActionParamCounter paction.Counter
var timeSpecParamCounter paction.Counter

// checkWakeupActionParams checks that at most one of the time specification
// parameters has been set
func checkWakeupActionParams() error {
	if wakeupActionParamCounter.Count() == 0 && repeat {
		return errors.New(
			"you have chosen to repeat the sleep" +
				" but have not specified any actions")
	}
	return nil
}

// checkTimeSpecParams checks that at most one of the time specification
// parameters has been set
func checkTimeSpecParams() error {
	if timeSpecParamCounter.Count() == 0 {
		return errors.New("no time specification has been set")
	}
	if timeSpecParamCounter.Count() > 1 {
		return fmt.Errorf(
			"the time specification has been set more than once: %s",
			timeSpecParamCounter.SetBy())
	}
	return nil
}

// addWakeupActionParams adds the program parameters to the PSet
func addWakeupActionParams(ps *param.PSet) error {
	countParams := wakeupActionParamCounter.MakeActionFunc()

	msgParam = ps.Add("message", psetter.String{Value: &msg},
		"print this message when you wake up",
		param.AltName("msg"),
		param.PostAction(countParams))

	afterSleepCmdParam = ps.Add("run",
		psetter.String{Value: &afterSleepCmd},
		"run the command in a subshell when you wake up",
		param.PostAction(countParams))

	ps.Add("show-time", psetter.Bool{Value: &showTime},
		"show the target time when you wake up",
		param.PostAction(countParams))

	ps.Add("format", psetter.String{Value: &showTimeFmt},
		"the format to use when showing the time."+
			" Setting this value forces the show-time flag on.",
		param.PostAction(paction.SetBool(&showTime, true)),
		param.PostAction(countParams))

	ps.AddFinalCheck(checkWakeupActionParams)

	return nil
}

// addParams adds the program parameters to the PSet
func addParams(ps *param.PSet) error {
	ps.Add("repeat", psetter.Bool{Value: &repeat},
		"repeatedly sleep. Sleep and then sleep again and again...")

	ps.Add(
		"repeat-count",
		psetter.Int64{
			Value: &repeatCount, Checks: []check.Int64{
				check.Int64GT(0),
			},
		},
		"the number of times to repeat the operation."+
			" Setting this value forces the repeat flag on.",
		param.AltName("times"),
		param.PostAction(paction.SetBool(&repeat, true)))

	ps.Add("dont-sleep", psetter.Bool{Value: &doSleep, Invert: true},
		"do everything except sleep - useful for testing the behaviour",
		param.Attrs(param.DontShowInStdUsage))

	return nil
}

// addTimeSpecParams adds the program parameters relating to specifying how
// long to sleep for to the PSet
func addTimeSpecParams(ps *param.PSet) error {
	countTimeSpecParams := timeSpecParamCounter.MakeActionFunc()

	ps.Add("utc", psetter.Bool{Value: &useUTC}, "use UTC time")

	ps.Add("timezone", psetter.TimeLocation{Value: &absTimeLocation},
		"the timezone that the time is in")
	absTimeParam = ps.Add("time", psetter.String{Value: &absTime},
		"the actual time to sleep until. Format: 'YYYYMMDD HH:MM:SS'. "+
			"This cannot be used with the repeat argument",
		param.PostAction(countTimeSpecParams))

	timeMinsParam = ps.Add("min",
		psetter.Int64{
			Value: &timeMins,
			Checks: []check.Int64{
				check.Int64GT(0),
				check.Int64Divides(24 * 60),
			},
		},
		"the minute to sleep until",
		param.PostAction(countTimeSpecParams))

	timeSecsParam = ps.Add("sec",
		psetter.Int64{
			Value: &timeSecs,
			Checks: []check.Int64{
				check.Int64GT(0),
				check.Int64Divides(24 * 60 * 60),
			},
		},
		"the second to sleep until",
		param.PostAction(countTimeSpecParams))

	perDayParam = ps.Add("per-day",
		psetter.Int64{
			Value: &perDay,
			Checks: []check.Int64{
				check.Int64GT(0),
				check.Int64Divides(24 * 60 * 60),
			},
		},
		"the number of parts to split the day into. "+
			"A value of 24 would sleep until the next hour",
		param.PostAction(countTimeSpecParams))

	perHourParam = ps.Add("per-hour",
		psetter.Int64{
			Value: &perHour,
			Checks: []check.Int64{
				check.Int64GT(0),
				check.Int64Divides(60 * 60),
			},
		},
		"the number of parts to split the hour into. "+
			"A value of 20 would sleep until the next 3-minute period. "+
			"This would be minute 00, 03, 06, 09, 12 etc",
		param.PostAction(countTimeSpecParams))

	perMinuteParam = ps.Add("per-min",
		psetter.Int64{
			Value: &perMinute,
			Checks: []check.Int64{
				check.Int64GT(0),
				check.Int64Divides(60),
			},
		},
		"the number of parts to split the minute into. "+
			"A value of 20 would sleep until the next 3-second period",
		param.PostAction(countTimeSpecParams))

	ps.AddFinalCheck(checkTimeSpecParams)

	return nil
}

// sleepCalc calculates the time to sleep
func sleepCalc(durationSecs int64, now time.Time) time.Duration {
	s := int64(now.Second())
	s += int64(now.Minute()) * 60
	s += int64(now.Hour()) * 3600
	s *= 1e9
	s += int64(now.Nanosecond())
	durationNano := durationSecs * 1e9

	var sleepNano int64

	remainder := s % durationNano
	if remainder != 0 {
		sleepNano = durationNano - remainder
	}

	return time.Duration(sleepNano) * time.Nanosecond
}

// sleepToAbsTime parses the value of the absTime into a time in the given
// timezone (Local if no value is given) and sleeps until that time
func sleepToAbsTime(ps *param.PSet) {
	targetTime, err := time.ParseInLocation(
		"20060102 15:04:05",
		absTime,
		absTimeLocation)
	if err != nil {
		ps.Help("couldn't parse target time: " +
			absTime + " : " + err.Error())
	}

	if repeat {
		ps.Help("the repeat parameter cannot be used with the " +
			absTimeParam.Name() + " parameter")
	}

	now := time.Now()
	if useUTC {
		now = now.UTC()
	}
	dur := targetTime.Sub(now)
	if dur < 0 {
		ps.Help("the target time " + absTime + " is in the past")
	}
	sleepToTarget(now, dur, ps)
	runShellCmd(ps)
}

// runShellCmd will run the given command, if any, in a subshell. it will
// check for errors and report them; it exits on any error
func runShellCmd(ps *param.PSet) {
	if afterSleepCmdParam.HasBeenSet() {
		// verbose.Println("about to run: ", afterSleepCmd)

		out, err := exec.Command("/bin/bash", "-c",
			afterSleepCmd).CombinedOutput()
		fmt.Println(string(out))
		if err != nil {
			fmt.Println("Command failed:", err)
			os.Exit(1)
		}
	}
}

func durationSecs(ps *param.PSet) int64 {
	if timeSecsParam.HasBeenSet() {
		return timeSecs
	}
	if timeMinsParam.HasBeenSet() {
		return timeMins * 60
	}
	if perDayParam.HasBeenSet() {
		return 24 * 60 * 60 / perDay
	}
	if perHourParam.HasBeenSet() {
		return 60 * 60 / perHour
	}
	if perMinuteParam.HasBeenSet() {
		return 60 / perMinute
	}
	ps.Help("Program error: the setting is not being handled: " +
		timeSpecParamCounter.SetBy())
	return 0
}

func main() {
	ps := paramset.NewOrDie(addParams,
		addTimeSpecParams,
		addWakeupActionParams,
		// verbose.AddParams,
		param.SetProgramDescription(
			"This will sleep until a given time.\n"+
				"You can specify the interval you want it to sleep for and"+
				" rather than sleeping for that period it will sleep until"+
				" the next round interval."+
				" So for instance you could choose to sleep until the next"+
				" hour and it will wake up at minute 00 rather than"+
				" 60 minutes later"))
	ps.Parse()

	if timeSpecParamCounter.Count() > 1 {
		ps.Help("only one time specification should be given",
			"it has been set at: ",
			"    "+timeSpecParamCounter.SetBy())
	}

	if absTimeParam.HasBeenSet() {
		sleepToAbsTime(ps)
	} else {
		var durationSecs = durationSecs(ps)

		for {
			now := time.Now()
			if useUTC {
				now = now.UTC()
			}

			sleepToTarget(now, sleepCalc(durationSecs, now), ps)
			if msgParam.HasBeenSet() {
				fmt.Println(msg)
			}
			runShellCmd(ps)

			if finished() {
				break
			}
		}
	}
}

// finished returns true if the repeats of the sleep are complete, false
// otherwise
func finished() bool {
	if !repeat {
		return true
	}

	if repeatCount > 0 {
		repeatCount--
		if repeatCount <= 0 {
			repeat = false
			return true
		}
	}
	return false
}

// sleepToTarget sleeps for the specified duration
func sleepToTarget(now time.Time, sleepFor time.Duration, ps *param.PSet) {
	targetTime := now.Add(sleepFor)

	// if verbose.IsOn() {
	// 	format := "15:04:05.000"
	// 	verbose.Println("sleeping for:", sleepFor.String())
	// 	verbose.Println("        from:", now.Format(format))
	// 	verbose.Println("       until:", targetTime.Format(format))
	// }

	if doSleep {
		time.Sleep(sleepFor)
	}

	if showTime {
		fmt.Println(targetTime.Format(showTimeFmt))
	}
}
