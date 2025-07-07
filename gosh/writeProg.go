package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/verbose.mod/verbose"
)

const (
	equals = `=======================================================`

	frameTag = "frame"
	webTag   = "webserver"
	argTag   = "argsloop"
	rlTag    = "readloop"

	splitSfx = " - splitline"
	filesSfx = " - filelist"
	ipeSfx   = " - in-place-edit"
)

// writeScript writes the contents of the named script. It panics if the
// script name is not found.
func (g *gosh) writeScript(scriptName string) {
	script, ok := g.scripts[scriptName]
	if !ok {
		panic(fmt.Errorf("invalid script name: %q", scriptName))
	}

	if len(script) == 0 {
		return
	}

	var (
		sectionStart = "Section start: " + scriptName
		sectionEnd   = "Section end:   " + scriptName
	)

	sectionFrame := strings.Repeat("=", len(sectionStart))

	if g.addComments {
		g.printBlank()
		g.print(g.comment(sectionFrame))
		g.print(g.comment(sectionStart))
		g.print(g.comment(sectionFrame))
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

	if g.addComments {
		g.print(g.comment(sectionFrame))
		g.print(g.comment(sectionEnd))
		g.print(g.comment(sectionFrame))
		g.printBlank()
	}
}

// writeImports writes the import statements into the Go file
func (g *gosh) writeImports() {
	if len(g.args) > 0 && !g.skipArgLoop {
		g.imports = append(g.imports, "os")
	}

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
		g.imports = append(g.imports, "log")
	}

	gogen.PrintImports(g.w, g.imports...)
}

// writeArgsLoop writes the statements of the loop over the arguments
// (if any) into the Go file
func (g *gosh) writeArgsLoop() {
	tag := argTag

	g.writeScript(beforeSect)

	if !g.skipArgLoop {
		g.gPrint("for _, _arg := range os.Args[1:] {", tag)
		g.in()
		g.gPrint("_ = _arg", tag) // force the use of _arg
	}

	g.writeScript(beforeInnerSect)

	g.writeScript(execSect)

	g.writeScript(afterInnerSect)

	if !g.skipArgLoop {
		g.out()
		g.gPrint("}", tag)
	}

	g.writeScript(afterSect)
}

// writeReadLoop writes the statements of the readloop
// (if any) into the Go file
func (g *gosh) writeReadLoop() {
	tag := rlTag

	g.gDecl("_fn", ` = "standard input"`, tag)
	g.gDecl("_fl", "", tag)

	if g.splitLine {
		g.gDecl("_sre",
			fmt.Sprintf(" = regexp.MustCompile(%q)", g.splitPattern),
			tag+splitSfx)
	}

	g.writeScript(beforeSect)

	if g.filesToRead {
		g.writeFileLoopOpen(tag + filesSfx)
		g.gDecl("_l", " = bufio.NewScanner(_f)", tag)
	} else {
		g.gDecl("_l", " = bufio.NewScanner(os.Stdin)", tag)
	}

	g.writeScript(beforeInnerSect)
	g.writeScanLoopOpen(tag)

	g.writeScript(execSect)

	g.writeScanLoopClose(tag)
	g.writeScript(afterInnerSect)

	if g.filesToRead {
		g.writeFileLoopClose(tag + filesSfx)
	}

	g.writeScript(afterSect)
}

// writeScanLoopOpen writes the code to open the loop reading from the scanner.
func (g *gosh) writeScanLoopOpen(tag string) {
	g.gPrint("for _l.Scan() {", tag)
	g.in()
	g.gPrint("_fl++", tag)

	if g.splitLine {
		g.gDecl("_lp", " = _sre.Split(_l.Text(), -1)", tag+splitSfx)
	}
}

