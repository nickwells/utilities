package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nickwells/col.mod/v3/col"
	"github.com/nickwells/col.mod/v3/col/colfmt"
	"github.com/nickwells/param.mod/v5/param/psetter"
	"github.com/nickwells/units.mod/v2/units"

	"golang.org/x/sys/unix"
)

// Created: Wed Jan 31 23:10:36 2018

const (
	nameStr     = "name"
	fSpStr      = "free"
	avSpStr     = "avail"
	totSpStr    = "total"
	usedSpStr   = "used"
	fileCntStr  = "totalFiles"
	freeFCntStr = "freeFiles"
)

// valFunc is the type of a fieldVal function in the fieldInfo struct
type valFunc func(name string, s *unix.Statfs_t) any

// fieldInfo records details about each field
type fieldInfo struct {
	fieldVal valFunc
	format   func() string
	shortFmt func() string
	col      func(int) *col.Col
}

// Prog holds program parameters and status
type Prog struct {
	dataFamily    *units.Family
	baseUnit      units.Unit
	displayUnits  units.Unit
	fiMap         map[string]fieldInfo
	allowedFields psetter.AllowedVals
}

// NewProg returns a new Prog instance with the default values set
func NewProg() *Prog {
	prog := &Prog{
		dataFamily: units.GetFamilyOrPanic(units.Data),
	}
	prog.baseUnit = prog.dataFamily.GetUnitOrPanic(
		prog.dataFamily.BaseUnitName())
	prog.displayUnits = prog.baseUnit

	prog.fiMap = map[string]fieldInfo{
		nameStr: {
			fieldVal: func(name string, s *unix.Statfs_t) any {
				return name
			},
			format:   func() string { return "%s" },
			shortFmt: func() string { return "%s" },
			col: func(w int) *col.Col {
				return col.New(colfmt.String{W: w}, "Name")
			},
		},
		fSpStr: {
			fieldVal: func(name string, s *unix.Statfs_t) any {
				vu := units.ValUnit{
					U: prog.baseUnit,
					V: float64(s.Bfree * uint64(s.Bsize)),
				}
				return vu.ConvertOrPanic(prog.displayUnits).V
			},
			format: func() string {
				return "%.0f " + prog.displayUnits.NamePlural()
			},
			shortFmt: func() string { return "%.0f" },
			col: func(_ int) *col.Col {
				units := "Units: " + prog.displayUnits.Name()
				return col.New(&colfmt.Float{W: 15}, units, "space", "free")
			},
		},
		avSpStr: {
			fieldVal: func(name string, s *unix.Statfs_t) any {
				vu := units.ValUnit{
					U: prog.baseUnit,
					V: float64(s.Bavail * uint64(s.Bsize)),
				}
				return vu.ConvertOrPanic(prog.displayUnits).V
			},
			format: func() string {
				return "%.0f " + prog.displayUnits.NamePlural()
			},
			shortFmt: func() string { return "%.0f" },
			col: func(_ int) *col.Col {
				units := "Units: " + prog.displayUnits.Name()
				return col.New(&colfmt.Float{W: 15}, units, "space", "available")
			},
		},
		totSpStr: {
			fieldVal: func(name string, s *unix.Statfs_t) any {
				vu := units.ValUnit{
					U: prog.baseUnit,
					V: float64(s.Blocks * uint64(s.Bsize)),
				}
				return vu.ConvertOrPanic(prog.displayUnits).V
			},
			format: func() string {
				return "%.0f " + prog.displayUnits.NamePlural()
			},
			shortFmt: func() string { return "%.0f" },
			col: func(_ int) *col.Col {
				units := "Units: " + prog.displayUnits.Name()
				return col.New(&colfmt.Float{W: 15}, units, "space", "total")
			},
		},
		usedSpStr: {
			fieldVal: func(name string, s *unix.Statfs_t) any {
				vu := units.ValUnit{
					U: prog.baseUnit,
					V: float64((s.Blocks - s.Bfree) * uint64(s.Bsize)),
				}
				return vu.ConvertOrPanic(prog.displayUnits).V
			},
			format: func() string {
				return "%.0f " + prog.displayUnits.NamePlural()
			},
			shortFmt: func() string { return "%.0f" },
			col: func(_ int) *col.Col {
				units := "Units: " + prog.displayUnits.Name()
				return col.New(&colfmt.Float{W: 15}, units, "space", "used")
			},
		},
		fileCntStr: {
			fieldVal: func(name string, s *unix.Statfs_t) any {
				return s.Files
			},
			format:   func() string { return "%d" },
			shortFmt: func() string { return "%d" },
			col: func(_ int) *col.Col {
				return col.New(&colfmt.Int{W: 12}, "files", "available")
			},
		},
		freeFCntStr: {
			fieldVal: func(name string, s *unix.Statfs_t) any {
				return s.Ffree
			},
			format:   func() string { return "%d" },
			shortFmt: func() string { return "%d" },
			col: func(_ int) *col.Col {
				return col.New(&colfmt.Int{W: 12}, "files", "remaining")
			},
		},
	}
	prog.allowedFields = psetter.AllowedVals{
		nameStr:     "the name of the directory",
		fSpStr:      "the total free space available",
		avSpStr:     "the space available to you",
		totSpStr:    "the total disk space on the filesystem",
		usedSpStr:   "the amount of disk space used",
		fileCntStr:  "the number of files on the filesystem",
		freeFCntStr: "the number of files that can still be created",
	}

	prog.addAllowedFields()
	prog.addFieldInfo()

	return prog
}

