package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/nickwells/errutil.mod/errutil"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/param.mod/v6/paramtest"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

const (
	testCodeFile0 = "testdata/codeFile0"
	testCodeFile1 = "testdata/codeFile1"
	testCodeFile2 = "testdata/codeFile2"

	testDataFile1   = "testdata/file1"
	testDataFile2   = "testdata/file2"
	testHasOrigFile = "testdata/hasOrigFile"

	testNoSuchFile = "testdata/nonesuch"

	snippetsDir = "snippets"
	snippet0    = "s0"
	snippet1    = "s1"
)

// cmpGoshStruct compares the value with the expected value and returns
// an error if they differ
func cmpGoshStruct(iVal, iExpVal any) error {
	val, ok := iVal.(*Gosh)
	if !ok {
		return errors.New("Bad value: not a pointer to a Gosh struct")
	}
	expVal, ok := iExpVal.(*Gosh)
	if !ok {
		return errors.New("Bad expected value: not a pointer to a Gosh struct")
	}

	return testhelper.DiffVals(val, expVal,
		[]string{"snippets"},             // ignore diffs in the snippet caches
		[]string{"snippetDirs"},          // ... and the snippet dir list
		[]string{"runInReadloopSetters"}, // ... and the lists of ByName param
		[]string{"runAsWebserverSetters"},
	)
}

// makePSet returns a param set with the gosh params set up
func makePSet(g *Gosh) *param.PSet {
	slp := &snippetListParams{}
	return paramset.NewNoHelpNoExitNoErrRptOrPanic(paramOptFuncs(g, slp)...)
}

// mkTestGosh makes a new Gosh and calls the goshSetters on it which are
// expected to set various fields as required.
func mkTestGosh(goshSetter ...func(g *Gosh)) *Gosh {
	g := newGosh()
	for _, gs := range goshSetter {
		gs(g)
	}
	return g
}

// mkTestParser populates and returns a paramtest.Parser ready to be added to
// the testcases.
func mkTestParser(
	errs errutil.ErrMap, id testhelper.ID, gs func(g *Gosh), args ...string,
) paramtest.Parser {
	actVal := mkTestGosh()

	expVal := mkTestGosh(gs)

	return paramtest.Parser{
		ID:             id,
		ExpParseErrors: errs,
		Val:            actVal,
		Ps:             makePSet(actVal),
		ExpVal:         expVal,
		Args:           args,
		CheckFunc:      cmpGoshStruct,
	}
}

// populateFileScriptEntries generates the slice of file contents for the
// codefiles. This slice is used to check that the -...-file parameters are
// working correctly
func populateFileScriptEntries(t *testing.T) ([]string, []ScriptEntry) {
	t.Helper()

	files := []string{
		testCodeFile0,
		testCodeFile1,
		testCodeFile2,
	}
	fileSE := []ScriptEntry{}
	for _, fName := range files {
		contents, err := os.ReadFile(fName)
		if err != nil {
			t.Fatalf("Could not read the test code file: %q: %v", fName, err)
		}
		fileSE = append(fileSE,
			ScriptEntry{expand: verbatim, value: string(contents)})
	}
	return files, fileSE
}

// populateSnippetScriptEntries generates the slice of snippets. This slice
// is used to check that the -...-snippet parameters are working correctly
func populateSnippetScriptEntries() ([]string, []ScriptEntry) {
	snippets := []string{
		snippet0,
		snippet1,
	}
	snippetSE := []ScriptEntry{}
	for _, sName := range snippets {
		snippetSE = append(snippetSE,
			ScriptEntry{expand: snippetExpand, value: sName})
	}
	return snippets, snippetSE
}

// populateCodeScriptEntries generates a slice of lines of code and the
// corresponding ScriptEntry elements
func populateCodeScriptEntries() ([]string, []ScriptEntry) {
	stmt := make([]string, 10)
	stmtSE := []ScriptEntry{}
	for i := 0; i < len(stmt); i++ {
		stmt[i] = fmt.Sprintf("// %d", i)
		stmtSE = append(stmtSE, ScriptEntry{expand: verbatim, value: stmt[i]})
	}
	return stmt, stmtSE
}

const (
	printTypeP = iota
	printTypePln
	printTypePf
	printTypeWP
	printTypeWPln
	printTypeWPf
	printTypeWebP
	printTypeWebPln
	printTypeWebPf
)