// writeScanLoopClose writes the code to close the loop reading from the
// scanner.
func (g *gosh) writeScanLoopClose(tag string) {
	g.out()
	g.gPrint("}", tag)
	g.gPrint("if _err := _l.Err(); _err != nil {", tag)
	g.in()
	g.gPrintErr(`"Error reading %q : %v\n", _fn, _err`, tag)
	g.out()
	g.gPrint("}", tag)
}

// writeFileLoopOpen writes the opening of the loop over the list of filenames.
func (g *gosh) writeFileLoopOpen(tag string) {
	g.gPrint("for _, _fn = range os.Args[1:] {", tag)
	{
		g.in()
		g.gDecl("_f", "", tag)
		g.gDecl("_err", "", tag)
		g.gPrint(`_f, _err = os.Open(_fn)`, tag)
		g.gPrint(`if _err != nil {`, tag)
		{
			g.in()
			g.gPrintErr(`"Error opening: %q : %v\n", _fn, _err`, tag)
			g.gPrint(`continue`, tag)
			g.out()
		}

		g.gPrint("}", tag)
		g.gPrint(`_fl = 0`, tag)

		g.writeInPlaceEditOpen(tag + ipeSfx)
	}
}

// writeFileLoopClose writes the code to close the loop ranging over the file
// names.
func (g *gosh) writeFileLoopClose(tag string) {
	g.gPrint(`_f.Close()`, tag)

	g.writeInPlaceEditClose(tag + ipeSfx)

	g.out()
	g.gPrint("}", tag)
}

// writeInPlaceEditOpen writes the declaration and initialisation of the
// writer used for in-place editing. It writes code to handle any errors
// detected.
func (g *gosh) writeInPlaceEditOpen(tag string) {
	if !g.inPlaceEdit {
		return
	}

	g.gDecl("_w", "", tag)
	g.gPrint(`_w, _err = os.CreateTemp(`, tag)
	{
		g.in()
		g.gPrint(`filepath.Dir(_fn),`, tag)
		g.gPrint(`filepath.Base(_fn) + ".*.new")`, tag)
		g.out()
	}

	g.gPrint(`if _err != nil {`, tag)
	{
		g.in()
		g.gPrintErr(`"Error creating the temp file for %q : %v\n", _fn, _err`,
			tag)
		g.gPrint(`_f.Close()`, tag)
		g.gPrint(`continue`, tag)
		g.out()
	}

	g.gPrint("}", tag)

	g.gPrint("{", tag)
	{
		g.in()
		g.gPrint(`_fi, _err := _f.Stat()`, tag)
		g.gPrint(`if _err != nil {`, tag)
		{
			g.in()
			g.gPrintErr(
				`"Warning: Couldn't get status for %q: %v\n", _fn, _err`,
				tag)
			g.out()
		}

		g.gPrint("} else {", tag)
		{
			g.in()
			g.gPrint(`_err = _w.Chmod(_fi.Mode())`, tag)
			g.gPrint(`if _err != nil {`, tag)
			{
				g.in()
				g.gPrintErr(
					`"Warning: Couldn't set mode for %q: %v\n", _fn, _err`,
					tag)
				g.out()
			}

			g.gPrint("}", tag)
			g.out()
		}

		g.gPrint("}", tag)
		g.out()
	}

	g.gPrint("}", tag)
}

// writeInPlaceEditClose writes the code to complete the operation of the
// in-place edit of the given files.
func (g *gosh) writeInPlaceEditClose(tag string) {
	if !g.inPlaceEdit {
		return
	}

	g.gPrint(`_w.Close()`, tag)
	g.gPrint(`if _err := os.Rename(_fn, _fn+"`+origExt+`"); _err != nil {`, tag)
	{
		g.in()
		g.gPrintErr(`"Error making copy of %q : %v\n", _fn, _err`, tag)
		g.out()
	}

	g.gPrint("}", tag)
	g.gPrint(`if _err := os.Rename(_w.Name(), _fn); _err != nil {`, tag)
	{
		g.in()
		g.gPrintErr(`"Error recreating %q : %v\n", _fn, _err`, tag)
		g.out()
	}

	g.gPrint("}", tag)
}

