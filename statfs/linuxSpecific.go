//go:build linux

package main

import (
	"github.com/nickwells/col.mod/v3/col"
	"github.com/nickwells/col.mod/v3/col/colfmt"

	"golang.org/x/sys/unix"
)

const (
	maxNameStr = "maxNameLen"
	flagsStr   = "flags"
)

// addAllowedFields adds the extra Linux-specific allowed fields
func addAllowedFields() {
	allowedFields[maxNameStr] = "the maximum length of filenames"
	allowedFields[flagsStr] = "show the mount flags"
}

// addFieldInfo adds the extra Linux-specific field info
func addFieldInfo() {
	const (
		maxFlagsLen = 30
	)

	mountFlags := map[int64]string{
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

	fiMap[maxNameStr] = fieldInfo{
		fieldVal: func(name string, s *unix.Statfs_t) any {
			return s.Namelen
		},
		format:   func() string { return "%d" },
		shortFmt: func() string { return "%d" },
		col: func(_ int) *col.Col {
			return col.New(&colfmt.Int{W: 4}, "max file", "name length")
		},
	}
	fiMap[flagsStr] = fieldInfo{
		fieldVal: func(name string, s *unix.Statfs_t) any {
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
	}
}