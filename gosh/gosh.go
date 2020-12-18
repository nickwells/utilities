package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nickwells/errutil.mod/errutil"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/snippet.mod/snippet"
	"github.com/nickwells/xdg.mod/xdg"
)

const (
	dfltHTTPPort        = 8080
	dfltHTTPPath        = "/"
	dfltHTTPHandlerName = "goshHandler"

	dfltSplitPattern = `\s+`

	dfltFormatter      = "gofmt"
	dfltFormatterArg   = "-w"
	goImportsFormatter = "goimports"

	goshCommentIntro = " gosh : "

	goshScriptGlobal = "global"
	goshScriptBefore = "before"
	goshScriptExec   = "exec"
	goshScriptAfter  = "after"
)

type expandFunc func(*Gosh, string) ([]string, error)

// ScriptEntry holds the values describing what should be added to the
// script. The value can be either a snippet filename or else text to be
// added verbatim; the expand func is set to handle these two cases
// appropriately.
type ScriptEntry struct {
	expand expandFunc
	value  string
}

// Gosh records all the details needed to build a Gosh program
type Gosh struct {
	w           *os.File
	indent      int
	addComments bool

	imports []string

	scripts map[string][]ScriptEntry

	runInReadLoop bool
	inPlaceEdit   bool
	splitLine     bool
	splitPattern  string

	runAsWebserver bool
	httpHandler    string
	httpPort       int64
	httpPath       string

	runInReadloopSetters  []*param.ByName
	runAsWebserverSetters []*param.ByName

	showFilename  bool
	dontClearFile bool
	dontRun       bool
	filename      string
	cleanupPath   string

	formatter     string
	formatterSet  bool
	formatterArgs []string

	args        []string
	filesToRead []string
	errMap      *errutil.ErrMap

	snippetDirs  []string
	showSnippets bool
	snippetUsed  map[string]bool
	snippets     *snippet.Cache

	baseTempDir string
	runDir      string

	localModules map[string]string
}

// CacheSnippet will cache the named snippet and copy any imports it requires
// into the set of imports for the gosh script
func (g *Gosh) CacheSnippet(sName string) error {
	s, err := g.snippets.Add(g.snippetDirs, sName)

	if err != nil {
		return err
	}

	g.imports = append(g.imports, s.Imports()...)
	return nil
}

// snippetExpand will return the snippet text. It also checks that the
// snippet is being used in the correct order and returns an error if not.
func snippetExpand(g *Gosh, sName string) ([]string, error) {
	s, err := g.snippets.Get(sName)

	if err != nil {
		return nil, err
	}
	g.snippetUsed[sName] = true
	for _, shouldBeUsed := range s.Follows() {
		if !g.snippetUsed[shouldBeUsed] {
			g.addError("Snippet out of order",
				fmt.Errorf("snippet %q should appear before snippet %q",
					shouldBeUsed, sName))
		}
	}

	if len(s.Text()) == 0 {
		return nil, nil
	}

	var content []string
	addSnippetComment(&content, s.Path())
	addSnippetComment(&content, "BEGIN")
	content = append(content, s.Text()...)
	addSnippetComment(&content, "END")

	return content, nil
}

// addSnippetComment writes the message at the end of a snippet comment
func addSnippetComment(script *[]string, message string) {
	*script = append(*script, "//"+goshCommentIntro+"snippet : "+message)
}

// NewGosh creates a new instance of the Gosh struct with all the initial
// default values set correctly.
func NewGosh() *Gosh {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't get the working directory:", err)
		os.Exit(1)
	}
	g := &Gosh{
		scripts: map[string][]ScriptEntry{
			goshScriptGlobal: {},
			goshScriptBefore: {},
			goshScriptExec:   {},
			goshScriptAfter:  {},
		},

		addComments:   true,
		splitPattern:  dfltSplitPattern,
		formatter:     dfltFormatter,
		formatterArgs: []string{dfltFormatterArg},

		errMap: errutil.NewErrMap(),

		httpPort:    dfltHTTPPort,
		httpPath:    dfltHTTPPath,
		httpHandler: dfltHTTPHandlerName,

		runDir: cwd,

		snippetUsed: map[string]bool{},
		snippets:    &snippet.Cache{},
	}

	g.setDfltSnippetPath()

	return g
}

// setDfltSnippetPath populates the snippetsDirs slice with the default value.
func (g *Gosh) setDfltSnippetPath() {
	snippetPath := []string{
		"github.com",
		"nickwells",
		"utilities",
		"gosh",
		"snippets"}

	g.snippetDirs = []string{
		filepath.Join(append([]string{xdg.ConfigHome()}, snippetPath...)...),
	}
	dirs := xdg.ConfigDirs()
	if len(dirs) > 0 {
		g.snippetDirs = append(g.snippetDirs,
			filepath.Join(append(dirs[:1], snippetPath...)...))
	}
}

// verbatim returns the passed string without any expansion. It is suitable
// as a ScriptEntry expand func for code to be added directly to the file
func verbatim(_ *Gosh, s string) ([]string, error) {
	return []string{s}, nil
}

// AddScriptEntry adds the script entry to the named script. It panics if the
// script name is invalid or the expandFunc is nil.
func (g *Gosh) AddScriptEntry(sName, v string, ef expandFunc) {
	s, ok := g.scripts[sName]
	if !ok {
		panic(fmt.Errorf("the script name is invalid: %q", sName))
	}
	if ef == nil {
		panic(errors.New("the expansion function is nil"))
	}
	g.scripts[sName] = append(s, ScriptEntry{expand: ef, value: v})
}

