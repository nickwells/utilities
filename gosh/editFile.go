package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nickwells/english.mod/english"
	"github.com/nickwells/verbose.mod/verbose"
)

// setEditor sets the script editor to be used. If the editor is set but
// cannot be found in the execution path then an error is added to the error
// map.
func (g *gosh) setEditor() {
	if !g.edit {
		return
	}

	editors := []struct {
		editor string
		source string
	}{
		{g.editorParam, "the '" + paramNameScriptEditor + "' parameter"},
		{os.Getenv(envVisual), "the '" + envVisual + "' environment variable"},
		{os.Getenv(envEditor), "the '" + envEditor + "' environment variable"},
	}

	for _, trialEditor := range editors {
		editor := strings.TrimSpace(trialEditor.editor)
		if editor == "" {
			continue
		}

		var err error
		if _, err = exec.LookPath(editor); err == nil {
			g.editor = editor
			return
		}

		parts := strings.Fields(editor)
		if parts[0] == editor {
			g.addError("bad editor",
				fmt.Errorf("cannot find %s (source: %s): %w",
					editor, trialEditor.source, err))

			continue
		}

		editor = parts[0]
		if _, err = exec.LookPath(editor); err == nil {
			g.editor = editor
			g.editorArgs = parts[1:]

			return
		}

		g.addError("bad editor",
			fmt.Errorf("cannot find %s (source: %s): %w",
				editor, trialEditor.source, err))

		continue
	}

	sources := make([]string, 0, len(editors))

	for _, e := range editors {
		sources = append(sources, e.source)
	}

	intro := "    "

	g.addError("no editor",
		errors.New("no editor has been given."+
			" Possible sources are:\n"+intro+
			english.Join(sources, ",\n"+intro, "\n or ")+
			",\nin that order."))
}

// editGoFile starts an editor to edit the program
func (g *gosh) editGoFile() {
	if !g.edit {
		return
	}

	defer g.dbgStack.Start("editGoFile", "editing the program")()
	intro := g.dbgStack.Tag()

	args := g.editorArgs
	args = append(args, filepath.Join(g.goshDir, goshFilename))
	verbose.Println(intro, " Command: "+g.editor+" "+strings.Join(args, " "))
	cmd := exec.Command(g.editor, args...) //nolint:gosec
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	g.reportFatalError("run the editor",
		cmd.Path+" "+strings.Join(cmd.Args, " "),
		err)
}
