package main

import (
	"fmt"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/timer.mod/timer"
	"github.com/nickwells/verbose.mod/verbose"
)

// writeScript writes the contents of the named script. It panics if the
// script name is not found.
func (g *Gosh) writeScript(scriptName string) {
	script, ok := g.scripts[scriptName]
	if !ok {
		panic(fmt.Errorf("invalid script name: %q", scriptName))
	}

	for _, se := range script {
		lines, err := se.expand(g, se.value)
		if err != nil {
			g.addError("script: "+scriptName, err)
			continue
		}
		for _, s := range lines {
			g.print(s)
		}
	}
}

// writeGoFileImports writes the import statements into the Go file
func (g *Gosh) writeGoFileImports() {
	g.imports = append(g.imports, "os") // os.Chdir(...)

	if g.runInReadLoop {
		g.imports = append(g.imports, "bufio")
		if g.inPlaceEdit {
			g.imports = append(g.imports, "path/filepath")
		}
		if g.splitLine {
			g.imports = append(g.imports, "regexp")
		}
	}
	if g.runAsWebserver {
		g.imports = append(g.imports, "net/http")
	}

	gogen.PrintImports(g.w, g.imports...)
}

// writeGoArgsLoop writes the statements of the loop over the arguments
// (if any) into the Go file
func (g *Gosh) writeGoArgsLoop() {
	tag := "argloop"

	g.gDecl("_arg", "", tag)
	g.gDecl("_args", " = []string{", tag)
	for _, arg := range g.args {
		g.gPrint(arg+",", tag)
	}
	g.gPrint("}", tag)

	g.gPrint("for _, _arg = range _args {", tag)
	g.in()
	g.gPrint("_ = _arg", tag) // force the use of _arg
	g.writeScript(goshScriptExec)
	g.out()
	g.gPrint("}", tag)
}

// writeGoFileReadLoop writes the statements of the readloop
// (if any) into the Go file
func (g *Gosh) writeGoFileReadLoop() {
	if !g.runInReadLoop {
		return
	}

	tag := "readloop"

	g.gDecl("_r", " = os.Stdin", tag)
	g.gDecl("_fn", ` = "standard input"`, tag)
	g.gDecl("_fl", "", tag)

	if g.splitLine {
		tag := tag + " - splitline"
		g.gDecl("_sre",
			fmt.Sprintf(" = regexp.MustCompile(%q)", g.splitPattern),
			tag)
	}
	if len(g.filesToRead) > 0 {
		tag := tag + " - filelist"
		g.gDecl("_fns", " = []string{", tag)
		for _, arg := range g.filesToRead {
			g.gPrint(arg+",", tag)
		}
		g.gPrint("}", tag)

		g.gPrint("for _, _fn = range _fns {", tag)
		g.in()
		g.gDecl("_f", "", tag)
		g.gDecl("_err", "", tag)
		g.gPrint(`_f, _err = os.Open(_fn)`, tag)
		g.gPrint(`_fl = 0`, tag)
		g.gPrint(`if _err != nil {`, tag)
		g.in()
		g.gPrintErr(`"Error opening: %q : %v\n", _fn, _err`, tag)
		g.gPrint(`continue`, tag)
		g.out()
		g.gPrint("}", tag)
		g.gPrint("_r = _f", tag)
		if g.inPlaceEdit {
			tag := tag + " - in-place-edit"
			g.gDecl("_w", "", tag)
			g.gPrint(`_w, _err = os.CreateTemp(`, tag)
			g.in()
			g.gPrint(`filepath.Dir(_fn),`, tag)
			g.gPrint(`filepath.Base(_fn) + ".*.new")`, tag)
			g.out()
			g.gPrint(`if _err != nil {`, tag)
			g.in()
			g.gPrintErr(
				`"Error creating the temp file for %q : %v\n", _fn, _err`,
				tag)
			g.gPrint(`_f.Close()`, tag)
			g.gPrint(`continue`, tag)
			g.out()
			g.gPrint("}", tag)
		}
	}

	g.gDecl("_l", " = bufio.NewScanner(_r)", tag)
	g.gPrint("for _l.Scan() {", tag)
	g.in()
	g.gPrint("_fl++", tag)

	if g.splitLine {
		tag := tag + " - splitline"
		g.gDecl("_lp", " = _sre.Split(_l.Text(), -1)", tag)
	}

	g.writeScript(goshScriptExec)

	g.out()
	g.gPrint("}", tag)
	g.gPrint("if _err := _l.Err(); _err != nil {", tag)
	g.in()
	g.gPrintErr(`"Error reading %q : %v\n", _fn, _err`, tag)
	g.out()
	g.gPrint("}", tag)
	if len(g.filesToRead) > 0 {
		tag := tag + " - filelist"
		g.gPrint(`_f.Close()`, tag)
		if g.inPlaceEdit {
			tag := tag + " - in-place-edit"
			g.gPrint(`_w.Close()`, tag)
			g.gPrint(
				`if _err := os.Rename(_fn, _fn+"`+origExt+`"); _err != nil {`,
				tag)
			g.in()
			g.gPrintErr(`"Error making copy of %q : %v\n", _fn, _err`, tag)
			g.out()
			g.gPrint("}", tag)
			g.gPrint(
				`if _err := os.Rename(_w.Name(), _fn); _err != nil {`,
				tag)
			g.in()
			g.gPrintErr(`"Error recreating %q : %v\n", _fn, _err`, tag)
			g.out()
			g.gPrint("}", tag)
		}
		g.out()
		g.gPrint("}", tag)
	}
}

