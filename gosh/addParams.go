package main

import (
	"errors"
	"fmt"
	"go/token"
	"regexp"
	"strings"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
	"github.com/nickwells/utilities/internal/stdparams"
)

const (
	paramGroupNameReadloop = "cmd-readloop"
	paramGroupNameWeb      = "cmd-web"
	paramGroupNameGosh     = "cmd-gosh"

	paramNameInPlaceEdit     = "in-place-edit"
	paramNameWPrint          = "w-print"
	paramNameSnippetDir      = "snippets-dir"
	paramNameExecFile        = "exec-file"
	paramNameBeforeFile      = "before-file"
	paramNameAfterFile       = "after-file"
	paramNameInnerBeforeFile = "inner-before-file"
	paramNameInnerAfterFile  = "inner-after-file"
	paramNameGlobalFile      = "global-file"

	paramNameImport              = "import"
	paramNameWorkspaceUse        = "workspace-use"
	paramNameIgnoreGoModTidyErrs = "ignore-go-mod-tidy-errors"

	paramNameDontFormat = "dont-format"

	paramNameSetGoCmd = "set-go-cmd"

	paramNameEditScript   = "edit-program"
	paramNameEditRepeat   = "edit-repeat"
	paramNameScriptEditor = "editor"
	envVisual             = "VISUAL"
	envEditor             = "EDITOR"
)

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

// makeShebangFileHelpText makes the fragment of help text describing how
// the contents of a shebang (#!) file appear in the given section.
func makeShebangFileHelpText(sect string) string {
	return "follow this with a file name (which must exist)." +
		" The contents of the file will appear with others" +
		" in the '" + sect + "' section in the order they are given." +
		"\n\n" +
		"Note that if the first line of the file starts with '#!' then" +
		" that first line is removed before the rest of the file is copied" +
		" in. This is to allow gosh to be used as an interpreter in Linux" +
		" Shebang files."
}

