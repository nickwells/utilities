package main

import (
	"errors"
	"fmt"
	"go/token"
	"math"
	"os"
	"regexp"
	"strings"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v6/paction"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
)

const (
	paramGroupNameReadloop = "cmd-readloop"
	paramGroupNameWeb      = "cmd-web"
	paramGroupNameGosh     = "cmd-gosh"

	paramNameWPrint     = "w-print"
	paramNameSnippetDir = "snippets-dir"

	paramNameExecFile          = "exec-file"
	paramNameBeforeFile        = "before-file"
	paramNameAfterFile         = "after-file"
	paramNameInnerBeforeFile   = "inner-before-file"
	paramNameInnerAfterFile    = "inner-after-file"
	paramNameGlobalFile        = "global-file"
	paramNameGlobalPackageFile = "global-package-file"
	paramNameCopyGoFile        = "copy-go-file"

	paramNameImport              = "import"
	paramNameWorkspaceUse        = "workspace-use"
	paramNameIgnoreGoModTidyErrs = "go-mod-tidy-ignore-errors"
	paramNameDontRunGoModTidy    = "go-mod-tidy-dont-run"

	paramNameFormat        = "format"
	paramNameFormatter     = "formatter"
	paramNameFormatterArgs = "formatter-args"

	paramNameDontPopImports = "dont-populate-imports"
	paramNameImporter       = "importer"
	paramNameImporterArgs   = "importer-args"

	paramNameSetGoCmd = "set-go-cmd"

	paramNameEditScript   = "edit-program"
	paramNameEditRepeat   = "edit-repeat"
	paramNameScriptEditor = "editor"
	envVisual             = "VISUAL"
	envEditor             = "EDITOR"

	paramNameInPlaceEdit  = "in-place-edit"
	paramNameReadloop     = "run-in-readloop"
	paramNameSplitLine    = "split-line"
	paramNameSplitPattern = "split-pattern"

	paramNamePreCheck = "pre-check"

	paramNameShowFilename = "show-filename"

	paramNameSetExecName    = "set-executable-name"
	paramNameDontExec       = "dont-exec"
	paramNameNoMoreParams   = "no-more-params"
	paramNameDontLoopOnArgs = "dont-loop-on-args"

	paramNameEnv      = "env"
	paramNameClearEnv = "clear-env"

	paramNameGlobalStdin      = "global-stdin"
	paramNameBeforeStdin      = "before-stdin"
	paramNameBeforeInnerStdin = "before-inner-stdin"
	paramNameExecStdin        = "exec-stdin"
	paramNameAfterInnerStdin  = "after-inner-stdin"
	paramNameAfterStdin       = "after-stdin"
)

var stdinParamNames = []string{
	paramNameGlobalStdin,
	paramNameBeforeStdin,
	paramNameBeforeInnerStdin,
	paramNameExecStdin,
	paramNameAfterInnerStdin,
	paramNameAfterStdin,
}

var readloopParamNames = []string{
	paramNameReadloop,
	paramNameSplitLine,
	paramNameSplitPattern,
	paramNameInPlaceEdit,
}

var fileParamNames = []string{
	paramNameExecFile,
	paramNameBeforeFile,
	paramNameAfterFile,
	paramNameInnerBeforeFile,
	paramNameInnerAfterFile,
	paramNameGlobalFile,
	paramNameGlobalPackageFile,
}

var editParamNames = []string{
	paramNameEditScript,
	paramNameEditRepeat,
	paramNameScriptEditor,
}

var importerParamNames = []string{
	paramNameDontPopImports,
	paramNameImporter,
	paramNameImporterArgs,
}

var formatterParamNames = []string{
	paramNameFormat,
	paramNameFormatter,
	paramNameFormatterArgs,
}

// makeSnippetHelpText returns the standard text for the various snippet
// parameters
func makeSnippetHelpText(section string) string {
	return "insert a snippet of code from the given" +
		" filename (which must be in one of the snippets directories" +
		" or a complete pathname) into the '" + section + "' section."
}

// makePrintHelpText makes the help text for the various print... parameters
func makePrintHelpText(sect string) string {
	return "follow this with the value to be printed." +
		makeCodeSectionHelpText(" resulting print", sect)
}

// makePrintVariantHelpText makes the help text for the print... parameters
// that use Fprint functions
func makePrintVariantHelpText(varName, desc string) string {
	return "\n\n" +
		"This variant will use the Fprint family of functions," +
		" passing '" + varName + "' as the writer." +
		" Such calls can be used to print to the " + desc +
		" which is called '" + varName + "' in the generated code."
}

// makeCodeSectionHelpText makes the fragment of help text describing how
// statements appear in the given section.
func makeCodeSectionHelpText(name, sect string) string {
	return " These" + name + " statements will appear with others" +
		" in the '" + sect + "' section in the order they are given."
}

func makeFileHelpText(sect string) string {
	return "follow this with a file name (which must exist)." +
		" The contents of the file will appear with others" +
		" in the '" + sect + "' section in the order they are given."
}

// makeShebangFileHelpText makes the fragment of help text describing how
// the contents of a shebang (#!) file appear in the given section.
func makeShebangFileHelpText(sect string) string {
	return makeFileHelpText(sect) +
		"\n\n" +
		"Note that any lines at the beginning of the file starting" +
		" with '#' are removed before the rest of the file is copied in." +
		" This includes a first line starting" +
		" with '#!/path/to/gosh -...file'" +
		" allowing this to function as a Shebang file."
}

// snippetPAF generates the Post-Action func (PAF) that adds the snippet name
// to the named script.
//
// Note that we pass a pointer to the snippet name rather than the string -
// this is necessary otherwise we are passing the text value at the point the
// PAF is being generated not at the point where the parameter value is
// given.
func snippetPAF(g *gosh, sName *string, scriptName string) param.ActionFunc {
	return func(_ location.L, _ *param.ByName, _ []string) error {
		err := g.CacheSnippet(*sName)
		if err != nil {
			return err
		}

		g.AddScriptEntry(scriptName, *sName, snippetExpand)

		return nil
	}
}