var fields = []string{
	nameStr,
	avSpStr,
}

var (
	showAsTable bool
	noLabel     bool
)

func (prog *Prog) makeReport(dirs ...string) *col.Report {
	var maxDirNameLen int
	for _, d := range dirs {
		if len(d) > maxDirNameLen {
			maxDirNameLen = len(d)
		}
	}

	cols := make([]*col.Col, 0, len(fields))
	h, err := col.NewHeader()
	if err != nil {
		log.Fatal("couldn't create the table header: ", err)
	}
	if noLabel {
		err = col.HdrOptDontPrint(h)
		if err != nil {
			log.Fatal("couldn't turn off the table header: ", err)
		}
	}

	for _, f := range fields {
		fi := prog.getFieldInfo(f)
		cols = append(cols, fi.col(maxDirNameLen))
	}

	return col.NewReport(h, os.Stdout, cols[0], cols[1:]...)
}

func getStat(dirName string) unix.Statfs_t {
	var s unix.Statfs_t
	err := unix.Statfs(dirName, &s)
	if err != nil {
		log.Fatal("Couldn't stat ", dirName, " Err: ", err, "\n")
	}
	return s
}

func (prog *Prog) reportStatAsTable(rpt *col.Report, dirName string, s unix.Statfs_t) {
	reportArgs := make([]any, 0, len(fields))
	for _, f := range fields {
		fi := prog.getFieldInfo(f)
		reportArgs = append(reportArgs, fi.fieldVal(dirName, &s))
	}
	err := rpt.PrintRow(reportArgs...)
	if err != nil {
		log.Fatal("Couldn't print Row: ", err)
	}
}

func (prog *Prog) reportStat(dirName string, s unix.Statfs_t) {
	for _, f := range fields {
		if !noLabel {
			fmt.Print(f, ": ")
		}
		fi := prog.getFieldInfo(f)
		format := fi.format()
		if noLabel {
			format = fi.shortFmt()
		}
		fmt.Printf(format, fi.fieldVal(dirName, &s))
		fmt.Println()
	}
}

// getFieldInfo returns the fieldInfo entry corresponding to the field name.
// It will report an error and exit if the field name is not found in the map
func (prog *Prog) getFieldInfo(f string) fieldInfo {
	fi, ok := prog.fiMap[f]
	if !ok {
		log.Fatal("internal error: unknown field: ", f)
	}
	return fi
}

func main() {

	prog := NewProg()
	ps := makeParamSet(prog)
	ps.Parse()

	dirs := ps.Remainder()
	if len(dirs) == 0 {
		dirs = append(dirs, ".")
	}

	rpt := prog.makeReport(dirs...)
	for _, dirName := range dirs {
		s := getStat(dirName)

		if showAsTable {
			prog.reportStatAsTable(rpt, dirName, s)
		} else {
			prog.reportStat(dirName, s)
		}
	}
}