// populatePrintScriptEntries generates a pair of maps representing various
// print arguments and the corresponding types of print ScriptEntry elements
func populatePrintScriptEntries() (map[int]string, map[int]ScriptEntry) {
	intro := map[int]string{
		printTypeP:      "fmt.Print(",
		printTypePln:    "fmt.Println(",
		printTypePf:     "fmt.Printf(",
		printTypeWP:     "fmt.Fprint(_w, ",
		printTypeWPln:   "fmt.Fprintln(_w, ",
		printTypeWPf:    "fmt.Fprintf(_w, ",
		printTypeWebP:   "fmt.Fprint(_rw, ",
		printTypeWebPln: "fmt.Fprintln(_rw, ",
		printTypeWebPf:  "fmt.Fprintf(_rw, ",
	}
	printVal := map[int]string{
		printTypeP:      `"Hello", 42, "\n"`,
		printTypePln:    `"Hello", 42`,
		printTypePf:     `"Hello %d\n", 42`,
		printTypeWP:     `"Hello, World", 42, "\n"`,
		printTypeWPln:   `"Hello, World", 42`,
		printTypeWPf:    `"Hello, World %d\n", 42`,
		printTypeWebP:   `"Hello, World", 42, "\n"`,
		printTypeWebPln: `"Hello, World", 42`,
		printTypeWebPf:  `"Hello, World %d\n", 42`,
	}
	printValSE := map[int]ScriptEntry{}
	for k := range intro {
		printValSE[k] = ScriptEntry{
			expand: verbatim,
			value:  intro[k] + printVal[k] + ")",
		}
	}

	return printVal, printValSE
}

// TestParseParamsCmdGosh will use the paramtest.Parser to make sure the
// behaviour of the parameter setting is as expected. This tests just the
// parameters in the 'cmd-gosh' group.
func TestParseParamsCmdGosh(t *testing.T) {
	testCases := []paramtest.Parser{}

	// no params; no change
	testCases = append(testCases,
		mkTestParser(nil,
			testhelper.MkID("no params no change"),
			func(g *Gosh) {}))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID("add-comments"), func(g *Gosh) {
			g.addComments = true
		}, "-add-comments"))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID("base-temp-dir with good dir"),
			func(g *Gosh) {
				g.baseTempDir = "testdata/baseTempDir"
			}, "-base-temp-dir", "testdata/baseTempDir"))

	{
		parseErrs := errutil.ErrMap{}
		parseErrs.AddError(
			"base-temp-dir",
			errors.New(
				`path: "testdata/nosuchdir": should exist but does not;`+
					` "testdata" exists but "nosuchdir" does not`+
					"\n"+
					`At: [command line]: Supplied Parameter:2:`+
					` "-base-temp-dir" "testdata/nosuchdir"`))

		testCases = append(testCases,
			mkTestParser(parseErrs,
				testhelper.MkID("base-temp-dir with bad dir"),
				func(g *Gosh) {},
				"-base-temp-dir", "testdata/nosuchdir"))
	}

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.buildArgs = []string{"-a", "-b", "-c", "-d", "-e"}
		},
			"-build-arg", "-a",
			"-build-args", "-b",
			"-args-build", "-c",
			"-b-arg", "-d",
			"-b-args", "-e"))

	for _, p := range []string{
		"-dont-exec",
		"-dont-run",
		"-no-exec",
		"-no-run",
	} {
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(p),
				func(g *Gosh) {
					g.dontRun = true
					g.dontCleanupUserChoice = true
				},
				p))
	}

	for _, p := range []string{
		"-dont-populate-imports",
		"-dont-auto-import",
	} {
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(p), func(g *Gosh) {
				g.dontPopulateImports = true
			}, p))
	}

	for _, p := range []string{
		"-edit-program",
		"-edit",
	} {
		testCases = append(testCases,
			mkTestParser(nil,
				testhelper.MkID(p), func(g *Gosh) { g.edit = true }, p))
	}

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID("edit-repeat"),
			func(g *Gosh) {
				g.edit = true
				g.editRepeat = true
			},
			"-edit-repeat"))

	testCases = append(testCases,
		mkTestParser(nil,
			testhelper.MkID(paramNameScriptEditor),
			func(g *Gosh) {
				g.editorParam = "xxx"
			},
			"-"+paramNameScriptEditor, "xxx"))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID("formatter"),
			func(g *Gosh) {
				g.formatter = "xxx"
				g.formatterSet = true
			},
			"-formatter", "xxx"))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID("formatter-args"),
			func(g *Gosh) { g.formatterArgs = []string{"-a", "-b", "-c"} },
			"-formatter-args", "-a,-b,-c"))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID("importer"),
			func(g *Gosh) {
				g.importPopulator = "xxx"
				g.importPopulatorSet = true
			},
			"-importer", "xxx"))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID("importer-args"),
			func(g *Gosh) {
				g.importPopulatorArgs = []string{"-a", "-b", "-c"}
			},
			"-importer-args", "-a,-b,-c"))

	for _, p := range []string{
		"-set-executable-name",
		"-set-program-name",
		"-executable-name",
		"-program-name",
	} {
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(p), func(g *Gosh) {
				g.execName = "TestGosh"
				g.dontCleanupUserChoice = true
			},
				p, "TestGosh"))
	}

	for _, p := range []string{
		"-show-filename",
		"-show-file",
		"-keep",
	} {
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(p),
				func(g *Gosh) {
					g.dontCleanupUserChoice = true
				},
				p))
	}

	for _, p := range []string{
		"-show-timings",
		"-show-timing",
		"-show-times",
		"-show-time",
	} {
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(p),
				func(g *Gosh) { g.dbgStack.ShowTimings = true }, p))
	}

	for _, tc := range testCases {
		_ = tc.Test(t)
	}
}