// scriptPAF generates the Post-Action func (PAF) that adds the text to the
// named script.
//
// Note that we pass a pointer to the text of the code rather than the string
// - this is necessary otherwise we are passing the text value at the point
// the PAF is being generated not at the point where the parameter value is
// given.
func scriptPAF(g *gosh, text *string, scriptName string) param.ActionFunc {
	return func(_ location.L, _ *param.ByName, _ []string) error {
		g.AddScriptEntry(scriptName, *text, verbatim)
		return nil
	}
}

// stdinPAF generates the Post-Action func (PAF) that reads from os.Stdin and
// adds the resulting text into the named script.
func stdinPAF(g *gosh, scriptName string) param.ActionFunc {
	return func(_ location.L, _ *param.ByName, _ []string) error {
		g.AddScriptEntry(scriptName, "", readFromStdin)
		return nil
	}
}

// parseShebangConfig creates a temporary config file, populates it with the
// supplied config and passes it to be read as a config file
func parseShebangConfig(loc location.L, p *param.ByName, config []byte) error {
	f, err := os.CreateTemp("", "gosh-shebang-*.cfg")
	if err != nil {
		return fmt.Errorf(
			"could not create the temporary shebang config file: %w",
			err)
	}
	defer os.Remove(f.Name()) //nolint:errcheck

	_, err = f.Write(config)
	if err != nil {
		return fmt.Errorf(
			"could not write the temporary shebang config file: %w",
			err)
	}

	return param.ConfigFileActionFunc(
		loc, p, []string{"shebang-config-file", f.Name()})
}

// shebangFilePAF generates the Post-Action func (PAF) that adds the contents
// of the shebang file to the named script. If the first line starts with
// '#!' it is removed before adding the rest of the contents.
//
// Note that we pass a pointer to the name of the file rather than the string
// - this is necessary otherwise we are passing the text value at the point
// the PAF is being generated not at the point where the parameter value is
// given.
func shebangFilePAF(g *gosh, text *string, scriptName string) param.ActionFunc {
	return func(loc location.L, p *param.ByName, _ []string) error {
		script, config, err := shebangFileContents(*text)
		if err != nil {
			return err
		}

		g.AddScriptEntry(scriptName, string(script), verbatim)

		if len(config) != 0 {
			return parseShebangConfig(loc, p, config)
		}

		return nil
	}
}

// packageFilePAF generates the Post-Action func (PAF) that adds the contents
// of the package file to the named script. This will strip out any package
// or import statements at the start of the file.
//
// Note that we pass a pointer to the name of the file rather than the string
// - this is necessary otherwise we are passing the text value at the point
// the PAF is being generated not at the point where the parameter value is
// given.
func packageFilePAF(g *gosh, text *string, scriptName string) param.ActionFunc {
	return func(_ location.L, _ *param.ByName, _ []string) error {
		contents, err := packageFileContents(*text)
		if err != nil {
			return err
		}

		g.AddScriptEntry(scriptName, contents, verbatim)

		return nil
	}
}

// checkImports checks the provided import name. If the name has an embedded
// '=' then it is split into two parts and the first part must be either a
// valid Go identifier or else a single period. Both parts must be non-empty.
func checkImports(v string) error {
	id, imp, ok := strings.Cut(v, "=")
	if !ok {
		return nil
	}

	if len(id) == 0 {
		return errors.New(
			"an import of the form id=import must have a non-empty id")
	}

	if len(imp) == 0 {
		return errors.New(
			"an import of the form id=import must have a non-empty import name")
	}

	if id == "." {
		return nil
	}

	if token.IsIdentifier(id) {
		return nil
	}

	return errors.New(`"` + id + `" is not a valid Go identifier`)
}

// addSnippetParams will add the parameters in the "snippet" parameter group
func addSnippetParams(g *gosh) func(ps *param.PSet) error {
	checkStringNotEmpty := check.StringLength[string](check.ValGT(0))

	return func(ps *param.PSet) error {
		ps.Add(paramNameSnippetDir,
			psetter.PathnameListAppender{
				Value:       &g.snippetDirs,
				Expectation: filecheck.DirExists(),
				Prepend:     true,
			},
			"add a new code snippets directory. The files in these"+
				" directories contain code to be added to the script by"+
				" the '...-snippet' parameters. The directory is added at"+
				" the start of the list of snippets directories and so"+
				" will be searched before any existing directories. This"+
				" can be given multiple times and each instance will"+
				" insert another directory into the list. When finding"+
				" snippets the directories are searched in order and the"+
				" first snippet found is used.",
			param.AltNames("snippet-dir"),
			param.Attrs(param.DontShowInStdUsage),
			param.SeeAlso(paramNameSnippetList),
		)

		var snippetName string

		ps.Add("exec-snippet",
			psetter.String[string]{
				Value:  &snippetName,
				Checks: []check.String{checkStringNotEmpty},
			},
			makeSnippetHelpText(execSect),
			param.AltNames("snippet", "e-s", "es"),
			param.PostAction(snippetPAF(g, &snippetName, execSect)),
			param.SeeAlso(paramNameSnippetDir, paramNameSnippetList),
		)

		ps.Add("before-snippet",
			psetter.String[string]{
				Value:  &snippetName,
				Checks: []check.String{checkStringNotEmpty},
			},
			makeSnippetHelpText(beforeSect),
			param.AltNames("b-s", "bs"),
			param.PostAction(snippetPAF(g, &snippetName, beforeSect)),
		)

		ps.Add("inner-before-snippet",
			psetter.String[string]{
				Value:  &snippetName,
				Checks: []check.String{checkStringNotEmpty},
			},
			makeSnippetHelpText(beforeInnerSect),
			param.AltNames("before-inner-snippet",
				"ib-s", "bi-s", "ibs", "bis"),
			param.PostAction(snippetPAF(g, &snippetName, beforeInnerSect)),
		)

		ps.Add("inner-after-snippet",
			psetter.String[string]{
				Value:  &snippetName,
				Checks: []check.String{checkStringNotEmpty},
			},
			makeSnippetHelpText(afterInnerSect),
			param.AltNames("after-inner-snippet", "ia-s", "ai-s", "ias", "ais"),
			param.PostAction(snippetPAF(g, &snippetName, afterInnerSect)),
		)

		ps.Add("after-snippet",
			psetter.String[string]{
				Value:  &snippetName,
				Checks: []check.String{checkStringNotEmpty},
			},
			makeSnippetHelpText(afterSect),
			param.AltNames("a-s", "as"),
			param.PostAction(snippetPAF(g, &snippetName, afterSect)),
		)

		ps.Add("global-snippet",
			psetter.String[string]{
				Value:  &snippetName,
				Checks: []check.String{checkStringNotEmpty},
			},
			makeSnippetHelpText(globalSect),
			param.AltNames("g-s", "gs"),
			param.PostAction(snippetPAF(g, &snippetName, globalSect)),
		)

		return nil
	}
}

