// statfs
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/col.mod/v2/col"
	"github.com/nickwells/col.mod/v2/col/colfmt"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/param.mod/v5/param/psetter"
	"github.com/nickwells/units.mod/units"
	"github.com/nickwells/unitsetter.mod/v3/unitsetter"

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
	maxNameStr  = "maxNameLen"
	flagsStr    = "flags"
)

const (
	maxFlagsLen = 30
)

var mountFlags = map[int64]string{
	unix.MS_MANDLOCK:    "mandatory locking permitted",
	unix.MS_NOATIME:     "access times not updated",
	unix.MS_NODEV:       "no device special file access",
	unix.MS_NODIRATIME:  "directory access times not updated",
	unix.MS_NOEXEC:      "program execution disallowed",
	unix.MS_NOSUID:      "set-user/group-id bits ignored",
	unix.MS_RDONLY:      "mounted readonly",
	unix.MS_RELATIME:    "atime is relative to mtime/ctime",
	unix.MS_SYNCHRONOUS: "writes are synched immediately",
}

// valFunc is the type of a fieldVal function in the fieldInfo struct
type valFunc func(name string, s *unix.Statfs_t) interface{}

// fieldInfo records details about each field
type fieldInfo struct {
	fieldVal valFunc
	format   func() string
	shortFmt func() string
	col      func(int) *col.Col
}

var fiMap = map[string]fieldInfo{
	nameStr: {
		fieldVal: func(name string, s *unix.Statfs_t) interface{} {
			return name
		},
		format:   func() string { return "%s" },
		shortFmt: func() string { return "%s" },
		col: func(w int) *col.Col {
			return col.New(colfmt.String{W: w},
				"Name")
		},
	},
	fSpStr: {
		fieldVal: func(name string, s *unix.Statfs_t) interface{} {
			f, err := units.ConvertFromBaseUnits(
				float64(s.Bfree*uint64(s.Bsize)),
				mult)
			if err != nil {
				return float64(0.0)
			}
			return f
		},
		format:   func() string { return "%.0f " + mult.NamePlural },
		shortFmt: func() string { return "%.0f" },
		col: func(_ int) *col.Col {
			units := "Units: " + mult.Name
			return col.New(&colfmt.Float{W: 15}, units, "space", "free")
		},
	},
	avSpStr: {
		fieldVal: func(name string, s *unix.Statfs_t) interface{} {
			f, err := units.ConvertFromBaseUnits(
				float64(s.Bavail*uint64(s.Bsize)),
				mult)
			if err != nil {
				return float64(0.0)
			}
			return f
		},
		format:   func() string { return "%.0f " + mult.NamePlural },
		shortFmt: func() string { return "%.0f" },
		col: func(_ int) *col.Col {
			units := "Units: " + mult.Name
			return col.New(&colfmt.Float{W: 15}, units, "space", "available")
		},
	},
	totSpStr: {
		fieldVal: func(name string, s *unix.Statfs_t) interface{} {
			f, err := units.ConvertFromBaseUnits(
				float64(s.Blocks*uint64(s.Bsize)),
				mult)
			if err != nil {
				return float64(0.0)
			}
			return f
		},
		format:   func() string { return "%.0f " + mult.NamePlural },
		shortFmt: func() string { return "%.0f" },
		col: func(_ int) *col.Col {
			units := "Units: " + mult.Name
			return col.New(&colfmt.Float{W: 15}, units, "space", "total")
		},
	},
	usedSpStr: {
		fieldVal: func(name string, s *unix.Statfs_t) interface{} {
			f, err := units.ConvertFromBaseUnits(
				float64((s.Blocks-s.Bfree)*uint64(s.Bsize)),
				mult)
			if err != nil {
				return float64(0.0)
			}
			return f
		},
		format:   func() string { return "%.0f " + mult.NamePlural },
		shortFmt: func() string { return "%.0f" },
		col: func(_ int) *col.Col {
			units := "Units: " + mult.Name
			return col.New(&colfmt.Float{W: 15}, units, "space", "used")
		},
	},
	fileCntStr: {
		fieldVal: func(name string, s *unix.Statfs_t) interface{} {
			return s.Files
		},
		format:   func() string { return "%d" },
		shortFmt: func() string { return "%d" },
		col: func(_ int) *col.Col {
			return col.New(&colfmt.Int{W: 12}, "files", "available")
		},
	},
	freeFCntStr: {
		fieldVal: func(name string, s *unix.Statfs_t) interface{} {
			return s.Ffree
		},
		format:   func() string { return "%d" },
		shortFmt: func() string { return "%d" },
		col: func(_ int) *col.Col {
			return col.New(&colfmt.Int{W: 12}, "files", "remaining")
		},
	},
	maxNameStr: {
		fieldVal: func(name string, s *unix.Statfs_t) interface{} {
			return s.Namelen
		},
		format:   func() string { return "%d" },
		shortFmt: func() string { return "%d" },
		col: func(_ int) *col.Col {
			return col.New(&colfmt.Int{W: 4}, "max file", "name length")
		},
	},
	flagsStr: {
		fieldVal: func(name string, s *unix.Statfs_t) interface{} {
			rval := ""
			sep := ""
			for f, flagName := range mountFlags {
				if (s.Flags & f) != 0 {
					rval += sep + flagName
					sep = ", "
				}
			}
			return rval
		},
		format:   func() string { return "%s" },
		shortFmt: func() string { return "%s" },
		col: func(_ int) *col.Col {
			return col.New(colfmt.String{W: maxFlagsLen}, "FS", "flags")
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
	maxNameStr:  "the maximum length of filenames",
	flagsStr:    "show the mount flags",
}
var fields = []string{
	nameStr,
	avSpStr,
}

var mult = units.ByteUnit
var showAsTable bool
var noLabel bool

func addParams(ps *param.PSet) error {
	unitDetails, err := units.GetUnitDetails(units.Data)
	if err != nil {
		return err
	}
	ps.Add("units",
		unitsetter.UnitSetter{
			Value: &mult,
			UD:    unitDetails,
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
				check.StringSliceNoDups,
				check.StringSliceLenGT(0),
			},
		},
		"choose which information to show about the file system")

	err = ps.SetRemHandler(param.NullRemHandler{}) // allow trailing arguments
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

	var cols = make([]*col.Col, 0, len(fields))
	var rpt *col.Report
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

	rpt, err = col.NewReport(h, os.Stdout, cols...)
	if err != nil {
		log.Fatal("couldn't create the table report: ", err)
	}
	return rpt
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
	reportArgs := make([]interface{}, 0, len(fields))
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
	ps := paramset.NewOrDie(addParams)
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
