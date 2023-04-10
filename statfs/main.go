package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/col.mod/v3/col"
	"github.com/nickwells/col.mod/v3/col/colfmt"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/param.mod/v5/param/psetter"
	"github.com/nickwells/units.mod/v2/units"
	"github.com/nickwells/unitsetter.mod/v4/unitsetter"
	"github.com/nickwells/versionparams.mod/versionparams"

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

var (
	dataFamily   = units.GetFamilyOrPanic(units.Data)
	baseUnit     = dataFamily.GetUnitOrPanic(dataFamily.BaseUnitName())
	displayUnits = baseUnit
)

var fiMap = map[string]fieldInfo{
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
				U: baseUnit,
				V: float64(s.Bfree * uint64(s.Bsize)),
			}
			return vu.ConvertOrPanic(displayUnits).V
		},
		format:   func() string { return "%.0f " + displayUnits.NamePlural() },
		shortFmt: func() string { return "%.0f" },
		col: func(_ int) *col.Col {
			units := "Units: " + displayUnits.Name()
			return col.New(&colfmt.Float{W: 15}, units, "space", "free")
		},
	},
	avSpStr: {
		fieldVal: func(name string, s *unix.Statfs_t) any {
			vu := units.ValUnit{
				U: baseUnit,
				V: float64(s.Bavail * uint64(s.Bsize)),
			}
			return vu.ConvertOrPanic(displayUnits).V
		},
		format:   func() string { return "%.0f " + displayUnits.NamePlural() },
		shortFmt: func() string { return "%.0f" },
		col: func(_ int) *col.Col {
			units := "Units: " + displayUnits.Name()
			return col.New(&colfmt.Float{W: 15}, units, "space", "available")
		},
	},
	totSpStr: {
		fieldVal: func(name string, s *unix.Statfs_t) any {
			vu := units.ValUnit{
				U: baseUnit,
				V: float64(s.Blocks * uint64(s.Bsize)),
			}
			return vu.ConvertOrPanic(displayUnits).V
		},
		format:   func() string { return "%.0f " + displayUnits.NamePlural() },
		shortFmt: func() string { return "%.0f" },
		col: func(_ int) *col.Col {
			units := "Units: " + displayUnits.Name()
			return col.New(&colfmt.Float{W: 15}, units, "space", "total")
		},
	},
	usedSpStr: {
		fieldVal: func(name string, s *unix.Statfs_t) any {
			vu := units.ValUnit{
				U: baseUnit,
				V: float64((s.Blocks - s.Bfree) * uint64(s.Bsize)),
			}
			return vu.ConvertOrPanic(displayUnits).V
		},
		format:   func() string { return "%.0f " + displayUnits.NamePlural() },
		shortFmt: func() string { return "%.0f" },
		col: func(_ int) *col.Col {
			units := "Units: " + displayUnits.Name()
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

var allowedFields = psetter.AllowedVals{
	nameStr:     "the name of the directory",
	fSpStr:      "the total free space available",
	avSpStr:     "the space available to you",
	totSpStr:    "the total disk space on the filesystem",
	usedSpStr:   "the amount of disk space used",
	fileCntStr:  "the number of files on the filesystem",
	freeFCntStr: "the number of files that can still be created",
}

var fields = []string{
	nameStr,
	avSpStr,
}

var (
	showAsTable bool
	noLabel     bool
)

func addParams(ps *param.PSet) error {
	ps.Add("units",
		unitsetter.UnitSetter{
			Value: &displayUnits,
			F:     dataFamily,
		},
		"set the units in which to display the results")

	ps.Add("no-label",
		psetter.Bool{
			Value: &noLabel,
		},
		"show the results without labels")

	ps.Add("table",
		psetter.Bool{
			Value: &showAsTable,
		},
		"show the results in a table rather than on a line")

	ps.Add("show",
		psetter.EnumList{
			Value:       &fields,
			AllowedVals: allowedFields,
			Checks: []check.StringSlice{
				check.SliceHasNoDups[[]string, string],
				check.SliceLength[[]string](check.ValGT(0)),
			},
		},
		"choose which information to show about the file system")

	err := ps.SetRemHandler(param.NullRemHandler{}) // allow trailing arguments
	if err != nil {
		return err
	}

	return nil
}

func makeReport(dirs ...string) *col.Report {
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
		fi := getFieldInfo(f)
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

func reportStatAsTable(rpt *col.Report, dirName string, s unix.Statfs_t) {
	reportArgs := make([]any, 0, len(fields))
	for _, f := range fields {
		fi := getFieldInfo(f)
		reportArgs = append(reportArgs, fi.fieldVal(dirName, &s))
	}
	err := rpt.PrintRow(reportArgs...)
	if err != nil {
		log.Fatal("Couldn't print Row: ", err)
	}
}

func reportStat(dirName string, s unix.Statfs_t) {
	for _, f := range fields {
		if !noLabel {
			fmt.Print(f, ": ")
		}
		fi := getFieldInfo(f)
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
func getFieldInfo(f string) fieldInfo {
	fi, ok := fiMap[f]
	if !ok {
		log.Fatal("internal error: unknown field: ", f)
	}
	return fi
}

func main() {
	addAllowedFields()
	addFieldInfo()

	ps := paramset.NewOrDie(
		versionparams.AddParams,

		addParams)
	ps.SetProgramDescription("Report on the status of file systems.\n\n" +
		"By default the file system to be reported will be that of the" +
		" current directory '.' but you can specify a list of alternative" +
		" directories by passing them after the terminating parameter" +
		" ('" + ps.TerminalParam() + "'). The value reported will be" +
		" the available space.")
	ps.Parse()

	dirs := ps.Remainder()
	if len(dirs) == 0 {
		dirs = append(dirs, ".")
	}

	rpt := makeReport(dirs...)
	for _, dirName := range dirs {
		s := getStat(dirName)

		if showAsTable {
			reportStatAsTable(rpt, dirName, s)
		} else {
			reportStat(dirName, s)
		}
	}
}