// TestParseParamsCmdReadloop will use the paramtest.Parser to make sure the
// behaviour of the parameter setting is as expected. This tests just the
// parameters in the 'cmd-readloop' group.
func TestParseParamsCmdReadloop(t *testing.T) {
	shouldExistErr := fmt.Errorf("path: %q: %w",
		testNoSuchFile, filecheck.ErrShouldExistButDoesNot)
	shouldNotExistErr := fmt.Errorf("path: %q: %w",
		testHasOrigFile+".orig", filecheck.ErrShouldNotExistButDoes)

	printVal, printValSE := populatePrintScriptEntries()

	testCases := []paramtest.Parser{}

	for _, p := range []string{
		"-" + paramNameInPlaceEdit,
		"-i",
	} {
		parseErrs := errutil.ErrMap{}
		parseErrs.AddError(
			"Final Checks",
			errors.New(
				`You have given the "-in-place-edit"`+
					` parameter but no filenames have been given`+
					` (they should be supplied following "--")`))
		testCases = append(testCases,
			mkTestParser(parseErrs,
				testhelper.MkID("in-place edit, no files"),
				func(g *Gosh) {
					g.runInReadLoop = true
					g.inPlaceEdit = true
				}, p))

		testCases = append(testCases,
			mkTestParser(nil,
				testhelper.MkID("in-place edit, bad file"),
				func(g *Gosh) {
					g.runInReadLoop = true
					g.inPlaceEdit = true
					g.errMap.AddError("file check", shouldExistErr)
				}, p, "--", testNoSuchFile))

		testCases = append(testCases,
			mkTestParser(nil,
				testhelper.MkID("in-place edit, has orig file"),
				func(g *Gosh) {
					g.runInReadLoop = true
					g.inPlaceEdit = true
					g.errMap.AddError("original file check", shouldNotExistErr)
				}, p, "--", testHasOrigFile))

		testCases = append(testCases,
			mkTestParser(nil,
				testhelper.MkID("in-place edit, good args"),
				func(g *Gosh) {
					g.runInReadLoop = true
					g.inPlaceEdit = true
					g.filesToRead = true
					g.args = []string{testDataFile1, testDataFile2}
				}, p, "--", testDataFile1, testDataFile2))
	}

	for _, p := range []string{
		"-" + paramNameReadloop,
		"-n",
	} {
		testCases = append(testCases,
			mkTestParser(nil,
				testhelper.MkID("run-in-readloop - bad file"),
				func(g *Gosh) {
					g.runInReadLoop = true
					g.errMap.AddError("file check", shouldExistErr)
				}, p, "--", testNoSuchFile))

		testCases = append(testCases,
			mkTestParser(nil,
				testhelper.MkID("run-in-readloop - good files"),
				func(g *Gosh) {
					g.runInReadLoop = true
					g.filesToRead = true
					g.args = []string{testDataFile1, testDataFile2}
				}, p, "--", testDataFile1, testDataFile2))

		testCases = append(testCases,
			mkTestParser(nil,
				testhelper.MkID("run-in-readloop - duplicate files"),
				func(g *Gosh) {
					g.runInReadLoop = true
					g.filesToRead = true
					g.args = []string{testDataFile1}
					g.errMap.AddError("duplicate filename",
						errors.New(`filename "`+
							testDataFile1+
							`" has been given more than once,`+
							` first at 0 and again at 1`))
				}, p, "--", testDataFile1, testDataFile1))
	}

	for _, p := range []string{
		"-split-line",
		"-s",
	} {
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(""),
				func(g *Gosh) {
					g.splitLine = true
					g.runInReadLoop = true
				}, p))
	}

	for _, p := range []string{
		"-split-pattern",
		"-sp",
	} {
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(""),
				func(g *Gosh) {
					g.splitLine = true
					g.runInReadLoop = true
					g.splitPattern = "[,.;:]"
				}, p, "[,.;:]"))
	}

	for _, p := range []struct {
		param string
		idx   int
	}{
		{"-w-print", printTypeWP},
		{"-w-p", printTypeWP},
		{"-w-println", printTypeWPln},
		{"-w-pln", printTypeWPln},
		{"-w-printf", printTypeWPf},
		{"-w-pf", printTypeWPf},
	} {
		parseErrs := errutil.ErrMap{}
		parseErrs.AddError(
			"Final Checks",
			errors.New(
				`You are writing to the file used when in-place editing`+
					` (through one of the "-w-print" printing parameters)`+
					` but you are not editing any files.`+"\n\n"+
					`Give the "-`+paramNameInPlaceEdit+`" parameter if you`+
					` want to edit a file in-place or else write to`+
					` standard output with a different printing parameter.`))

		testCases = append(testCases,
			mkTestParser(parseErrs, testhelper.MkID(""), func(g *Gosh) {
				g.imports = []string{"fmt"}
				g.scripts[execSect] = []ScriptEntry{printValSE[p.idx]}
			}, p.param, printVal[p.idx]))

		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(""),
				func(g *Gosh) {
					g.imports = []string{"fmt"}
					g.inPlaceEdit = true
					g.runInReadLoop = true
					g.filesToRead = true
					g.args = []string{testDataFile1}
					g.scripts[execSect] = []ScriptEntry{printValSE[p.idx]}
				}, p.param, printVal[p.idx],
				"-"+paramNameInPlaceEdit, "--", testDataFile1))
	}

	for _, tc := range testCases {
		_ = tc.Test(t)
	}
}