// snippetPAF generates the Post-Action func (PAF) that adds the snippet name
// to the named script.
//
// Note that we pass a pointer to the snippet name rather than the string -
// this is necessary otherwise we are passing the text value at the point the
// PAF is being generated not at the point where the parameter value is
// given.
func snippetPAF(g *Gosh, sName *string, scriptName string) param.ActionFunc {
	return func(_ location.L, _ *param.ByName, _ []string) error {
		_, err := g.snippets.Add(g.snippetDirs, *sName)
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
func scriptPAF(g *Gosh, text *string, scriptName string) param.ActionFunc {
	return func(_ location.L, _ *param.ByName, _ []string) error {
		g.AddScriptEntry(scriptName, *text, verbatim)
		return nil
	}
}

// shebangFilePAF generates the Post-Action func (PAF) that adds the contents
// of the shebang file to the named script. If the first line starts with
// '#!' it is removed before adding the rest of the contents.
//
// Note that we pass a pointer to the name of the file rather than the string
// - this is necessary otherwise we are passing the text value at the point
// the PAF is being generated not at the point where the parameter value is
// given.
func shebangFilePAF(g *Gosh, text *string, scriptName string) param.ActionFunc {
	return func(_ location.L, _ *param.ByName, _ []string) error {
		contents, err := shebangFileContents(*text)
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
func addSnippetParams(g *Gosh) func(ps *param.PSet) error {
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
			psetter.String{
				Value:  &snippetName,
				Checks: []check.String{check.StringLength[string](check.ValGT(0))},
			},
			makeSnippetHelpText(execSect),
			param.AltNames("snippet", "e-s", "es"),
			param.PostAction(snippetPAF(g, &snippetName, execSect)),
			param.SeeAlso(paramNameSnippetDir, paramNameSnippetList),
		)

		ps.Add("before-snippet",
			psetter.String{
				Value:  &snippetName,
				Checks: []check.String{check.StringLength[string](check.ValGT(0))},
			},
			makeSnippetHelpText(beforeSect),
			param.AltNames("b-s", "bs"),
			param.PostAction(snippetPAF(g, &snippetName, beforeSect)),
		)

		ps.Add("inner-before-snippet",
			psetter.String{
				Value:  &snippetName,
				Checks: []check.String{check.StringLength[string](check.ValGT(0))},
			},
			makeSnippetHelpText(beforeInnerSect),
			param.AltNames("before-inner-snippet",
				"ib-s", "bi-s", "ibs", "bis"),
			param.PostAction(snippetPAF(g, &snippetName, beforeInnerSect)),
		)

		ps.Add("inner-after-snippet",
			psetter.String{
				Value:  &snippetName,
				Checks: []check.String{check.StringLength[string](check.ValGT(0))},
			},
			makeSnippetHelpText(afterInnerSect),
			param.AltNames("after-inner-snippet", "ia-s", "ai-s", "ias", "ais"),
			param.PostAction(snippetPAF(g, &snippetName, afterInnerSect)),
		)

		ps.Add("after-snippet",
			psetter.String{
				Value:  &snippetName,
				Checks: []check.String{check.StringLength[string](check.ValGT(0))},
			},
			makeSnippetHelpText(afterSect),
			param.AltNames("a-s", "as"),
			param.PostAction(snippetPAF(g, &snippetName, afterSect)),
		)

		ps.Add("global-snippet",
			psetter.String{
				Value:  &snippetName,
				Checks: []check.String{check.StringLength[string](check.ValGT(0))},
			},
			makeSnippetHelpText(globalSect),
			param.AltNames("g-s", "gs"),
			param.PostAction(snippetPAF(g, &snippetName, globalSect)),
		)

		return nil
	}
}

// addWebParams will add the parameters in the "web" parameter group
func addWebParams(g *Gosh) func(ps *param.PSet) error {
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
					" The webserver will listen on port "+
					fmt.Sprintf("%d", dfltHTTPPort)+
					" unless the port number has been set explicitly"+
					" through the http-port parameter.",
				param.AltNames("http"),
				param.GroupName(paramGroupNameWeb),
			),
		)

		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("http-port",
				psetter.Int64{
					Value: &g.httpPort,
					Checks: []check.Int64{
						check.ValGT[int64](0),
						check.ValLT[int64]((1 << 16) + 1),
					},
				},
				"set the port number that the webserver will listen on."+
					" Setting this will also force the script to be run"+
					" within an http handler function."+
					" Note that if you set this to a value less"+
					" than 1024 you will need to have superuser privilege.",
				param.PostAction(paction.SetBool(&g.runAsWebserver, true)),
				param.GroupName(paramGroupNameWeb),
			),
		)

		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("http-path",
				psetter.String{
					Value: &g.httpPath,
					Checks: []check.String{
						check.StringLength[string](check.ValGT(0)),
					},
				},
				"set the path name (the pattern) that the webserver will"+
					" listen on. Setting this will also force the script"+
					" to be run within an http handler function.",
				param.PostAction(paction.SetBool(&g.runAsWebserver, true)),
				param.GroupName(paramGroupNameWeb),
			),
		)

		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("http-handler",
				psetter.String{
					Value: &g.httpHandler,
					Checks: []check.String{
						check.StringLength[string](check.ValGT(0)),
					},
				},
				"set the handler for the web server. Setting this will"+
					" also force the program to be run as a web server."+
					" Note that no script is expected in this case as the"+
					" function is supplied here.",
				param.PostAction(paction.SetBool(&g.runAsWebserver, true)),
				param.AltNames("http-h"),
				param.GroupName(paramGroupNameWeb),
			),
		)

		var codeVal string
		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("web-print",
				psetter.String{
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
				param.PostAction(paction.SetBool(&g.runAsWebserver, true)),
				param.GroupName(paramGroupNameWeb),
				param.PostAction(scriptPAF(g, &codeVal, execSect)),
				param.PostAction(paction.AppendStrings(&g.imports, "fmt")),
			),
		)

		ps.AddFinalCheck(func() error {
			if len(g.scripts[execSect]) > 0 &&
				g.httpHandler != dfltHTTPHandlerName {
				return errors.New(
					"You have provided an HTTP handler but also given" +
						" lines of code to run. These lines of code will" +
						" never run.")
			}

			return nil
		})

		return nil
	}
}