// addError adds the error to the named error map entry
func (g *Gosh) addError(name string, err error) {
	g.errMap.AddError(name, err)
}

// checkScripts checks that not all the scripts are empty
func (g *Gosh) checkScripts() {
	for _, s := range g.scripts {
		if len(s) > 0 {
			return
		}
	}
	g.addError("no code", errors.New("There is no code to run"))
}

// reportErrors checks if there are errors to report and if there are it
// reports them and exits.
func (g *Gosh) reportErrors() {
	if errCount, _ := g.errMap.CountErrors(); errCount != 0 {
		g.errMap.Report(os.Stderr, "gosh")
		os.Exit(1)
	}
}

// in increases the indent level by 1
func (g *Gosh) in() {
	g.indent++
}

// out decreases the indent level by 1
func (g *Gosh) out() {
	g.indent--
}

// indentStr returns a string to provide the current indent
func (g *Gosh) indentStr() string {
	return strings.Repeat("\t", g.indent)
}

// comment returns the standard comment string explaining why the line is
// in the generated code
func (g *Gosh) comment(text string) string {
	if !g.addComments {
		return ""
	}
	return "\t//" + goshCommentIntro + text
}

// varInfo records information about a variable. This is for the
// autogenerated variable declarations and for generating the note for the
// usage message
type varInfo struct {
	typeName string
	desc     string
}
type varMap map[string]varInfo

var knownVarMap varMap = varMap{
	"_arg": {
		typeName: "string",
		desc:     "the argument",
	},
	"_args": {
		typeName: "[]string",
		desc:     "the list of arguments",
	},
	"_r": {
		typeName: "io.Reader",
		desc:     "the reader for the scanner (may be stdin)",
	},
	"_rw": {
		typeName: "http.ResponseWriter",
		desc:     "the response writer for the web server",
	},
	"_req": {
		typeName: "*http.Request",
		desc:     "the request to the seb server",
	},
	"_gh": {
		typeName: dfltHTTPHandlerName,
		desc:     "the HTTP Handler (providing ServeHTTP)",
	},
	"_w": {
		typeName: "*os.File",
		desc:     "the file written to if editing in place",
	},
	"_l": {
		typeName: "*bufio.Scanner",
		desc:     "a buffered scanner used to read the files",
	},
	"_fl": {
		typeName: "int",
		desc:     "the current line number in the file",
	},
	"_fn": {
		typeName: "string",
		desc:     "the name of the file (or stdin)",
	},
	"_fns": {
		typeName: "[]string",
		desc:     "the list of names of the files",
	},
	"_f": {
		typeName: "*os.File",
		desc:     "the file being read",
	},
	"_err": {
		typeName: "error",
		desc:     "an error",
	},
	"_sre": {
		typeName: "*regexp.Regexp",
		desc:     "the regexp used to split lines",
	},
	"_lp": {
		typeName: "[]string",
		desc:     "the parts of the line (when split)",
	},
}

// nameType looks up the name in knownVarMap and if it is found it will
// return the name and type as a single string suitable for use as a variable
// or parameter declaration
func (g *Gosh) nameType(name string) string {
	vi, ok := knownVarMap[name]
	if !ok {
		panic(fmt.Errorf("%q is not in the map of known variables", name))
	}
	return name + " " + vi.typeName
}

// gDecl declares a variable. The variable must be in the map of known
// variables (which is used to provide a note for the usage message). The
// declaration is indented and the Gosh comment is added
func (g *Gosh) gDecl(name, initVal, tag string) {
	fmt.Fprintln(g.w,
		g.indentStr()+"var "+g.nameType(name)+initVal+g.comment(tag))
}

// makeKnownVarList will format the entries in knownVarMap into a form
// suitable for the usage message
func makeKnownVarList() string {
	kvl := ""
	var keys = make([]string, 0, len(knownVarMap))
	maxVarNameLen := 0
	maxTypeNameLen := 0
	for k, vi := range knownVarMap {
		keys = append(keys, k)
		if len(k) > maxVarNameLen {
			maxVarNameLen = len(k)
		}
		if len(vi.typeName) > maxTypeNameLen {
			maxTypeNameLen = len(vi.typeName)
		}
	}
	sort.Strings(keys)

	sep := ""
	for _, k := range keys {
		vi := knownVarMap[k]
		kvl += fmt.Sprintf("%s%-*.*s %-*.*s  %s",
			sep,
			maxVarNameLen, maxVarNameLen, k,
			maxTypeNameLen, maxTypeNameLen, vi.typeName,
			vi.desc)
		sep = "\n"
	}
	return kvl
}

// gPrint prints the text with the appropriate indent and the Gosh comment
func (g *Gosh) gPrint(s, tag string) {
	if s == "" {
		fmt.Fprintln(g.w)
		return
	}

	fmt.Fprintln(g.w, g.indentStr()+s+g.comment(tag))
}

// gPrintErr prints a line that reports an error with the appropriate indent
// and the Gosh comment
func (g *Gosh) gPrintErr(s, tag string) {
	fmt.Fprintln(g.w,
		g.indentStr()+"fmt.Fprintf(os.Stderr, "+s+")"+g.comment(tag))
}

// print prints the text with the appropriate indent and no comment. This
// should be used for user-supplied code
func (g *Gosh) print(s string) {
	fmt.Fprintln(g.w, g.indentStr()+s)
}
