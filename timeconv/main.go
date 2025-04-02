package main

import (
	"fmt"
	"os"
)

// Created: Sun Oct 22 11:17:41 2017

func main() {
	prog := newProg()
	ps := makeParamSet(prog)
	ps.Parse()

	if prog.listTZNames {
		prog.listTimezoneNames()
		os.Exit(0)
	}

	tIn := prog.getTime()
	tOut := tIn.In(prog.toZone)
	fmt.Println(tOut.Format(prog.outFormat))
}