// addReadloopParams will add the parameters in the "readloop" parameter
// group
func addReadloopParams(g *Gosh) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		var codeVal string

		ps.AddGroup(paramGroupNameReadloop,
			"parameters relating to building a script with a read-loop.")

		g.runInReadloopSetters = append(g.runInReadloopSetters,
			ps.Add("run-in-readloop", psetter.Bool{Value: &g.runInReadLoop},
				"have the script code being run within a loop that reads"+
					" from stdin one a line at a time. The value of each"+
					" line can be accessed by calling 'line.Text()'. Note"+
					" that any newline will have been removed and will"+
					" need to be added back if you want to print the line.",
				param.AltNames("n"),
				param.GroupName(paramGroupNameReadloop),
			),
		)

		g.runInReadloopSetters = append(g.runInReadloopSetters,
			ps.Add("split-line", psetter.Bool{Value: &g.splitLine},
				"split the lines into fields around runs of whitespace"+
					" characters. The fields will be available in a slice"+
					" of strings (see the Note '"+noteVars+"')."+
					" Setting this will also force"+
					" the script to be run in a loop reading from stdin.",
				param.AltNames("s"),
				param.PostAction(paction.SetBool(&g.runInReadLoop, true)),
				param.GroupName(paramGroupNameReadloop),
			),
		)

		g.runInReadloopSetters = append(g.runInReadloopSetters,
			ps.Add("split-pattern", psetter.String{Value: &g.splitPattern},
				"change the behaviour when splitting the line into"+
					" fields. The provided string must compile into a"+
					" regular expression. Setting this will also force"+
					" the script to be run in a loop reading from stdin"+
					" and for each line to be split.",
				param.AltNames("sp"),
				param.PostAction(paction.SetBool(&g.runInReadLoop, true)),
				param.PostAction(paction.SetBool(&g.splitLine, true)),
				param.GroupName(paramGroupNameReadloop),
			),
		)

		g.runInReadloopSetters = append(g.runInReadloopSetters,
			ps.Add(paramNameInPlaceEdit, psetter.Bool{Value: &g.inPlaceEdit},
				"read each file given as a residual parameter"+
					" (after "+ps.TerminalParam()+") and replace its"+
					" contents with whatever is printed to the '_w' file."+
					" The original file will be kept in a copy with the"+
					" original name and  '.orig' extension. If any of the"+
					" supplied files already has a '.orig' copy then the"+
					" file will be reported and execution will stop",
				param.AltNames("i"),
				param.PostAction(paction.SetBool(&g.runInReadLoop, true)),
				param.GroupName(paramGroupNameReadloop),
				param.SeeAlso(paramNameWPrint),
			),
		)

		writeToIPEFile := ps.Add(paramNameWPrint,
			psetter.String{
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
					"You have given the %q parameter but no filenames have"+
						" been given (they should be supplied following %q)",
					"-"+paramNameInPlaceEdit, ps.TerminalParam())
			}

			if writeToIPEFile.HasBeenSet() && !g.inPlaceEdit {
				return fmt.Errorf(
					"You are writing to the file used when in-place editing"+
						" (through one of the %q printing parameters)"+
						" but you are not editing any files."+
						"\n\n"+
						"Give the %q parameter if you want to"+
						" edit a file in-place or else write to standard"+
						" output with a different printing parameter.",
					"-"+paramNameWPrint, "-"+paramNameInPlaceEdit)
			}

			return nil
		})

		return nil
	}
}