// addWebParams will add the parameters in the "web" parameter group
func addWebParams(g *gosh) func(ps *param.PSet) error {
	checkStringNotEmpty := check.StringLength[string](check.ValGT(0))

	return func(ps *param.PSet) error {
		ps.AddGroup(paramGroupNameWeb,
			"parameters relating to building a script as a web-server.")

		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("http-server", psetter.Bool{Value: &g.runAsWebserver},
				"run a webserver with the script code being run"+
					" within an http handler function having the"+
					" following signature"+
					"\n\n"+
					g.defaultHandlerFuncDecl()+
					"\n\n"+
					"The webserver will listen on port "+
					fmt.Sprintf("%d", dfltHTTPPort)+
					" unless the port number has been set explicitly"+
					" through the http-port parameter.",
				param.AltNames("http"),
				param.GroupName(paramGroupNameWeb),
			),
		)

		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("http-port",
				psetter.Int[int64]{
					Value: &g.httpPort,
					Checks: []check.Int64{
						check.ValGT[int64](0),
						check.ValLE[int64](math.MaxUint16),
					},
				},
				"set the port number that the webserver will listen on."+
					" Setting this will also force the script to be run"+
					" within an http handler function."+
					" Note that if you set this to a value less"+
					" than 1024 you will need to have superuser privilege.",
				param.PostAction(paction.SetVal(&g.runAsWebserver, true)),
				param.GroupName(paramGroupNameWeb),
			),
		)

		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("http-path",
				psetter.String[string]{
					Value:  &g.httpPath,
					Checks: []check.String{checkStringNotEmpty},
				},
				"set the path name (the pattern) that the webserver will"+
					" listen on. Setting this will also force the script"+
					" to be run within an http handler function.",
				param.PostAction(paction.SetVal(&g.runAsWebserver, true)),
				param.GroupName(paramGroupNameWeb),
			),
		)

		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("http-handler",
				psetter.String[string]{
					Value:  &g.httpHandler,
					Checks: []check.String{checkStringNotEmpty},
				},
				"set the handler for the web server. Setting this will"+
					" also force the program to be run as a web server."+
					" Note that no script is expected in this case as the"+
					" function is supplied here.",
				param.PostAction(paction.SetVal(&g.runAsWebserver, true)),
				param.AltNames("http-h"),
				param.GroupName(paramGroupNameWeb),
			),
		)

		var codeVal string
		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("web-print",
				psetter.String[string]{
					Value: &codeVal,
					Editor: addPrint{
						prefixes:    []string{"web-"},
						paramToCall: webPrintMap,
						needsVal:    needsValMap,
					},
				},
				makePrintHelpText(execSect)+
					makePrintVariantHelpText("_rw",
						"HTTP handler's ResponseWriter"),
				param.AltNames("web-printf", "web-println",
					"web-p", "web-pf", "web-pln"),
				param.PostAction(paction.SetVal(&g.runAsWebserver, true)),
				param.GroupName(paramGroupNameWeb),
				param.PostAction(scriptPAF(g, &codeVal, execSect)),
				param.PostAction(paction.AppendStrings(&g.imports, "fmt")),
			),
		)

		ps.AddFinalCheck(func() error {
			if len(g.scripts[execSect]) > 0 &&
				g.httpHandler != dfltHTTPHandlerName {
				return errors.New(
					"you have provided an HTTP handler but also given" +
						" lines of code to run. These lines of code will" +
						" never run")
			}

			return nil
		})

		return nil
	}
}

