package main

import (
	"fmt"
	"path/filepath"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
)

const origExt = ".orig"

// fileProvisos records the checks to be carried out on the files
var fileProvisos = filecheck.Provisos{
	Existence: filecheck.MustExist,
	Checks: []check.FileInfo{
		check.FileInfoIsRegular,
	},
}

// origFileProvisos records the checks to be carried out on the files with
// extension '.orig'
var origFileProvisos = filecheck.Provisos{Existence: filecheck.MustNotExist}

// AddErr adds the error to the named error map entry
func (g *Gosh) AddErr(name string, err error) {
	g.filesErrMap[name] = append(g.filesErrMap[name], err)
}

// HandleRemainder processes the filenames. It will populate the filenames in
// the fileNameHandler and record any errors found.
//
// It will first convert any file names that are not absolute into names
// based at the current working directory. Then it will check that there are
// no duplicates, that they all exist, that they are all files, that there
// are no existing files with the same name plus the '.orig' extension. If
// any of these conditions is not met it will report the error, add it to the
// ErrMap and return
func (g *Gosh) HandleRemainder(ps *param.PSet, _ *location.L) {
	names := ps.Remainder()
	goodNames := make([]string, 0, len(names))
	dupMap := make(map[string]int)

	for i, name := range names {
		if !filepath.IsAbs(name) {
			name = filepath.Join(g.cwd, name)
		}

		if firstIdx, exists := dupMap[name]; exists {
			g.AddErr("duplicate filename",
				fmt.Errorf(
					"filename %q has been given multiple times,"+
						" first at %d and again at %d",
					name, firstIdx, i))
			continue
		}
		dupMap[name] = i

		if err := fileProvisos.StatusCheck(name); err != nil {
			g.AddErr("file check", err)
			continue
		}

		if err := origFileProvisos.StatusCheck(name + origExt); err != nil {
			g.AddErr("original file check", err)
			continue
		}

		goodNames = append(goodNames, fmt.Sprintf("%q", name))
	}
	g.filesToRead = goodNames
}