// TestParseParamsCmdWeb will use the paramtest.Parser to make sure the
// behaviour of the parameter setting is as expected. This tests just the
// parameters in the 'cmd-web' group.
func TestParseParamsCmdWeb(t *testing.T) {
	printVal, printValSE := populatePrintScriptEntries()

	testCases := []paramtest.Parser{}

	for _, p := range []struct {
		param string
		idx   int
	}{
		{"-web-print", printTypeWebP},
		{"-web-p", printTypeWebP},
		{"-web-println", printTypeWebPln},
		{"-web-pln", printTypeWebPln},
		{"-web-printf", printTypeWebPf},
		{"-web-pf", printTypeWebPf},
	} {
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(""),
				func(g *Gosh) {
					g.imports = []string{"fmt"}
					g.runAsWebserver = true
					g.scripts[execSect] = []ScriptEntry{printValSE[p.idx]}
				}, p.param, printVal[p.idx]))
	}

	for _, p := range []string{
		"-http-handler",
		"-http-h",
	} {
		const handlerName = "HTTPHandler"
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(""),
				func(g *Gosh) {
					g.runAsWebserver = true
					g.httpHandler = handlerName
				}, p, handlerName))
	}

	{
		const pathName = "HTTP-Path"
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(""),
				func(g *Gosh) {
					g.runAsWebserver = true
					g.httpPath = pathName
				}, "-http-path", pathName))
	}

	{
		const httpPortNum = 8001
		const httpPortStr = "8001"
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(""),
				func(g *Gosh) {
					g.runAsWebserver = true
					g.httpPort = httpPortNum
				}, "-http-port", httpPortStr))
	}
	testCases = append(testCases,
		mkTestParser(nil,
			testhelper.MkID(""), func(g *Gosh) { g.runAsWebserver = true },
			"-http-server"))
	testCases = append(testCases,
		mkTestParser(nil,
			testhelper.MkID(""), func(g *Gosh) { g.runAsWebserver = true },
			"-http"))

	for _, tc := range testCases {
		_ = tc.Test(t)
	}
}

