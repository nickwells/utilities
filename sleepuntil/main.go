package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/nickwells/verbose.mod/verbose"
)

const dfltShowTimeFmt = "20060102.150405"

// Prog holds program parameters and status
type Prog struct {
	absTime time.Time

	showTimeFmt   string
	afterSleepCmd string
	msg           string

	showTime bool
	useUTC   bool
	doSleep  bool

	repeat bool

	repeatCount int64

	offset     int64
	offsetMins int64

	timeMins  int64
	timeSecs  int64
	perDay    int64
	perHour   int64
	perMinute int64
}

// NewProg returns a new Prog instance with the default values set

func NewProg() *Prog {
	return &Prog{
		showTimeFmt: dfltShowTimeFmt,
		doSleep:     true,
		repeatCount: -1,
	}
}

// sleepCalc calculates the time to sleep
func sleepCalc(durationSecs, offsetSecs int64, now time.Time) time.Duration {
	s := int64(now.Second())
	s += int64(now.Minute()) * secondsPerMinute
	s += int64(now.Hour()) * secondsPerHour
	s *= nanoSecondsPerSecond
	s += int64(now.Nanosecond())

	durationNano := durationSecs * nanoSecondsPerSecond

	offsetNormalised := offsetSecs % durationSecs
	offsetNano := offsetNormalised * nanoSecondsPerSecond

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
func (prog *Prog) sleepToAbsTime() {
	now := time.Now()
	if prog.useUTC {
		now = now.UTC()
	}

	dur := prog.absTime.Sub(now)
	if dur > 0 {
		prog.sleepToTarget(now, dur)
	}
}

// runShellCmd will run the given command, if any, in a subshell. it will
// check for errors and report them; it exits on any error
func (prog *Prog) runShellCmd() {
	if len(prog.afterSleepCmd) > 0 {
		out, err := exec.Command("/bin/bash", "-c", //nolint:gosec
			prog.afterSleepCmd).CombinedOutput()

		fmt.Print(string(out))

		if err != nil {
			fmt.Println("Command failed:", err)
			os.Exit(1)
		}
	}
}

func (prog *Prog) calcDurationSecs() int64 {
	if prog.timeSecs > 0 {
		return prog.timeSecs
	}

	if prog.timeMins > 0 {
		return prog.timeMins * secondsPerMinute
	}

	if prog.perDay > 0 {
		return secondsPerDay / prog.perDay
	}

	if prog.perHour > 0 {
		return secondsPerHour / prog.perHour
	}

	if prog.perMinute > 0 {
		return secondsPerMinute / prog.perMinute
	}

	return 0
}

func main() {
	prog := NewProg()
	ps := makeParamSet(prog)
	ps.Parse()

	if !prog.absTime.IsZero() {
		prog.sleepToAbsTime()
		prog.action()
	} else {
		durationSecs := prog.calcDurationSecs()

		for {
			now := time.Now()
			if prog.useUTC {
				now = now.UTC()
			}

			prog.sleepToTarget(now,
				sleepCalc(durationSecs,
					prog.offset+(prog.offsetMins*secondsPerMinute),
					now))
			prog.action()

			if prog.finished() {
				break
			}
		}
	}
}

// action will perform the actions that should happen after waking up from
// the sleep
func (prog *Prog) action() {
	if len(prog.msg) > 0 {
		fmt.Println(prog.msg)
	}

	prog.runShellCmd()
}

// finished returns true if the repeats of the sleep are complete, false
// otherwise
func (prog *Prog) finished() bool {
	if !prog.repeat {
		return true
	}

	if prog.repeatCount > 0 {
		prog.repeatCount--
		if prog.repeatCount <= 0 {
			prog.repeat = false
			return true
		}
	}

	return false
}

// sleepToTarget sleeps for the specified duration
func (prog *Prog) sleepToTarget(now time.Time, sleepFor time.Duration) {
	if verbose.IsOn() {
		format := "15:04:05.000000"

		verbose.Println("sleeping for: ", sleepFor.String())
		verbose.Println("        from: ", now.Format(format))
		verbose.Println("       until: ", now.Add(sleepFor).Format(format))
	}

	if prog.doSleep {
		time.Sleep(sleepFor)
	}

	if prog.showTime {
		fmt.Println(now.Add(sleepFor).Format(prog.showTimeFmt))
	}
}