// addParams returns a func that will add parameters to the passed ParamSet
func addParams(g *Gosh) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		var codeVal string
		var fileName string

		ps.Add("exec", psetter.String{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", execSect),
			param.AltNames("e", "c"), // python and bash use '-c'
			param.PostAction(scriptPAF(g, &codeVal, execSect)),
		)

		ps.Add(paramNameExecFile,
			psetter.Pathname{
				Value:       &fileName,
				Expectation: filecheck.FileNonEmpty(),
			},
			makeShebangFileHelpText(execSect),
			param.AltNames("shebang", "e-f"),
			param.PostAction(shebangFilePAF(g, &fileName, execSect)),
			param.SeeAlso(
				paramNameBeforeFile, paramNameAfterFile, paramNameGlobalFile),
			param.SeeNote(noteShebangScripts),
		)

		ps.Add("exec-print",
			psetter.String{
				Value: &codeVal,
				Editor: addPrint{
					prefixes:    []string{"exec-"},
					paramToCall: stdPrintMap,
					needsVal:    needsValMap,
				},
			},
			makePrintHelpText(execSect),
			param.AltNames("print", "printf", "println", "p", "pf", "pln"),
			param.PostAction(scriptPAF(g, &codeVal, execSect)),
			param.PostAction(paction.AppendStrings(&g.imports, "fmt")),
		)

		ps.Add("before", psetter.String{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", beforeSect),
			param.AltNames("b"),
			param.PostAction(scriptPAF(g, &codeVal, beforeSect)),
		)

		ps.Add("inner-before", psetter.String{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", beforeInnerSect),
			param.AltNames("before-inner", "ib", "bi"),
			param.PostAction(scriptPAF(g, &codeVal, beforeInnerSect)),
		)

		ps.Add(paramNameBeforeFile,
			psetter.Pathname{
				Value:       &fileName,
				Expectation: filecheck.FileNonEmpty(),
			},
			makeShebangFileHelpText(beforeSect),
			param.AltNames("b-f"),
			param.PostAction(shebangFilePAF(g, &fileName, beforeSect)),
			param.SeeAlso(
				paramNameExecFile, paramNameAfterFile, paramNameGlobalFile),
			param.SeeNote(noteShebangScripts),
		)

		ps.Add("before-print",
			psetter.String{
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

		ps.Add("inner-before-print",
			psetter.String{
				Value: &codeVal,
				Editor: addPrint{
					prefixes: []string{
						"inner-before-", "ib-",
						"before-inner-", "bi-"},
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

		ps.Add("inner-after", psetter.String{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", afterInnerSect),
			param.AltNames("after-inner", "ia", "ai"),
			param.PostAction(scriptPAF(g, &codeVal, afterInnerSect)),
		)

		ps.Add("after", psetter.String{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", afterSect),
			param.AltNames("a"),
			param.PostAction(scriptPAF(g, &codeVal, afterSect)),
		)

		ps.Add(paramNameAfterFile,
			psetter.Pathname{
				Value:       &fileName,
				Expectation: filecheck.FileNonEmpty(),
			},
			makeShebangFileHelpText(afterSect),
			param.AltNames("a-f"),
			param.PostAction(shebangFilePAF(g, &fileName, afterSect)),
			param.SeeAlso(
				paramNameBeforeFile, paramNameExecFile, paramNameGlobalFile),
			param.SeeNote(noteShebangScripts),
		)

		ps.Add("inner-after-print",
			psetter.String{
				Value: &codeVal,
				Editor: addPrint{
					prefixes: []string{
						"inner-after-", "ia-",
						"after-inner-", "ai-"},
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

		ps.Add("after-print",
			psetter.String{
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

		ps.Add("global", psetter.String{Value: &codeVal},
			"follow this with Go code."+
				" For instance, functions that you might want to call from"+
				" several places, global variables or data types."+
				makeCodeSectionHelpText("", globalSect),
			param.AltNames("g"),
			param.PostAction(scriptPAF(g, &codeVal, globalSect)),
		)

		ps.Add(paramNameGlobalFile,
			psetter.Pathname{
				Value:       &fileName,
				Expectation: filecheck.FileNonEmpty(),
			},
			makeShebangFileHelpText(globalSect),
			param.AltNames("g-f"),
			param.PostAction(shebangFilePAF(g, &fileName, globalSect)),
			param.SeeAlso(
				paramNameBeforeFile, paramNameExecFile, paramNameAfterFile),
			param.SeeNote(noteShebangScripts),
		)

		ps.Add(paramNameImport,
			psetter.StrListAppender{
				Value: &g.imports,
				Checks: []check.String{
					check.StringLength[string](check.ValGT(0)),
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
				" Note that if a workspace use directive is given"+
				" the version of Go that you are using must have"+
				" the 'go work ...' command available;"+
				" this command was added in Go 1.18."+
				"\n\n"+
				" Note that the local changes to the other module"+
				" may result in errors from the 'go mod tidy' command."+
				" These would normally cause gosh to abort but you"+
				" can suppress this behaviour with"+
				" the '"+paramNameIgnoreGoModTidyErrs+"' parameter",
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.AltNames("go-work-use"),
			param.SeeAlso(paramNameIgnoreGoModTidyErrs),
		)

		ps.Add("copy-go-file",
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
				" Note also that the file must be in package 'main'.",
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

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
func addGoshParams(g *Gosh) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		ps.AddGroup(paramGroupNameGosh,
			"parameters controlling the behaviour of the gosh command"+
				" rather than the program it generates.")

		const showFileParam = "show-filename"
		ps.Add(showFileParam, psetter.Bool{Value: &g.showFilename},
			"show the filename where the program has been constructed."+
				" This will also prevent the generated code from being"+
				" cleared after execution has successfully completed,"+
				" the assumption being that if you want to know the"+
				" filename you will also want to examine its contents.",
			param.AltNames("show-file", "keep"),
			param.PostAction(paction.SetBool(&g.dontClearFile, true)),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.GroupName(paramGroupNameGosh),
		)

		execNameRE := regexp.MustCompile(`^[a-zA-Z][-a-zA-Z0-9+._]*$`)
		ps.Add("set-exec-name",
			psetter.String{
				Value: &g.execName,
				Checks: []check.String{
					check.StringLength[string](check.ValBetween(1, 50)),
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
				" preserve it.",
			param.AltNames("program-name"),
			param.PostAction(paction.SetBool(&g.dontClearFile, true)),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add("dont-exec", psetter.Bool{Value: &g.dontRun},
			"don't run the generated code - this prevents the generated"+
				" code from being cleared and forces the "+showFileParam+
				" parameter to true. This can be"+
				" useful if you have completed the work you were using"+
				" the generated code for and now want to save the file "+
				" for future use.",
			param.AltNames("dont-run", "no-exec", "no-run"),
			param.PostAction(paction.SetBool(&g.showFilename, true)),
			param.PostAction(paction.SetBool(&g.dontClearFile, true)),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.GroupName(paramGroupNameGosh),
		)

		goCmdName := gogen.GetGoCmdName()
		ps.Add(paramNameSetGoCmd, psetter.String{Value: &goCmdName},
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

		ps.Add("formatter", psetter.String{Value: &g.formatter},
			"the name of the formatter command to run. If the default"+
				" value is not replaced then this program shall look"+
				" for the "+goImportsFormatter+" program and use"+
				" that if it is found.",
			param.PostAction(paction.SetBool(&g.formatterSet, true)),
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add("formatter-args", psetter.StrList{Value: &g.formatterArgs},
			"the arguments to pass to the formatter command. Note that"+
				" the final argument will always be the name of the"+
				" generated program.",
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add(paramNameDontFormat, psetter.Bool{Value: &g.dontFormat},
			"don't format the generated code - this prevents the"+
				" generated code from being run through the formatter."+
				"\n\n"+
				"This can be useful with shebang scripts where you"+
				" want the marginal performance improvement. An"+
				" additional advantage is that the script will run"+
				" successfully even if you don't have access to"+
				" goimports (or some other formatter)."+
				"\n\n"+
				"Note that you will have to give any imports on the"+
				" command line using the "+paramNameImport+" parameter.",
			param.AltNames("dont-fmt", "no-format", "no-fmt"),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.GroupName(paramGroupNameGosh),
			param.SeeAlso(paramNameImport),
		)

		ps.Add("build-arg", psetter.StrListAppender{Value: &g.buildArgs},
			"add an argument to pass to the go build command.",
			param.AltNames("build-args", "args-build", "b-args", "b-arg"),
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add("add-comments", psetter.Bool{Value: &g.addComments},
			"add end-of-line comments to show the lines of code"+
				" generated by gosh.",
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

		stdparams.AddTiming(ps, g.dbgStack, param.GroupName(paramGroupNameGosh))

		ps.Add(paramNameEditScript, psetter.Bool{Value: &g.edit},
			"edit the generated code just before running it.",
			param.AltNames("edit"),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.SeeAlso(paramNameScriptEditor, paramNameEditRepeat),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add(paramNameEditRepeat, psetter.Bool{Value: &g.editRepeat},
			"after the program has run, you will be asked if you want"+
				" to repeat the edit/build/run loop.",
			param.PostAction(paction.SetBool(&g.edit, true)),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
			param.SeeAlso(paramNameScriptEditor, paramNameEditScript),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add(paramNameScriptEditor, psetter.String{Value: &g.editorParam},
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
			param.SeeAlso(paramNameEditScript, paramNameEditRepeat),
			param.Attrs(param.DontShowInStdUsage),
			param.GroupName(paramGroupNameGosh),
		)

		ps.Add(paramNameIgnoreGoModTidyErrs,
			psetter.Bool{
				Value: &g.ignoreGoModTidyErrs,
			},
			"don't abort when the 'go mod tidy' command reports errors;"+
				" the error message, if any, will still be written to stderr."+
				"\n\n"+
				" This should only be set when a workspace is in use",
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
			param.SeeAlso(paramNameWorkspaceUse),
			param.GroupName(paramGroupNameGosh),
		)

		return nil
	}
}