// TestParseParams will use the paramtest.Parser to make sure the behaviour
// of the parameter setting is as expected. This tests bad combinations of
// parameters.
func TestParseParamsBad(t *testing.T) {
	stmt, stmtSE := populateCodeScriptEntries()

	testCases := []paramtest.Parser{}

	{
		parseErrs := errutil.ErrMap{}
		parseErrs.AddError(
			"Final Checks",
			errors.New(`gosh cannot run in a read-loop and run as a webserver at the same time. Parameters set at:
	[command line]: Supplied Parameter:2: "-http"
	[command line]: Supplied Parameter:1: "-run-in-readloop"`))

		testCases = append(testCases,
			mkTestParser(parseErrs, testhelper.MkID(""), func(g *Gosh) {
				g.runInReadLoop = true
				g.runAsWebserver = true
			}, "-run-in-readloop", "-http"))
	}

	{
		const httpHandler = "HTTPHandler"
		parseErrs := errutil.ErrMap{}
		parseErrs.AddError(
			"Final Checks",
			errors.New(`You have provided an HTTP handler but also given`+
				` lines of code to run. These lines of code will never run.`))

		testCases = append(testCases,
			mkTestParser(parseErrs, testhelper.MkID(""), func(g *Gosh) {
				g.scripts[execSect] = []ScriptEntry{stmtSE[0]}
				g.httpHandler = httpHandler
				g.runAsWebserver = true
			}, "-e", stmt[0], "-http-handler", httpHandler))
	}

	for _, tc := range testCases {
		_ = tc.Test(t)
	}
}

// TestParseParamsPrinting will use the paramtest.Parser to make sure the
// behaviour of the parameter setting is as expected. This tests just the
// printing parameters.
func TestParseParamsPrinting(t *testing.T) {
	printVal, printValSE := populatePrintScriptEntries()

	testCases := []paramtest.Parser{}

	for _, p := range []struct {
		param      string
		idx        int
		scriptPart string
	}{
		{"-after-print", printTypeP, afterSect},
		{"-a-p", printTypeP, afterSect},
		{"-after-println", printTypePln, afterSect},
		{"-a-pln", printTypePln, afterSect},
		{"-after-printf", printTypePf, afterSect},
		{"-a-pf", printTypePf, afterSect},

		{"-after-inner-print", printTypeP, afterInnerSect},
		{"-ai-p", printTypeP, afterInnerSect},
		{"-after-inner-println", printTypePln, afterInnerSect},
		{"-ai-pln", printTypePln, afterInnerSect},
		{"-after-inner-printf", printTypePf, afterInnerSect},
		{"-ai-pf", printTypePf, afterInnerSect},

		{"-before-print", printTypeP, beforeSect},
		{"-b-p", printTypeP, beforeSect},
		{"-before-println", printTypePln, beforeSect},
		{"-b-pln", printTypePln, beforeSect},
		{"-before-printf", printTypePf, beforeSect},
		{"-b-pf", printTypePf, beforeSect},

		{"-before-inner-print", printTypeP, beforeInnerSect},
		{"-bi-p", printTypeP, beforeInnerSect},
		{"-before-inner-println", printTypePln, beforeInnerSect},
		{"-bi-pln", printTypePln, beforeInnerSect},
		{"-before-inner-printf", printTypePf, beforeInnerSect},
		{"-bi-pf", printTypePf, beforeInnerSect},

		{"-exec-print", printTypeP, execSect},
		{"-print", printTypeP, execSect},
		{"-p", printTypeP, execSect},
		{"-println", printTypePln, execSect},
		{"-pln", printTypePln, execSect},
		{"-printf", printTypePf, execSect},
		{"-pf", printTypePf, execSect},
	} {
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(""),
				func(g *Gosh) {
					g.imports = []string{"fmt"}
					g.scripts[p.scriptPart] = []ScriptEntry{printValSE[p.idx]}
				}, p.param, printVal[p.idx]))
	}

	for _, tc := range testCases {
		_ = tc.Test(t)
	}
}

