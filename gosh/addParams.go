// gosh
package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nickwells/check.mod/check"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

const (
	paramGroupNameReadloop = "cmd-readloop"
	paramGroupNameWeb      = "cmd-web"

	paramNameInPlaceEdit = "in-place-edit"
	paramNameHTTPServer  = "http-server"
)

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
					" for the "+paramNameHTTPServer+" parameter for details. Note"+
					" that if you set this to a value less than 1024 you"+
					" will need to have superuser privilege.",
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
					" for details. If you set this to a value less than"+
					" 1024 you will need to have superuser privilege.",
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

		g.runAsWebserverSetters = append(g.runAsWebserverSetters,
			ps.Add("web-print",
				psetter.StrListAppender{
					Value: &g.script,
					Editor: addPrint{
						prefixes:    []string{"web-"},
						paramToCall: webPrintMap,
						needsVal:    needsValMap,
					},
				},
				"follow this with the value to be printed. These print"+
					" statements will be mixed in with the exec statements"+
					" in the order they are given."+
					"\n\nThis variant will use the Fprint variants,"+
					" passing '_rw' as the writer. Such calls can be used to"+
					" print to the HTTP handler's ResponseWriter "+
					" which is called '_rw' in the generated code.",
				param.AltName("web-printf"),
				param.AltName("web-println"),
				param.AltName("web-p"),
				param.AltName("web-pf"),
				param.AltName("web-pln"),
				param.PostAction(paction.SetBool(&g.runAsWebserver, true)),
				param.GroupName(paramGroupNameWeb),
			),
		)

		ps.AddFinalCheck(func() error {
			if len(g.script) > 0 && g.httpHandler != dfltHTTPHandlerName {
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
					" of strings called 'f'. Setting this will also force"+
					" the script to be run in the loop reading from stdin.",
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
					" the script to be run in the loop reading from stdin"+
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

			if len(ps.Remainder()) != 0 && !g.runInReadLoop {
				return fmt.Errorf(
					"You have given filenames but no parameters" +
						" indicating that they should be read. One of the" +
						" following should be given: " +
						strings.Join(g.runInReadloopParamNames(), ", "))
			}

			return nil
		})

		return nil
	}
}

// addParams will add parameters to the passed ParamSet
func addParams(g *Gosh) func(ps *param.PSet) error {
	return func(ps *param.PSet) error {
		ps.Add("exec", psetter.StrListAppender{Value: &g.script},
			"follow this with the Go code to be run."+
				" This will be placed inside a main() function.",
			param.AltName("e"),
		)

		ps.Add("print",
			psetter.StrListAppender{
				Value: &g.script,
				Editor: addPrint{
					paramToCall: stdPrintMap,
					needsVal:    needsValMap,
				},
			},
			"follow this with the value to be printed. These print"+
				" statements will be mixed in with the exec statements"+
				" in the order they are given.",
			param.AltName("printf"),
			param.AltName("println"),
			param.AltName("p"),
			param.AltName("pf"),
			param.AltName("pln"),
		)

		ps.Add("begin", psetter.StrListAppender{Value: &g.preScript},
			"follow this with Go code to be run at the beginning."+
				" This will be placed inside a main() function before"+
				" the code given for the exec parameters and also"+
				" before any read-loop.",
			param.AltName("before"),
			param.AltName("b"),
		)

		ps.Add("begin-print",
			psetter.StrListAppender{
				Value: &g.preScript,
				Editor: addPrint{
					prefixes:    []string{"begin-", "b-"},
					paramToCall: stdPrintMap,
					needsVal:    needsValMap,
				},
			},
			"follow this with the value to be printed. These print"+
				" statements will be mixed in with the exec statements"+
				" in the order they are given.",
			param.AltName("begin-printf"),
			param.AltName("begin-println"),
			param.AltName("b-p"),
			param.AltName("b-pf"),
			param.AltName("b-pln"),
		)

		ps.Add("end", psetter.StrListAppender{Value: &g.postScript},
			"follow this with Go code to be run at the end."+
				" This will be placed inside a main() function after"+
				" the code given for the exec parameters and most"+
				" importantly outside any read-loop.",
			param.AltName("after"),
			param.AltName("a"))

		ps.Add("end-print",
			psetter.StrListAppender{
				Value: &g.postScript,
				Editor: addPrint{
					prefixes:    []string{"end-", "a-"},
					paramToCall: stdPrintMap,
					needsVal:    needsValMap,
				},
			},
			"follow this with the value to be printed. These print"+
				" statements will be mixed in with the exec statements"+
				" in the order they are given.",
			param.AltName("end-printf"),
			param.AltName("end-println"),
			param.AltName("a-p"),
			param.AltName("a-pf"),
			param.AltName("a-pln"),
		)

		ps.Add("w-print",
			psetter.StrListAppender{
				Value: &g.script,
				Editor: addPrint{
					prefixes:    []string{"w-"},
					paramToCall: wPrintMap,
					needsVal:    needsValMap,
				},
			},
			"follow this with the value to be printed. These print"+
				" statements will be mixed in with the exec statements"+
				" in the order they are given."+
				"\n\nThis variant will use the Fprint variants,"+
				" passing '_w' as the writer. Such calls can be used to"+
				" print to the output file used for in-place editing"+
				" which is called '_w' in the generated code.",
			param.AltName("w-printf"),
			param.AltName("w-println"),
			param.AltName("w-p"),
			param.AltName("w-pf"),
			param.AltName("w-pln"),
		)

		ps.Add("global", psetter.StrListAppender{Value: &g.globalsList},
			"follow this with Go code that should be placed at global scope."+
				" For instance, functions that you might want to call from"+
				" several places, global variables or data types.",
			param.AltName("g"),
		)

		ps.Add("imports",
			psetter.StrListAppender{
				Value:  &g.imports,
				Checks: []check.String{check.StringLenGT(0)},
			},
			"provide any explicit imports.",
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
			param.Attrs(param.DontShowInStdUsage),
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

		ps.AddFinalCheck(func() error {
			if err := check.StringSliceNoDups(g.imports); err != nil {
				return fmt.Errorf("bad list of imports: %s", err)
			}
			return nil
		})

		return nil
	}
}

// runInReadloopParamNames returns a slice of strings, each one the name of a
// parameter that will set the runInreadLoop flag
func (g *Gosh) runInReadloopParamNames() []string {
	rval := make([]string, 0, len(g.runInReadloopSetters))
	for _, p := range g.runInReadloopSetters {
		rval = append(rval, fmt.Sprintf("%q", "-"+p.Name()))
	}
	return rval
}