// addReadloopParams will add the parameters in the "readloop" parameter
// group
func addReadloopParams(g *gosh) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		var codeVal string

		ps.AddGroup(paramGroupNameReadloop,
			"parameters relating to building a script with a read-loop.")

		g.runInReadloopSetters = append(g.runInReadloopSetters,
			ps.Add(paramNameReadloop, psetter.Bool{Value: &g.runInReadLoop},
				"have the script code run within a loop that reads"+
					" from stdin one line at a time. The value of each"+
					" line can be accessed by calling '_l.Text()'. Note"+
					" that any newline will have been removed and will"+
					" need to be added back if you want to print the line."+
					"\n\n"+
					"You can give filenames to read from instead of stdin"+
					" as residual parameters"+
					" (after "+ps.TerminalParam()+").",
				param.AltNames("n"),
				param.GroupName(paramGroupNameReadloop),
				param.SeeAlso(readloopParamNames...),
			),
		)

		g.runInReadloopSetters = append(g.runInReadloopSetters,
			ps.Add(paramNameSplitLine, psetter.Bool{Value: &g.splitLine},
				"split the lines into fields around runs of whitespace"+
					" characters. The fields will be available in a slice"+
					" of strings (see the Note '"+noteVars+"')."+
					" Setting this will also force"+
					" the script to be run in a loop reading from stdin"+
					" or from a list of files.",
				param.AltNames("s", "split"),
				param.PostAction(paction.SetVal(&g.runInReadLoop, true)),
				param.GroupName(paramGroupNameReadloop),
				param.SeeAlso(readloopParamNames...),
			),
		)

		g.runInReadloopSetters = append(g.runInReadloopSetters,
			ps.Add(paramNameSplitPattern,
				psetter.String[string]{Value: &g.splitPattern},
				"change the behaviour when splitting the line into"+
					" fields. The provided string must compile into a"+
					" regular expression. Setting this will also force"+
					" the script to be run in a loop reading from stdin"+
					" or from a list of files"+
					" and for each line to be split.",
				param.AltNames("sp"),
				param.PostAction(paction.SetVal(&g.runInReadLoop, true)),
				param.PostAction(paction.SetVal(&g.splitLine, true)),
				param.GroupName(paramGroupNameReadloop),
				param.SeeAlso(readloopParamNames...),
			),
		)

		g.runInReadloopSetters = append(g.runInReadloopSetters,
			ps.Add(paramNameInPlaceEdit, psetter.Bool{Value: &g.inPlaceEdit},
				"read each file given as a residual parameter"+
					" (after "+ps.TerminalParam()+") and replace its"+
					" contents with whatever is printed to the '_w' file."+
					" The original file will be kept in a copy with the"+
					" original name and a '"+origExt+"' extension. If any"+
					" of the supplied files already has a '"+origExt+"'"+
					" copy this is an error.",
				param.AltNames("i"),
				param.PostAction(paction.SetVal(&g.runInReadLoop, true)),
				param.GroupName(paramGroupNameReadloop),
				param.SeeAlso(paramNameWPrint),
			),
		)

		writeToIPEFile := ps.Add(paramNameWPrint,
			psetter.String[string]{
				Value: &codeVal,
				Editor: addPrint{
					prefixes:    []string{"w-"},
					paramToCall: wPrintMap,
					needsVal:    needsValMap,
				},
			},
			makePrintHelpText(execSect)+
				makePrintVariantHelpText("_w",
					"output file used for in-place editing"),
			param.AltNames("w-printf", "w-println", "w-p", "w-pf", "w-pln"),
			param.PostAction(scriptPAF(g, &codeVal, execSect)),
			param.PostAction(paction.AppendStrings(&g.imports, "fmt")),
			param.GroupName(paramGroupNameReadloop),
			param.SeeAlso(paramNameInPlaceEdit),
		)

		if err := ps.SetNamedRemHandler(g, "filenames"); err != nil {
			return err
		}

		ps.AddFinalCheck(func() error {
			if len(ps.Remainder()) == 0 && g.inPlaceEdit {
				return fmt.Errorf(
					"you have given the %q parameter but no filenames have"+
						" been given (they should be supplied following %q)",
					"-"+paramNameInPlaceEdit, ps.TerminalParam())
			}

			if writeToIPEFile.HasBeenSet() && !g.inPlaceEdit {
				return fmt.Errorf(
					"you are writing to the file used when in-place editing"+
						" (through one of the %q printing parameters)"+
						" but you are not editing any files."+
						"\n\n"+
						"Give the %q parameter if you want to"+
						" edit a file in-place or else write to standard"+
						" output with a different printing parameter",
					"-"+paramNameWPrint, "-"+paramNameInPlaceEdit)
			}

			return nil
		})

		return nil
	}
}

// addStdinParams returns a func that will add parameters to the passed
// ParamSet for specifying reading the code from stdin.
func addStdinParams(g *gosh) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		var stdinCount paction.Counter
		commonOpts := []param.OptFunc{
			param.Attrs(param.CommandLineOnly),
			param.SeeAlso(stdinParamNames...),
			param.PostAction(
				stdinCount.MakeActionFunc()),
		}

		stdinParams := []struct {
			name        string
			altNames    []string
			sectionName string
		}{
			{
				name:        paramNameGlobalStdin,
				altNames:    []string{"g-stdin"},
				sectionName: globalSect,
			},
			{
				name:        paramNameBeforeStdin,
				altNames:    []string{"b-stdin"},
				sectionName: beforeSect,
			},
			{
				name:        paramNameBeforeInnerStdin,
				altNames:    []string{"bi-stdin", "b-i-stdin"},
				sectionName: beforeInnerSect,
			},
			{
				name:        paramNameExecStdin,
				altNames:    []string{"e-stdin"},
				sectionName: execSect,
			},
			{
				name:        paramNameAfterInnerStdin,
				altNames:    []string{"ai-stdin", "a-i-stdin"},
				sectionName: afterInnerSect,
			},
			{
				name:        paramNameAfterStdin,
				altNames:    []string{"a-stdin"},
				sectionName: afterSect,
			},
		}
		for _, pInfo := range stdinParams {
			ps.Add(pInfo.name,
				psetter.Nil{},
				"read code from standard input and"+
					" add it to the '"+pInfo.sectionName+"' section."+
					"\n\n"+
					"Note that only one such argument may be given"+
					" regardless of the code section it is indended for.",
				append(commonOpts,
					param.AltNames(pInfo.altNames...),
					param.PostAction(stdinPAF(g, pInfo.sectionName)))...,
			)
		}

		ps.AddFinalCheck(func() error {
			if stdinCount.Total() > 1 {
				return fmt.Errorf(
					"multiple ...-stdin parameters have been given: %s",
					stdinCount.SetBy())
			}

			return nil
		})

		return nil
	}
}

