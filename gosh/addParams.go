package main

import (
	"errors"
	"fmt"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

const (
	paramGroupNameReadloop = "cmd-readloop"
	paramGroupNameWeb      = "cmd-web"

	paramNameInPlaceEdit = "in-place-edit"
	paramNameHTTPServer  = "http-server"

	paramNameSnippetDir  = "snippets-dir"
	paramNameSnippetList = "snippets-list"

	globalSect = "global"
	beforeSect = "before"
	execSect   = "exec"
	afterSect  = "after"
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
				" insert another direcory into the list. When finding"+
				" snippets the directories are searched in order and the"+
				" first snippet found is used.",
			param.AltName("snippet-dir"),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add(paramNameSnippetList,
			psetter.Bool{Value: &g.showSnippets},
			"list all the available snippets and exit, no program is run."+
				" It will also show any per-snippet documentation and"+
				" report on any problems detected with the snippets.",
			param.AltName("show-snippets"),
			param.AltName("snippet-list"),
			param.Attrs(param.CommandLineOnly|param.DontShowInStdUsage),
		)

		var snippetName string
		ps.Add("exec-snippet",
			psetter.String{
				Value:  &snippetName,
				Checks: []check.String{check.StringLenGT(0)},
			},
			makeSnippetHelpText(execSect),
			param.AltName("snippet"),
			param.AltName("e-s"),
			param.PostAction(snippetPAF(g, &snippetName, goshScriptExec)),
			param.SeeAlso(paramNameSnippetDir, paramNameSnippetList),
		)

		ps.Add("before-snippet",
			psetter.String{
				Value:  &snippetName,
				Checks: []check.String{check.StringLenGT(0)},
			},
			makeSnippetHelpText(beforeSect),
			param.AltName("b-s"),
			param.PostAction(snippetPAF(g, &snippetName, goshScriptBefore)),
		)

		ps.Add("after-snippet",
			psetter.String{
				Value:  &snippetName,
				Checks: []check.String{check.StringLenGT(0)},
			},
			makeSnippetHelpText(afterSect),
			param.AltName("a-s"),
			param.PostAction(snippetPAF(g, &snippetName, goshScriptAfter)),
		)

		ps.Add("global-snippet",
			psetter.String{
				Value:  &snippetName,
				Checks: []check.String{check.StringLenGT(0)},
			},
			makeSnippetHelpText(globalSect),
			param.AltName("g-s"),
			param.PostAction(snippetPAF(g, &snippetName, goshScriptGlobal)),
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
			ps.Add(paramNameHTTPServer, psetter.Bool{Value: &g.runAsWebserver},
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
				param.AltName("http"),
				param.GroupName(paramGroupNameWeb),
			),
		)

		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("http-port",
				psetter.Int64{
					Value: &g.httpPort,
					Checks: []check.Int64{
						check.Int64GT(0),
						check.Int64LT((1 << 16) + 1),
					},
				},
				"set the port number that the webserver will listen on."+
					" Setting this will also force the script to be run"+
					" within an http handler function. See the description"+
					" for the "+paramNameHTTPServer+" parameter for"+
					" details. Note that if you set this to a value less"+
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
						check.StringLenGT(0),
					},
				},
				"set the path name (the pattern) that the webserver will"+
					" listen on. Setting this will also force the script"+
					" to be run within an http handler function. See the"+
					" description for the "+paramNameHTTPServer+" parameter"+
					" for details.",
				param.PostAction(paction.SetBool(&g.runAsWebserver, true)),
				param.GroupName(paramGroupNameWeb),
			),
		)

		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("http-handler",
				psetter.String{
					Value: &g.httpHandler,
					Checks: []check.String{
						check.StringLenGT(0),
					},
				},
				"set the handler for the web server. Setting this will"+
					" also force the program to be run as a web server."+
					" Note that no script is expected in this case as the"+
					" function is supplied here.",
				param.PostAction(paction.SetBool(&g.runAsWebserver, true)),
				param.AltName("http-h"),
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
				param.AltName("web-printf"),
				param.AltName("web-println"),
				param.AltName("web-p"),
				param.AltName("web-pf"),
				param.AltName("web-pln"),
				param.PostAction(paction.SetBool(&g.runAsWebserver, true)),
				param.GroupName(paramGroupNameWeb),
				param.PostAction(scriptPAF(g, &codeVal, goshScriptExec)),
			),
		)

		ps.AddFinalCheck(func() error {
			if len(g.scripts[goshScriptExec]) > 0 &&
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
		ps.AddGroup(paramGroupNameReadloop,
			"parameters relating to building a script with a read-loop.")

		g.runInReadloopSetters = append(g.runInReadloopSetters,
			ps.Add("run-in-readloop", psetter.Bool{Value: &g.runInReadLoop},
				"have the script code being run within a loop that reads"+
					" from stdin one a line at a time. The value of each"+
					" line can be accessed by calling 'line.Text()'. Note"+
					" that any newline will have been removed and will"+
					" need to be added back if you want to print the line.",
				param.AltName("n"),
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
				param.AltName("s"),
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
				param.AltName("sp"),
				param.PostAction(paction.SetBool(&g.runInReadLoop, true)),
				param.PostAction(paction.SetBool(&g.splitLine, true)),
				param.GroupName(paramGroupNameReadloop),
			),
		)

		g.runInReadloopSetters = append(g.runInReadloopSetters,
			ps.Add(paramNameInPlaceEdit, psetter.Bool{Value: &g.inPlaceEdit},
				"read each file given as a residual parameter"+
					" (after "+ps.TerminalParam()+") and replace its"+
					" contents with whatever is printed to the '_w' file "+
					"(you can use the 'w-print...' parameters for this)."+
					" The original file will be kept in a copy with the"+
					" original name and  '.orig' extension. If any of the"+
					" supplied files already has a '.orig' copy then the"+
					" file will be reported and execution will stop",
				param.AltName("i"),
				param.PostAction(paction.SetBool(&g.runInReadLoop, true)),
				param.GroupName(paramGroupNameReadloop),
			),
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

			return nil
		})

		return nil
	}
}

