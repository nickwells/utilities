package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nickwells/param.mod/v6/param"
)

// Created: Sun Oct 22 11:17:41 2017

const (
	tsNow = iota
	tsDateTimeStr
	tsTimeStr
)

// Prog holds program parameters and status
type Prog struct {
	fromZone *time.Location
	toZone   *time.Location

	inFormat  string
	outFormat string

	useUSDateOrder bool
	noSecs         bool
	noCentury      bool
	showTimezone   bool
	showAMPM       bool
	showMonthName  bool

	datePartSep string
	dtStr       string
	tStr        string
	timeSource  int

	fromZoneParam *param.ByName
	dtParam       *param.ByName
	tParam        *param.ByName
}

// NewProg returns a new Prog instance with the default values set
func NewProg() *Prog {
	return &Prog{
		fromZone:   time.Local,
		toZone:     time.Local,
		inFormat:   dateFmt + " " + timeFmt,
		outFormat:  dateFmt + " " + timeFmt,
		timeSource: tsNow,
	}
}

// getTime returns the time according to the parameters given
func (prog *Prog) getTime() time.Time {
	switch prog.timeSource {
	case tsNow:
		return time.Now()
	case tsDateTimeStr:
		tIn, err := time.ParseInLocation(
			prog.inFormat, prog.dtStr, prog.fromZone)
		if err != nil {
			fmt.Println("Cannot parse the date and time:", err)
			os.Exit(1)
		}
		return tIn
	case tsTimeStr:
		dtStr := time.Now().In(prog.fromZone).Format(dateFmt) + " " + prog.tStr
		tIn, err := time.ParseInLocation(prog.inFormat, dtStr, prog.fromZone)
		if err != nil {
			fmt.Println("Cannot parse the time:", err)
			os.Exit(1)
		}
		return tIn
	}

	fmt.Println("Unknown time source:", prog.timeSource)
	os.Exit(1)
	return time.Time{}
}

func main() {
	prog := NewProg()
	ps := makeParamSet(prog)
	ps.Parse()

	tIn := prog.getTime()

	tOut := tIn.In(prog.toZone)
	fmt.Println(tOut.Format(prog.outFormat))
}
