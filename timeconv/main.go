package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// Created: Sun Oct 22 11:17:41 2017

func main() {
	ps := paramset.NewOrDie(
		versionparams.AddParams,

		addParams,

		param.SetProgramDescription(
			"this will convert the passed date into the equivalent time"+
				" in the given timezone. If no 'from' timezone is given"+
				" the local timezone is used. Similarly for the 'to'"+
				" timezone. If no time or date is given then the current"+
				" time is used. Only one of the time or date can be"+
				" given. A time or date must be given if the 'from'"+
				" timezone is given."),
	)
	ps.Parse()

	tIn := time.Now()

	if dtParam.HasBeenSet() {
		var err error
		tIn, err = time.ParseInLocation(inFormat, dtStr, fromZone)
		if err != nil {
			fmt.Println("Cannot parse the date and time:", err)
			os.Exit(1)
		}
	} else if tParam.HasBeenSet() {
		nowDateStr := time.Now().In(fromZone).Format(dateFmt)
		var err error
		tIn, err = time.ParseInLocation(inFormat, nowDateStr+" "+tStr, fromZone)
		if err != nil {
			fmt.Println("Cannot parse the time:", err)
			os.Exit(1)
		}
	}

	tOut := tIn.In(toZone)
	fmt.Println(tOut.Format(outFormat))
}