// writeGoFileWebserverInit writes the webserver boilerplate code
// (if any) into the Go file
func (g *Gosh) writeGoFileWebserverInit() {
	if !g.runAsWebserver {
		return
	}

	g.gPrint(
		fmt.Sprintf(`http.Handle(%q, %s)`, g.httpPath, g.httpHandlerInstance()),
		"webserver")
	g.gPrint(
		fmt.Sprintf(`log.Fatal(http.ListenAndServe(":%d", nil))`, g.httpPort),
		"webserver")
}

// httpHandlerInstance returns either the value of the httpHandler (or, if it
// is still set to the default, an instance of that)
func (g *Gosh) httpHandlerInstance() string {
	if g.httpHandler != dfltHTTPHandlerName {
		return g.httpHandler
	}
	return g.httpHandler + "{}"
}

// writeGoFileWebserverHandler writes the webserver handler function
// (if any) into the Go file
func (g *Gosh) writeGoFileWebserverHandler() {
	if !g.runAsWebserver {
		return
	}
	if g.httpHandler != dfltHTTPHandlerName {
		return
	}

	g.gPrint("", "webserver")
	g.gPrint("type "+dfltHTTPHandlerName+" struct{}", "webserver")

	g.gPrint("", "webserver")
	g.gPrint(g.defaultHandlerFuncDecl()+" {", "webserver")
	g.in()
	g.writeScript(goshScriptExec)
	g.out()
	g.gPrint("}", "webserver")
}

// defaultHandlerFuncDecl returns the func declaration for the default HTTP
// Handler func
func (g *Gosh) defaultHandlerFuncDecl() string {
	return fmt.Sprintf("func (%s)ServeHTTP(%s, %s)",
		dfltHTTPHandlerName,
		g.nameType("_rw"),
		g.nameType("_req"))
}

// writeGoFile writes the contents of the Go file
func (g *Gosh) writeGoFile() {
	intro := constantWidthStr("writeGoFile")
	defer timer.Start(intro, verboseTimer)()

	verbose.Print(intro, ": Writing the contents of the Go file\n")

	g.gPrint("package main", "frame")

	g.writeGoshComment()
	g.writeGoFileImports()
	g.writeScript(goshScriptGlobal)

	g.gPrint("", "frame")
	g.gPrint("func main() {", "frame")
	g.in()
	g.gPrint(fmt.Sprintf("if err := os.Chdir(%q); err != nil {", g.runDir),
		"frame")
	g.in()
	g.gPrint(
		fmt.Sprintf("fmt.Printf(%q, %q, err)",
			"Couldn't change directory to %q: %v\n", g.runDir),
		"frame")
	g.gPrint("os.Exit(1)", "frame")
	g.out()
	g.gPrint("}", "frame")

	g.writeScript(goshScriptBefore)

	if g.runAsWebserver {
		g.writeGoFileWebserverInit()
	} else if g.runInReadLoop {
		g.writeGoFileReadLoop()
	} else if len(g.args) > 0 {
		g.writeGoArgsLoop()
	} else {
		g.writeScript(goshScriptExec)
	}

	g.writeScript(goshScriptAfter)

	g.out()
	g.gPrint("}", "frame")

	if g.runAsWebserver {
		g.writeGoFileWebserverHandler()
	}
}

// writeGoshComment writes the introductory comment
func (g *Gosh) writeGoshComment() {
	defer g.print(
		`// ==================================================================`)
	g.print(`

// ==================================================================
// This code was generated by gosh.
// go install github.com/nickwells/utilities/gosh@latest`)
	if !g.addComments {
		return
	}
	g.print(`//
// All lines of code generated by gosh (apart from these) end
// with a comment like this: '//` + goshCommentIntro + `...'.
// User provided code has no automatic end-of-line comment.`)
}