// addParams returns a func that will add parameters to the passed ParamSet
func addParams(g *gosh) func(ps *param.PSet) error {
	checkStringNotEmpty := check.StringLength[string](check.ValGT(0))

	return func(ps *param.PSet) error {
		var codeVal string

		var fileName string

		// Exec section params

		ps.Add("exec", psetter.String[string]{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", execSect),
			param.AltNames("e", "c", "code"), // python and bash use '-c'
			param.PostAction(scriptPAF(g, &codeVal, execSect)),
			param.ValueName("Go-code"),
		)

		ps.Add(paramNameExecFile,
			psetter.Pathname{
				Value:       &fileName,
				Expectation: filecheck.FileNonEmpty(),
			},
			makeShebangFileHelpText(execSect),
			param.AltNames("shebang", "e-f"),
			param.PostAction(shebangFilePAF(g, &fileName, execSect)),
			param.SeeAlso(fileParamNames...),
			param.SeeNote(noteShebangScripts),
		)

		ps.Add("exec-print",
			psetter.String[string]{
				Value: &codeVal,
				Editor: addPrint{
					prefixes:    []string{"exec-", "e-"},
					paramToCall: stdPrintMap,
					needsVal:    needsValMap,
				},
			},
			makePrintHelpText(execSect),
			param.AltNames("exec-printf", "exec-println",
				"print", "printf", "println",
				"e-p", "e-pf", "e-pln",
				"p", "pf", "pln"),
			param.PostAction(scriptPAF(g, &codeVal, execSect)),
			param.PostAction(paction.AppendStrings(&g.imports, "fmt")),
		)

		// Before section params

		ps.Add("before", psetter.String[string]{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", beforeSect),
			param.AltNames("b"),
			param.PostAction(scriptPAF(g, &codeVal, beforeSect)),
			param.ValueName("Go-code"),
		)

		ps.Add(paramNameBeforeFile,
			psetter.Pathname{
				Value:       &fileName,
				Expectation: filecheck.FileNonEmpty(),
			},
			makeShebangFileHelpText(beforeSect),
			param.AltNames("b-f"),
			param.PostAction(shebangFilePAF(g, &fileName, beforeSect)),
			param.SeeAlso(fileParamNames...),
			param.SeeNote(noteShebangScripts),
		)

		ps.Add("before-print",
			psetter.String[string]{
				Value: &codeVal,
				Editor: addPrint{
					prefixes:    []string{"before-", "b-"},
					paramToCall: stdPrintMap,
					needsVal:    needsValMap,
				},
			},
			makePrintHelpText(beforeSect),
			param.AltNames("before-printf", "before-println",
				"b-p", "b-pf", "b-pln"),
			param.PostAction(scriptPAF(g, &codeVal, beforeSect)),
			param.PostAction(paction.AppendStrings(&g.imports, "fmt")),
		)

		// Inner-Before section params

		ps.Add("inner-before", psetter.String[string]{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", beforeInnerSect),
			param.AltNames("before-inner", "ib", "bi"),
			param.PostAction(scriptPAF(g, &codeVal, beforeInnerSect)),
			param.ValueName("Go-code"),
		)

		ps.Add(paramNameInnerBeforeFile,
			psetter.Pathname{
				Value:       &fileName,
				Expectation: filecheck.FileNonEmpty(),
			},
			makeShebangFileHelpText(beforeInnerSect),
			param.AltNames("before-inner-file", "ib-f", "bi-f"),
			param.PostAction(shebangFilePAF(g, &fileName, beforeInnerSect)),
			param.SeeAlso(fileParamNames...),
			param.SeeNote(noteShebangScripts),
		)

		ps.Add("inner-before-print",
			psetter.String[string]{
				Value: &codeVal,
				Editor: addPrint{
					prefixes: []string{
						"inner-before-", "ib-",
						"before-inner-", "bi-",
					},
					paramToCall: stdPrintMap,
					needsVal:    needsValMap,
				},
			},
			makePrintHelpText(beforeInnerSect),
			param.AltNames(
				"inner-before-printf", "inner-before-println",
				"before-inner-printf", "before-inner-println",
				"before-inner-print",
				"ib-p", "ib-pf", "ib-pln",
				"bi-p", "bi-pf", "bi-pln"),
			param.PostAction(scriptPAF(g, &codeVal, beforeInnerSect)),
			param.PostAction(paction.AppendStrings(&g.imports, "fmt")),
		)

		// Inner-After section params

		ps.Add("inner-after", psetter.String[string]{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", afterInnerSect),
			param.AltNames("after-inner", "ia", "ai"),
			param.PostAction(scriptPAF(g, &codeVal, afterInnerSect)),
			param.ValueName("Go-code"),
		)

		ps.Add(paramNameInnerAfterFile,
			psetter.Pathname{
				Value:       &fileName,
				Expectation: filecheck.FileNonEmpty(),
			},
			makeShebangFileHelpText(afterInnerSect),
			param.AltNames("after-inner-file", "ia-f", "ai-f"),
			param.PostAction(shebangFilePAF(g, &fileName, afterInnerSect)),
			param.SeeAlso(fileParamNames...),
			param.SeeNote(noteShebangScripts),
		)

		ps.Add("inner-after-print",
			psetter.String[string]{
				Value: &codeVal,
				Editor: addPrint{
					prefixes: []string{
						"inner-after-", "ia-",
						"after-inner-", "ai-",
					},
					paramToCall: stdPrintMap,
					needsVal:    needsValMap,
				},
			},
			makePrintHelpText(afterInnerSect),
			param.AltNames(
				"inner-after-printf", "inner-after-println",
				"after-inner-printf", "after-inner-println",
				"after-inner-print",
				"ia-p", "ia-pf", "ia-pln",
				"ai-p", "ai-pf", "ai-pln"),
			param.PostAction(scriptPAF(g, &codeVal, afterInnerSect)),
			param.PostAction(paction.AppendStrings(&g.imports, "fmt")),
		)

		// Inner-After section params

		ps.Add("after", psetter.String[string]{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", afterSect),
			param.AltNames("a"),
			param.PostAction(scriptPAF(g, &codeVal, afterSect)),
			param.ValueName("Go-code"),
		)

		ps.Add(paramNameAfterFile,
			psetter.Pathname{
				Value:       &fileName,
				Expectation: filecheck.FileNonEmpty(),
			},
			makeShebangFileHelpText(afterSect),
			param.AltNames("a-f"),
			param.PostAction(shebangFilePAF(g, &fileName, afterSect)),
			param.SeeAlso(fileParamNames...),
			param.SeeNote(noteShebangScripts),
		)

		ps.Add("after-print",
			psetter.String[string]{
				Value: &codeVal,
				Editor: addPrint{
					prefixes:    []string{"after-", "a-"},
					paramToCall: stdPrintMap,
					needsVal:    needsValMap,
				},
			},
			makePrintHelpText(afterSect),
			param.AltNames("after-printf", "after-println",
				"a-p", "a-pf", "a-pln"),
			param.PostAction(scriptPAF(g, &codeVal, afterSect)),
			param.PostAction(paction.AppendStrings(&g.imports, "fmt")),
		)

		// Global section params

		ps.Add("global", psetter.String[string]{Value: &codeVal},
			"follow this with Go code."+
				" For instance, functions that you might want to call from"+
				" several places, global variables or data types."+
				makeCodeSectionHelpText("", globalSect),
			param.AltNames("g"),
			param.PostAction(scriptPAF(g, &codeVal, globalSect)),
			param.ValueName("Go-code"),
		)

		ps.Add(paramNameGlobalFile,
			psetter.Pathname{
				Value:       &fileName,
				Expectation: filecheck.FileNonEmpty(),
			},
			makeShebangFileHelpText(globalSect),
			param.AltNames("g-f"),
			param.PostAction(shebangFilePAF(g, &fileName, globalSect)),
			param.SeeAlso(fileParamNames...),
			param.SeeNote(noteShebangScripts),
		)

		ps.Add(paramNameGlobalPackageFile,
			psetter.Pathname{
				Value:       &fileName,
				Expectation: filecheck.FileNonEmpty(),
			},
			makeFileHelpText(globalSect)+
				"\n\n"+
				"Note that the file is expected to be a file from another"+
				" Go package and should contain at least a 'package'"+
				" statement. Any 'package' statement and any 'import'"+
				" statements are removed before the file is inserted"+
				" into the gosh-generated program.",
			param.AltNames("global-file-package", "g-f-p", "g-p-f"),
			param.PostAction(packageFilePAF(g, &fileName, globalSect)),
			param.SeeAlso(append(fileParamNames, paramNameCopyGoFile)...),
		)

		ps.Add("global-print",
			psetter.String[string]{
				Value: &codeVal,
				Editor: addPrint{
					prefixes:    []string{"global-", "g-"},
					paramToCall: stdPrintMap,
					needsVal:    needsValMap,
				},
			},
			makePrintHelpText(globalSect),
			param.AltNames("global-printf", "global-println",
				"g-p", "g-pf", "g-pln"),
			param.PostAction(scriptPAF(g, &codeVal, globalSect)),
			param.PostAction(paction.AppendStrings(&g.imports, "fmt")),
		)

		// Env params

		ps.Add(paramNameEnv,
			psetter.StrListAppender[string]{
				Value: &g.env,
				Checks: []check.String{
					checkStringNotEmpty,
					check.StringContains[string]("="),
				},
			},
			"provide values to be added to the environment when the"+
				" generated program is run.",
			param.SeeAlso(paramNameClearEnv),
			param.ValueName("key=val"),
		)

		ps.Add(paramNameClearEnv,
			psetter.Bool{Value: &g.clearEnv},
			"clear environment before the generated program is run."+
				" Unfortunately a completely empty environment will be"+
				" automatically populated by the Go exec package with"+
				" the environment of the calling program. Consequently "+
				"'_' is always set to the full path of the generated"+
				" executable.",
			param.SeeAlso(paramNameEnv),
		)

		// Miscellaneous other params

		ps.Add(paramNameImport,
			psetter.StrListAppender[string]{
				Value: &g.imports,
				Checks: []check.String{
					checkStringNotEmpty,
					checkImports,
				},
			},
			"provide any explicit imports."+
				"\n\n"+
				"Note that the import path can be given with"+
				" a leading ...= in which case the part before"+
				" the '=' must be a '.' or a valid Go identifier"+
				" and is used as an alias for the package name.",
			param.AltNames("imports", "I"),
			param.ValueName("package"),
		)

		ps.Add("local-module",
			ModuleMapSetter{
				Value: &g.localModules,
			},
			"the name and mapping of a local module."+
				" This will add a replace directive in the 'go.mod' file.",
			param.Attrs(param.DontShowInStdUsage),
			param.AltNames("replace", "mod-replace"),
		)

		ps.Add(paramNameWorkspaceUse,
			psetter.PathnameListAppender{
				Value:         &g.workspace,
				Expectation:   filecheck.DirExists(),
				ForceAbsolute: true,
			},
			"the name of a module to be added to the 'go.work' file."+
				"\n\n"+
				"Note that if a workspace use directive is given"+
				" the version of Go that you are using must have"+
				" the 'go work ...' command available;"+
				" this command was added in Go 1.18."+
				"\n\n"+
				"Note that the local changes to the other module"+
				" may result in errors from the 'go mod tidy' command."+
				" These would normally cause gosh to abort but you"+
				" can suppress this behaviour with"+
				" the '"+paramNameIgnoreGoModTidyErrs+"' parameter",
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.AltNames("go-work-use"),
			param.SeeAlso(paramNameIgnoreGoModTidyErrs),
		)

		ps.Add(paramNameCopyGoFile,
			psetter.PathnameListAppender{
				Value:       &g.copyGoFiles,
				Expectation: filecheck.FileExists(),
				Checks: []check.String{
					check.StringHasSuffix[string](".go"),
				},
			},
			"add a file to the list of Go files to be copied into"+
				" the gosh directory before building the program. Note"+
				" that the file must exist and will be copied with a name"+
				" guaranteeing uniqueness so you don't need to worry"+
				" about the files being copied having different names."+
				" If the file is not already in package 'main' the"+
				" package name will be changed so it is."+
				"\n\n"+
				"This differs from "+paramNameGlobalPackageFile+" behaviour"+
				" in that it will generate a separate file rather than"+
				" embedding the code in the same file as the rest of the"+
				" gosh code. It also leaves the import statement unchanged"+
				" which may be convenient.",
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.SeeAlso(paramNameGlobalPackageFile),
		)

		ps.Add(paramNameDontLoopOnArgs,
			psetter.Bool{
				Value: &g.skipArgLoop,
			},
			"don't loop over the program arguments."+
				"\n\n"+
				"Without this any arguments to the generated program"+
				" (after "+ps.TerminalParam()+") are processed in a loop.",
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.AltNames("skip-arg-loop", "no-arg-loop"),
			param.SeeAlso(paramNameReadloop),
		)

		ps.Add(paramNameNoMoreParams,
			psetter.Nil{},
			"don't take any more arguments to gosh."+
				"\n\n"+
				"If this is given then any subsequent arguments will be"+
				" passed to the generated program. From the commant line"+
				" this is more conveniently achieved by given the"+
				" standard terminal parameter"+
				" ('"+param.DfltTerminalParam+"') but in shebang scripts"+
				" this parameter can be given as a script parameter at"+
				" the start of the shebang file. Then any parameters"+
				" to the script will be used as parameters to the"+
				" generated program rather than to gosh.",
			param.Attrs(param.IsTerminalParam|param.DontShowInStdUsage),
			param.AltNames("no-more-args"),
			param.SeeNote(noteShebangScriptParams),
		)

		// Final checks

		ps.AddFinalCheck(func() error {
			if g.runAsWebserver && g.runInReadLoop {
				errStr := "gosh cannot run in a read-loop" +
					" and run as a webserver at the same time." +
					" Parameters set at:"

				for _, p := range g.runAsWebserverSetters {
					for _, w := range p.WhereSet() {
						errStr += "\n\t" + w
					}
				}

				for _, p := range g.runInReadloopSetters {
					for _, w := range p.WhereSet() {
						errStr += "\n\t" + w
					}
				}

				return errors.New(errStr)
			}

			return nil
		})

		return nil
	}
}

// addGoshParams returns a function that adds the parameters which control
// the behaviour of the gosh command rather than the program it generates.
func addGoshParams(g *gosh) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		ps.AddGroup(paramGroupNameGosh,
			"parameters controlling the behaviour of the gosh command"+
				" rather than the program it generates.")

		ps.Add(paramNameShowFilename,
			psetter.Bool{Value: &g.dontCleanupUserChoice},
			"show the filename where the program has been constructed."+
				" This will also prevent the generated code from being"+
				" cleared after execution has successfully completed,"+
				" the assumption being that if you want to know the"+
				" filename you will also want to examine its contents.",
			param.AltNames("show-file", "keep"),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.GroupName(paramGroupNameGosh),
		)

		execNameRE := regexp.MustCompile(`^[a-zA-Z][-a-zA-Z0-9+._]*$`)

		const maxExecNameLen = 50

		ps.Add(paramNameSetExecName,
			psetter.String[string]{
				Value: &g.execName,
				Checks: []check.String{
					check.StringLength[string](
						check.ValBetween(1, maxExecNameLen)),
					check.StringMatchesPattern[string](execNameRE,
						"The program name must start with a letter and"+
							" be followed by zero or more"+
							" letters, digits,"+
							" dashes, plus-signs,"+
							" underscores or dots"),
				},
			},
			"set the name of the program to be generated. This will"+
				" also prevent the generated code from being cleared after"+
				" execution has successfully completed, the assumption being"+
				" that if you have set the program name you will want to"+
				" preserve it."+
				"\n\n"+
				fmt.Sprintf(
					"The program name must be between 1 and %d characters long",
					maxExecNameLen),
			param.AltNames(
				"set-program-name", "program-name", "executable-name"),
			param.PostAction(paction.SetVal(&g.dontCleanupUserChoice, true)),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add(paramNameDontExec, psetter.Bool{Value: &g.dontRun},
			"don't run the generated code - this prevents the generated"+
				" code from being cleared and forces"+
				" the "+paramNameShowFilename+
				" parameter to true. This can be"+
				" useful if you have completed the work you were using"+
				" the generated code for and now want to save the file "+
				" for future use.",
			param.AltNames("dont-run", "no-exec", "no-run"),
			param.PostAction(paction.SetVal(&g.dontCleanupUserChoice, true)),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.GroupName(paramGroupNameGosh),
		)

		goCmdName := gogen.GetGoCmdName()
		ps.Add(paramNameSetGoCmd, psetter.String[string]{Value: &goCmdName},
			"the name of the Go command to use."+
				" Note that it must be an executable program either"+
				" in your PATH or else as a pathname",
			param.PostAction(
				func(_ location.L, _ *param.ByName, _ []string) error {
					return gogen.SetGoCmdName(goCmdName)
				}),
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
		)

		// Import-populator params

		ps.Add(paramNameImporter, psetter.String[string]{Value: &g.importPopulator},
			"the name of the command to run in order to populate the"+
				" import statements. If no"+
				" value is given then one of "+importerCmds()+
				" will be used."+
				" They are checked in the order given and the first"+
				" which is installed and executable will be used."+
				" Note that you should give just the executable"+
				" name with this parameter and any arguments separately.",
			param.PostAction(paction.SetVal(&g.importPopulatorSet, true)),
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
			param.SeeAlso(importerParamNames...),
		)

		ps.Add(paramNameImporterArgs,
			psetter.StrList[string]{Value: &g.importPopulatorArgs},
			"the arguments to pass to the import populating command."+
				" Note that the final argument will always be the name"+
				" of the generated program.",
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
			param.SeeAlso(importerParamNames...),
		)

		ps.Add(paramNameDontPopImports,
			psetter.Bool{Value: &g.dontPopulateImports},
			"dont automatically generate the import statements"+
				" - if this is set the"+
				" generated code will not have the import statements"+
				" automatically populated."+
				"\n\n"+
				"This can be useful with shebang scripts where you"+
				" want the marginal performance improvement. An"+
				" additional advantage is that the script will run"+
				" successfully even if you don't have access to"+
				" any of the default import populators."+
				"\n\n"+
				"Note that you will have to give any missing imports on the"+
				" command line using the "+paramNameImport+" parameter.",
			param.AltNames(
				"dont-auto-import", "no-auto-import", "no-import-gen"),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.GroupName(paramGroupNameGosh),
			param.SeeAlso(paramNameImport),
			param.SeeAlso(importerParamNames...),
		)

		// Formatter params

		ps.Add(paramNameFormatter, psetter.String[string]{Value: &g.formatter},
			"the name of the formatter command to run. If no"+
				" value is given and the "+paramNameFormat+
				" is set then one of "+formatterCmds()+
				" will be used."+
				" They are checked in the order given and the first"+
				" which is installed and executable will be used."+
				" Note that you should give just the executable"+
				" name with this parameter and any arguments separately.",
			param.PostAction(paction.SetVal(&g.formatterSet, true)),
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
			param.SeeAlso(formatterParamNames...),
		)

		ps.Add(paramNameFormatterArgs,
			psetter.StrList[string]{Value: &g.formatterArgs},
			"the arguments to pass to the formatter command. Note that"+
				" the final argument will always be the name of the"+
				" generated program.",
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
			param.SeeAlso(formatterParamNames...),
		)

		ps.Add(paramNameFormat,
			psetter.Bool{Value: &g.formatCode},
			"format the generated code - unless this is set the"+
				" generated code will not be formatted."+
				"\n\n"+
				"You might want to format the code if you are going"+
				" to keep the generated code for later reuse or if"+
				" you are going to edit it in an edit loop.",
			param.AltNames("fmt"),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.GroupName(paramGroupNameGosh),
			param.SeeAlso(paramNameImport),
			param.SeeAlso(formatterParamNames...),
		)

		// Edit params

		ps.Add(paramNameEditScript, psetter.Bool{Value: &g.edit},
			"edit the generated code just before running it.",
			param.AltNames("edit"),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.SeeAlso(editParamNames...),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add(paramNameEditRepeat, psetter.Bool{Value: &g.editRepeat},
			"after the program has run, you will be asked if you want"+
				" to repeat the edit/build/run loop.",
			param.PostAction(paction.SetVal(&g.edit, true)),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.SeeAlso(editParamNames...),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add(paramNameScriptEditor,
			psetter.String[string]{Value: &g.editorParam},
			"This will give the name of an editor to use for editing"+
				" your program. Note that this does not force the file to"+
				" be edited so you can set this in a configuration"+
				" file. Its validity is only checked though when you use it."+
				"\n\n"+
				"If this parameter is not given or if the resulting"+
				" program is not executable, the editor will be taken"+
				" from the environment variables:"+
				" '"+envVisual+"' and"+
				" '"+envEditor+"' in that order",
			param.SeeAlso(editParamNames...),
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
		)

		// go mod tidy params

		ps.Add(paramNameIgnoreGoModTidyErrs,
			psetter.Bool{
				Value: &g.ignoreGoModTidyErrs,
			},
			"don't abort when the 'go mod tidy' command reports errors;"+
				" the error message, if any, will still be written to stderr."+
				"\n\n"+
				"This should only be set when a workspace is in use",
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.AltNames("ignore-go-mod-tidy-errors"),
			param.SeeAlso(paramNameWorkspaceUse),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add(paramNameDontRunGoModTidy,
			psetter.Bool{
				Value: &g.dontRunGoModTidy,
			},
			"don't run the 'go mod tidy' command"+
				"\n\n"+
				"This should only be set if you know that no"+
				" non-standard packages are being used",
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.AltNames("dont-run-go-mod-tidy", "no-go-mod-tidy"),
			param.SeeAlso(paramNameWorkspaceUse),
			param.GroupName(paramGroupNameGosh),
		)

		// Miscellaneous params

		ps.Add("build-arg",
			psetter.StrListAppender[string]{Value: &g.buildArgs},
			"add an argument to pass to the go build command.",
			param.AltNames("build-args", "args-build", "b-args", "b-arg"),
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add("add-comments", psetter.Bool{Value: &g.addComments},
			"add end-of-line comments to show the lines of code"+
				" generated by gosh.",
			param.AltNames("add-comment",
				"comments", "comment",
				"gosh-comments"),
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add("base-temp-dir",
			psetter.Pathname{
				Value:       &g.baseTempDir,
				Expectation: filecheck.DirExists(),
			},
			"set the directory where the temporary directories in which"+
				" the gosh program will be generated",
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add(paramNamePreCheck,
			psetter.Bool{
				Value: &g.preCheck,
			},
			"don't run any code but instead check that the commands that"+
				" gosh needs are available and recommend any fixes that"+
				" should be made",
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
		)

		// Final checks

		ps.AddFinalCheck(func() error {
			if g.formatCode && !g.edit {
				g.dontCleanupUserChoice = true
			}

			return nil
		})

		return nil
	}
}