// TestParseParamsSnippets will use the paramtest.Parser to make sure the
// behaviour of the parameter setting is as expected. This tests just the
// snippet parameters.
func TestParseParamsSnippets(t *testing.T) {
	snippets, snippetsSE := populateSnippetScriptEntries()
	sdPath := filepath.Join("testdata", snippetsDir)

	testCases := []paramtest.Parser{}

	for _, p := range []struct {
		param      string
		scriptPart string
	}{
		{"-after-snippet", afterSect},
		{"-a-s", afterSect},
		{"-as", afterSect},

		{"-after-inner-snippet", afterInnerSect},
		{"-ai-s", afterInnerSect},
		{"-ais", afterInnerSect},

		{"-before-snippet", beforeSect},
		{"-b-s", beforeSect},
		{"-bs", beforeSect},

		{"-before-inner-snippet", beforeInnerSect},
		{"-bi-s", beforeInnerSect},
		{"-bis", beforeInnerSect},

		{"-exec-snippet", execSect},
		{"-snippet", execSect},
		{"-e-s", execSect},
		{"-es", execSect},
	} {
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(""),
				func(g *Gosh) {
					g.scripts[p.scriptPart] = []ScriptEntry{
						snippetsSE[0],
						snippetsSE[1],
					}
					g.snippetDirs = append([]string{sdPath}, g.snippetDirs...)
				},
				"-snippet-dir", filepath.Join("testdata", snippetsDir),
				p.param, snippets[0],
				p.param, snippets[1]))
	}

	for _, tc := range testCases {
		_ = tc.Test(t)
	}
}

// TestParseParamsCmd will use the paramtest.Parser to make sure the
// behaviour of the parameter setting is as expected. This tests just the
// parameters in the 'cmd' group.
func TestParseParamsCmd(t *testing.T) {
	stmt, stmtSE := populateCodeScriptEntries()
	file, fileSE := populateFileScriptEntries(t)

	testCases := []paramtest.Parser{}

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.scripts[afterSect] = []ScriptEntry{stmtSE[0], stmtSE[1]}
		}, "-after", stmt[0], "-a", stmt[1]))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.scripts[afterSect] = []ScriptEntry{fileSE[0], fileSE[1]}
		}, "-after-file", file[0], "-a-f", file[1]))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.scripts[afterInnerSect] = []ScriptEntry{stmtSE[0], stmtSE[1]}
		}, "-after-inner", stmt[0], "-ai", stmt[1]))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.scripts[beforeSect] = []ScriptEntry{stmtSE[0], stmtSE[1]}
		}, "-before", stmt[0], "-b", stmt[1]))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.scripts[beforeSect] = []ScriptEntry{fileSE[0], fileSE[1]}
		}, "-before-file", file[0], "-b-f", file[1]))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.scripts[beforeInnerSect] = []ScriptEntry{stmtSE[0], stmtSE[1]}
		}, "-before-inner", stmt[0], "-bi", stmt[1]))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.scripts[execSect] = []ScriptEntry{stmtSE[0], stmtSE[1], stmtSE[2]}
		}, "-exec", stmt[0], "-e", stmt[1], "-c", stmt[2]))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.scripts[execSect] = []ScriptEntry{fileSE[0], fileSE[1], fileSE[2]}
		}, "-exec-file", file[0],
			"-e-f", file[1],
			"-shebang", file[2]))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.scripts[globalSect] = []ScriptEntry{stmtSE[0], stmtSE[1]}
		}, "-global", stmt[0], "-g", stmt[1]))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.scripts[globalSect] = []ScriptEntry{fileSE[0], fileSE[1]}
		}, "-global-file", file[0], "-g-f", file[1]))

	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.imports = []string{"a/b", "c/d", "e/f"}
		}, "-imports", "a/b",
			"-import", "c/d",
			"-I", "e/f"))

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Cannot find the current working directory:", err)
	}
	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
			g.localModules = map[string]string{
				"a": filepath.Join(cwd, "testdata"),
			}
		}, "-local-module", "a=>testdata"))

	{
		sdPath := filepath.Join("testdata", snippetsDir)

		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID(""), func(g *Gosh) {
				g.snippetDirs = append([]string{sdPath}, g.snippetDirs...)
			}, "-snippets-dir", sdPath))
	}

	for _, tc := range testCases {
		_ = tc.Test(t)
	}
}