// writeWebserverInit writes the webserver boilerplate code
// (if any) into the Go file
func (g *gosh) writeWebserverInit() {
	tag := webTag

	g.writeScript(beforeSect)
	g.writeScript(beforeInnerSect)

	g.gPrint(fmt.Sprintf(`http.Handle(%q, %s)`,
		g.httpPath, g.httpHandlerInstance()),
		tag)

	g.writeScript(afterInnerSect)
	g.writeScript(afterSect)

	g.gPrint(fmt.Sprintf(`log.Fatal(http.ListenAndServe(":%d", nil))`,
		g.httpPort),
		tag)
}

// httpHandlerInstance returns either the value of the httpHandler (or, if it
// is still set to the default, an instance of that)
func (g *gosh) httpHandlerInstance() string {
	if g.httpHandler != dfltHTTPHandlerName {
		return g.httpHandler
	}

	return g.httpHandler + "{}"
}

// writeWebserverHandler writes the webserver handler function
// (if any) into the Go file
func (g *gosh) writeWebserverHandler() {
	if g.httpHandler != dfltHTTPHandlerName {
		return
	}

	tag := webTag

	g.gPrint("", tag)
	g.gPrint("type "+dfltHTTPHandlerName+" struct{}", tag)

	g.gPrint("", tag)
	g.gPrint(g.defaultHandlerFuncDecl()+" {", tag)
	g.in()
	g.writeScript(execSect)
	g.out()
	g.gPrint("}", tag)
}

// defaultHandlerFuncDecl returns the func declaration for the default HTTP
// Handler func
func (g *gosh) defaultHandlerFuncDecl() string {
	return fmt.Sprintf("func (%s)ServeHTTP(%s, %s)",
		dfltHTTPHandlerName,
		g.nameType("_rw"),
		g.nameType("_req"))
}

// writeGoFile creates the gosh.go file and writes its contents. It is run
// from within the temp directory.
func (g *gosh) writeGoFile() {
	defer g.dbgStack.Start("writeGoFile", "Writing the Go file")()

	intro := g.dbgStack.Tag()

	verbose.Println(intro, " Creating the Go file: ", goshFilename)

	var err error

	g.w, err = os.Create(goshFilename)

	defer g.w.Close()

	g.reportFatalError("create the Go file", goshFilename, err)

	g.gPrint("package main", frameTag)

	g.writeImports()
	g.writeGoshComment()
	g.writeScript(globalSect)

	g.writeMainOpen()

	if g.runAsWebserver {
		g.writeWebserverInit()
	} else if g.runInReadLoop {
		g.writeReadLoop()
	} else if len(g.args) > 0 {
		g.writeArgsLoop()
	} else {
		g.writeScript(beforeSect)
		g.writeScript(beforeInnerSect)
		g.writeScript(execSect)
		g.writeScript(afterInnerSect)
		g.writeScript(afterSect)
	}

	g.writeMainClose()

	if g.runAsWebserver {
		g.writeWebserverHandler()
	}
}

// writeMainOpen writes the opening of the main func.
func (g *gosh) writeMainOpen() {
	g.gPrint("", frameTag)
	g.gPrint("func main() {", frameTag)
	g.in()
}

// writeMainClose writes the closing of the main func.
func (g *gosh) writeMainClose() {
	g.out()
	g.gPrint("}", frameTag)
}

// writeGoshComment writes the introductory comment
func (g *gosh) writeGoshComment() {
	defer g.print(`// ` + equals)

	g.print(`

// ` + equals)
	g.print(`// This code was generated by gosh.
// go install github.com/nickwells/utilities/gosh@latest`)

	if !g.addComments {
		return
	}

	g.print(`//
// All lines of code generated by gosh (apart from these) end
// with a comment like this: '//` + goshCommentIntro + `...'.
// User provided code has no automatic end-of-line comment.`)
}