// addParams will add parameters to the passed ParamSet
func addParams(g *Gosh) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		var codeVal string

		ps.Add("exec", psetter.String{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", execSect),
			param.AltName("e"),
			// ... and to help our python-speaking friends feel at home
			// (bash also uses -c)
			param.AltName("c"),
			param.PostAction(scriptPAF(g, &codeVal, goshScriptExec)),
		)

		ps.Add("exec-print",
			psetter.String{
				Value: &codeVal,
				Editor: addPrint{
					paramToCall: stdPrintMap,
					needsVal:    needsValMap,
				},
			},
			makePrintHelpText(execSect),
			param.AltName("print"),
			param.AltName("printf"),
			param.AltName("println"),
			param.AltName("p"),
			param.AltName("pf"),
			param.AltName("pln"),
			param.PostAction(scriptPAF(g, &codeVal, goshScriptExec)),
		)

		ps.Add("before", psetter.String{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", beforeSect),
			param.AltName("b"),
			param.PostAction(scriptPAF(g, &codeVal, goshScriptBefore)),
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
			param.AltName("before-printf"),
			param.AltName("before-println"),
			param.AltName("b-p"),
			param.AltName("b-pf"),
			param.AltName("b-pln"),
			param.PostAction(scriptPAF(g, &codeVal, goshScriptBefore)),
		)

		ps.Add("after", psetter.String{Value: &codeVal},
			"follow this with Go code."+
				makeCodeSectionHelpText("", afterSect),
			param.AltName("a"),
			param.PostAction(scriptPAF(g, &codeVal, goshScriptAfter)),
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
			param.AltName("after-printf"),
			param.AltName("after-println"),
			param.AltName("a-p"),
			param.AltName("a-pf"),
			param.AltName("a-pln"),
			param.PostAction(scriptPAF(g, &codeVal, goshScriptAfter)),
		)

		ps.Add("w-print",
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
			param.AltName("w-printf"),
			param.AltName("w-println"),
			param.AltName("w-p"),
			param.AltName("w-pf"),
			param.AltName("w-pln"),
			param.PostAction(scriptPAF(g, &codeVal, goshScriptExec)),
		)

		ps.Add("global", psetter.String{Value: &codeVal},
			"follow this with Go code."+
				" For instance, functions that you might want to call from"+
				" several places, global variables or data types."+
				makeCodeSectionHelpText("", globalSect),
			param.AltName("g"),
			param.PostAction(scriptPAF(g, &codeVal, goshScriptGlobal)),
		)

		ps.Add("import",
			psetter.StrListAppender{
				Value:  &g.imports,
				Checks: []check.String{check.StringLenGT(0)},
			},
			"provide any explicit imports.",
			param.AltName("imports"),
			param.AltName("I"),
		)

		const showFileParam = "show-filename"
		ps.Add(showFileParam, psetter.Bool{Value: &g.showFilename},
			"show the filename where the program has been constructed."+
				" This will also prevent the generated code from being"+
				" cleared after execution has successfully completed,"+
				" the assumption being that if you want to know the"+
				" filename you will also want to examine its contents.",
			param.AltName("show-file"),
			param.PostAction(paction.SetBool(&g.dontClearFile, true)),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("set-filename",
			psetter.String{
				Value: &g.filename,
				Checks: []check.String{
					check.StringLenGT(3),
					check.StringHasSuffix(".go"),
					check.StringNot(
						check.StringHasSuffix("_test.go"),
						"a string ending with _test.go"+
							" - the file must not be a test file."),
				},
			},
			"set the filename where the program will be constructed. This will"+
				" also prevent the generated code from being cleared after"+
				" execution has successfully completed, the assumption being"+
				" that if you have set the filename you will want to preserve"+
				" its contents."+
				"\n\n"+
				"This will also have the consequence that the directory is not"+
				" created and the module is not initialised. This may cause"+
				" problems depending on your current directory (if you are in"+
				" a Go module directory) and the setting of the GO111MODULE"+
				" environment variable.",
			param.AltName("file-name"),
			param.PostAction(paction.SetBool(&g.dontClearFile, true)),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("dont-exec", psetter.Bool{Value: &g.dontRun},
			"don't run the generated code - this prevents the generated"+
				" code from being cleared and forces the "+showFileParam+
				" parameter to true. This can be"+
				" useful if you have completed the work you were using"+
				" the generated code for and now want to save the file "+
				" for future use.",
			param.AltName("dont-run"),
			param.AltName("no-exec"),
			param.AltName("no-run"),
			param.PostAction(paction.SetBool(&g.showFilename, true)),
			param.PostAction(paction.SetBool(&g.dontClearFile, true)),
			param.Attrs(param.DontShowInStdUsage|param.CommandLineOnly),
		)

		ps.Add("formatter", psetter.String{Value: &g.formatter},
			"the name of the formatter command to run. If the default"+
				" value is not replaced then this program shall look"+
				" for the "+goImportsFormatter+" program and use"+
				" that if it is found.",
			param.PostAction(paction.SetBool(&g.formatterSet, true)),
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("formatter-args", psetter.StrList{Value: &g.formatterArgs},
			"the arguments to pass to the formatter command. Note that"+
				" the final argument will always be the name of the"+
				" generated program.",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("no-comment", psetter.Bool{Value: &g.addComments, Invert: true},
			"do not generate the end-of-line comments.",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("base-temp-dir",
			psetter.Pathname{
				Value:       &g.baseTempDir,
				Expectation: filecheck.DirExists(),
			},
			"set the directory where the temporary directories in which"+
				" the gosh program will be generated",
			param.Attrs(param.DontShowInStdUsage),
		)

		ps.Add("local-module",
			ModuleMapSetter{
				Value:     &g.localModules,
				Separator: "=>",
			},
			"give the name and mapping of a local module."+
				" The name should be the module name"+
				" and the mapping should be the path to"+
				" the module directory from your current directory."+
				" They should be separated by '"+ModuleMapSeparator+"'.",
			param.Attrs(param.DontShowInStdUsage),
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
