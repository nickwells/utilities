package main

import (
	"fmt"

	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
)

const origExt = ".orig"

// fileProvisos records the checks to be carried out on the files
var fileProvisos = filecheck.FileExists()

// origFileProvisos records the checks to be carried out on the files with
// extension '.orig'
var origFileProvisos = filecheck.IsNew()

// HandleRemainder processes the trailing parameters . If gosh has the
// 'runInReadLoop' flag set then they are treated as files and added to the
// filesToRead. Otherwise they are added to the list of args and that is
// looped over instead.
func (g *Gosh) HandleRemainder(ps *param.PSet, _ *location.L) {
	if g.runInReadLoop {
		g.populateFilesToRead(ps.Remainder())
	} else {
		g.populateArgs(ps.Remainder())
	}
}

// populateFilesToRead will populate the filenames in the filesToRead value
// in the Gosh struct and record any errors found.
//
// It will first check that there are no duplicate filess, that they all
// exist, that they are all files, that there are no existing files with the
// same name plus the '.orig' extension. If any of these conditions is not
// met it will report the error, add it to the ErrMap and return.

func (g *Gosh) populateFilesToRead(names []string) {
	goodNames := make([]string, 0, len(names))
	dupMap := make(map[string]int)

	for i, name := range names {
		if firstIdx, exists := dupMap[name]; exists {
			g.addError("duplicate filename",
				fmt.Errorf(
					"filename %q has been given multiple times,"+
						" first at %d and again at %d",
					name, firstIdx, i))
			continue
		}
		dupMap[name] = i

		if err := fileProvisos.StatusCheck(name); err != nil {
			g.addError("file check", err)
			continue
		}

		if g.inPlaceEdit {
			if err := origFileProvisos.StatusCheck(name + origExt); err != nil {
				g.addError("original file check", err)
				continue
			}
		}

		goodNames = append(goodNames, fmt.Sprintf("%q", name))
	}
	g.filesToRead = goodNames
}

// populateArgs will copy the remainder params into the args list on the Gosh
// struct. It will quote them as they are copied.
func (g *Gosh) populateArgs(names []string) {
	for _, name := range names {
		g.args = append(g.args, fmt.Sprintf("%q", name))
	}
}
