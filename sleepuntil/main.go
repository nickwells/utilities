package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/verbose.mod/verbose"
)

var showTime bool

const dfltShowTimeFmt = "20060102.150405"

var showTimeFmt = dfltShowTimeFmt

var useUTC bool
var doSleep = true

var repeat bool
var repeatCount int64 = -1

var afterSleepCmd string
var msg string

var absTime time.Time

var offset int64
var offsetMins int64

var timeMins int64
var timeSecs int64
var perDay int64
var perHour int64
var perMinute int64

// sleepCalc calculates the time to sleep
func sleepCalc(durationSecs, offsetSecs int64, now time.Time) time.Duration {
	s := int64(now.Second())
	s += int64(now.Minute()) * 60
	s += int64(now.Hour()) * 3600
	s *= 1e9
	s += int64(now.Nanosecond())

	durationNano := durationSecs * 1e9

	offsetNormalised := offsetSecs % durationSecs
	offsetNano := offsetNormalised * 1e9

	var sleepNano int64

	remainder := (s % durationNano)
	remainder -= offsetNano
	if remainder < 0 {
		remainder += durationNano
	}
	remainder %= durationNano
	if remainder != 0 {
		sleepNano = durationNano - remainder
	}

	return time.Duration(sleepNano) * time.Nanosecond
}

// sleepToAbsTime parses the value of the absTime into a time in the given
// timezone (Local if no value is given) and sleeps until that time
func sleepToAbsTime() {
	now := time.Now()
	if useUTC {
		now = now.UTC()
	}

	dur := absTime.Sub(now)
	if dur > 0 {
		sleepToTarget(now, dur)
	}
}

// runShellCmd will run the given command, if any, in a subshell. it will
// check for errors and report them; it exits on any error
func runShellCmd() {
	if len(afterSleepCmd) > 0 {
		out, err := exec.Command("/bin/bash", "-c",
			afterSleepCmd).CombinedOutput()
		fmt.Print(string(out))
		if err != nil {
			fmt.Println("Command failed:", err)
			os.Exit(1)
		}
	}
}

func calcDurationSecs() int64 {
	if timeSecs > 0 {
		return timeSecs
	}
	if timeMins > 0 {
		return timeMins * 60
	}
	if perDay > 0 {
		return 24 * 60 * 60 / perDay
	}
	if perHour > 0 {
		return 60 * 60 / perHour
	}
	if perMinute > 0 {
		return 60 / perMinute
	}

	fmt.Println("Program error: the sleep setting is not being handled: " +
		timeParamCounter.SetBy())
	os.Exit(1)

	return 0
}

func main() {
	ps := paramset.NewOrDie(
		addParams,
		addTimeParams,
		addActionParams,
		verbose.AddParams,
		addExamples,
		param.SetProgramDescription(
			"This will sleep until a given time and then perform the"+
				" chosen actions."+
				"\n\n"+
				"You can specify either a particular time of day to sleep"+
				" until or some fragment of the day or some regular"+
				" period (which must divide the day into a whole number"+
				" of parts)."+
				"\n\n"+
				" So for instance you could choose to sleep until the next"+
				" hour and it will wake up at minute 00 rather than"+
				" 60 minutes later."+
				"\n\n"+
				"You can give an offset to the regular time and the delay"+
				" will be adjusted accordingly."))
	ps.Parse()

	if !absTime.IsZero() {
		sleepToAbsTime()
		action()
	} else {
		var durationSecs = calcDurationSecs()

		for {
			now := time.Now()
			if useUTC {
				now = now.UTC()
			}

			sleepToTarget(now,
				sleepCalc(durationSecs, offset+(offsetMins*60), now))
			action()

			if finished() {
				break
			}
		}
	}
}

// action will perform the actions that should happen after waking up from
// the sleep
func action() {
	if len(msg) > 0 {
		fmt.Println(msg)
	}
	runShellCmd()
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
func sleepToTarget(now time.Time, sleepFor time.Duration) {
	if verbose.IsOn() {
		format := "15:04:05.000000"
		verbose.Println("sleeping for: ", sleepFor.String())
		verbose.Println("        from: ", now.Format(format))
		verbose.Println("       until: ", now.Add(sleepFor).Format(format))
	}

	if doSleep {
		time.Sleep(sleepFor)
	}

	if showTime {
		fmt.Println(now.Add(sleepFor).Format(showTimeFmt))
	}
}
