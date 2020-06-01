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

var script []string
var preScript []string
var postScript []string
var globalsList []string
var imports []string

var runInReadLoop bool
var splitLine bool
var splitPattern = `\s+`
var runInReadloopSetters []*param.ByName

var runAsWebserver bool

const defaultHTTPPort = 8080

var httpPort int64 = defaultHTTPPort
var runAsWebserverSetters []*param.ByName

var showFilename bool
var dontClearFile bool
var dontRun bool
var filename string

const goImports = "goimports"

var formatter = "gofmt"
var formatterSet bool
var formatterArgs = []string{"-w"}

type addPrint struct {
	prefixes    []string
	paramToCall map[string]string
	needsVal    map[string]bool
}

var needsValMap = map[string]bool{
	"printf": true,
	"pf":     true,
}

var stdPrintMap = map[string]string{
	"print":   "fmt.Print(",
	"p":       "fmt.Print(",
	"printf":  "fmt.Printf(",
	"pf":      "fmt.Printf(",
	"println": "fmt.Println(",
	"pln":     "fmt.Println(",
}

var wPrintMap = map[string]string{
	"print":   "fmt.Fprint(w, ",
	"p":       "fmt.Fprint(w, ",
	"printf":  "fmt.Fprintf(w, ",
	"pf":      "fmt.Fprintf(w, ",
	"println": "fmt.Fprintln(w, ",
	"pln":     "fmt.Fprintln(w, ",
}

// Edit wraps the parameter value in a call to the appropriate variant of the
// mapped function call. The exact print function to use is determined by the
// parameter name and this will determine the errors that might be
// generated. For instance, if a value is needed then an empty string for
// paramVal will generate an error.
func (ap addPrint) Edit(paramName, paramVal string) (string, error) {
	fullParamName := paramName
	for _, pfx := range ap.prefixes {
		s := strings.TrimPrefix(paramName, pfx)
		if s != paramName {
			paramName = s
			break
		}
	}

	callName, ok := ap.paramToCall[paramName]
	if !ok {
		panic(fmt.Errorf("unexpected parameter name: %q", fullParamName))
	}
	if ap.needsVal[paramName] && paramVal == "" {
		return "", errors.New("The parameter value must not be empty")
	}

	return callName + paramVal + ")", nil
}

// addWebParams will add the parameters in the "web" parameter group
func addWebParams(ps *param.PSet) error {
	const webGroup = "cmd-web"

	ps.AddGroup(webGroup,
		"parameters relating to building a script as a web-server.")

	const webServerParam = "run-as-webserver"
	runAsWebserverSetters = append(runAsWebserverSetters,
		ps.Add(webServerParam, psetter.Bool{Value: &runAsWebserver},
			"run a webserver with the script code being run"+
				" within an http handler function called 'goshHandler'"+
				" taking two parameters:\n"+
				" w (an http.ResponseWriter)\n"+
				" r (a pointer to an http.Request).\n"+
				" The webserver will listen on port "+
				fmt.Sprintf("%d", defaultHTTPPort)+
				" unless the port number has been set explicitly"+
				" through the http-port parameter.",
			param.AltName("http"),
			param.GroupName(webGroup),
		),
	)

	runAsWebserverSetters = append(runAsWebserverSetters,
		ps.Add("http-port",
			psetter.Int64{
				Value: &httpPort,
				Checks: []check.Int64{
					check.Int64GT(0),
					check.Int64LT((1 << 16) + 1),
				},
			},
			"set the port number that the webserver will listen on."+
				" Setting this will also force the script to be run"+
				" within an http handler function. See the description"+
				" for the "+webServerParam+" parameter for details. Note"+
				" that if you set this to a value less than 1024 you"+
				" will need to have superuser privilege.",
			param.PostAction(paction.SetBool(&runAsWebserver, true)),
			param.GroupName(webGroup),
		),
	)

	ps.Add("w-print",
		psetter.StrListAppender{
			Value: &script,
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
			" passing 'w' as the writer. Such calls can be used to"+
			" print to the writer passed in to the HTTP handler"+
			" which is called 'w' in the generated code. You can"+
			" think of the 'w' as referring to the web or to a"+
			" writer if it helps you to remember.",
		param.AltName("w-printf"),
		param.AltName("w-println"),
		param.AltName("w-p"),
		param.AltName("w-pf"),
		param.AltName("w-pln"),
		param.GroupName(webGroup),
	)

	return nil
}

// addReadloopParams will add the parameters in the "readloop" parameter
// group
func addReadloopParams(ps *param.PSet) error {
	const rlGroup = "cmd-readloop"

	ps.AddGroup(rlGroup,
		"parameters relating to building a script with a read-loop.")

	runInReadloopSetters = append(runInReadloopSetters,
		ps.Add("run-in-readloop", psetter.Bool{Value: &runInReadLoop},
			"have the script code being run within a loop that reads from stdin"+
				" one a line at a time. The value of each line can be"+
				" accessed by calling 'line.Text()'. Note that any"+
				" newline will have been removed and will need to be added"+
				" back if you want to print the line.",
			param.AltName("n"),
			param.GroupName(rlGroup),
		),
	)

	runInReadloopSetters = append(runInReadloopSetters,
		ps.Add("split-line", psetter.Bool{Value: &splitLine},
			"split the lines into fields around runs of whitespace"+
				" characters. The fields will be available in a slice"+
				" of strings called 'f'. Setting this will also force"+
				" the script to be run in the loop reading from stdin.",
			param.AltName("s"),
			param.PostAction(paction.SetBool(&runInReadLoop, true)),
			param.GroupName(rlGroup),
		),
	)

	runInReadloopSetters = append(runInReadloopSetters,
		ps.Add("split-pattern", psetter.String{Value: &splitPattern},
			"change the behaviour when splitting the line into fields."+
				" The provided string must compile into a regular expression."+
				" Setting this will also force the script to be run in the"+
				" loop reading from stdin and for each line to be split.",
			param.AltName("sp"),
			param.PostAction(paction.SetBool(&runInReadLoop, true)),
			param.PostAction(paction.SetBool(&splitLine, true)),
			param.GroupName(rlGroup),
		),
	)
	return nil
}

// addParams will add parameters to the passed ParamSet
func addParams(ps *param.PSet) error {
	err := addWebParams(ps)
	if err != nil {
		return err
	}
	err = addReadloopParams(ps)
	if err != nil {
		return err
	}

	ps.Add("exec", psetter.StrListAppender{Value: &script},
		"follow this with the Go code to be run."+
			" This will be placed inside a main() function.",
		param.AltName("e"),
	)

	ps.Add("print",
		psetter.StrListAppender{
			Value: &script,
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

	ps.Add("begin", psetter.StrListAppender{Value: &preScript},
		"follow this with Go code to be run at the beginning."+
			" This will be placed inside a main() function before"+
			" the code given for the exec parameters and also"+
			" before any read-loop.",
		param.AltName("before"),
		param.AltName("b"),
	)

	ps.Add("begin-print",
		psetter.StrListAppender{
			Value: &preScript,
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

	ps.Add("end", psetter.StrListAppender{Value: &postScript},
		"follow this with Go code to be run at the end."+
			" This will be placed inside a main() function after"+
			" the code given for the exec parameters and most"+
			" importantly outside any read-loop.",
		param.AltName("after"),
		param.AltName("a"))

	ps.Add("end-print",
		psetter.StrListAppender{
			Value: &postScript,
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

	ps.Add("global", psetter.StrListAppender{Value: &globalsList},
		"follow this with Go code that should be placed at global scope."+
			" For instance, functions that you might want to call from"+
			" several places, global variables or data types.",
		param.AltName("g"),
	)

	ps.Add("imports",
		psetter.StrListAppender{
			Value:  &imports,
			Checks: []check.String{check.StringLenGT(0)},
		},
		"provide any explicit imports.",
		param.AltName("I"),
	)

	const showFileParam = "show-filename"
	ps.Add(showFileParam, psetter.Bool{Value: &showFilename},
		"show the filename where the program has been constructed."+
			" This will also prevent the generated code from being"+
			" cleared after execution has successfully completed,"+
			" the assumption being that if you want to know the"+
			" filename you will also want to examine its contents.",
		param.AltName("show-file"),
		param.PostAction(paction.SetBool(&dontClearFile, true)),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("set-filename",
		psetter.String{
			Value: &filename,
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
		param.PostAction(paction.SetBool(&dontClearFile, true)),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("dont-exec", psetter.Bool{Value: &dontRun},
		"don't run the generated code - this prevents the generated"+
			" code from being cleared and forces the "+showFileParam+
			" parameter to true. This can be"+
			" useful if you have completed the work you were using"+
			" the generated code for and now want to save the file "+
			" for future use.",
		param.AltName("dont-run"),
		param.AltName("no-exec"),
		param.AltName("no-run"),
		param.PostAction(paction.SetBool(&showFilename, true)),
		param.PostAction(paction.SetBool(&dontClearFile, true)),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("formatter", psetter.String{Value: &formatter},
		"the name of the formatter command to run. If the default"+
			" value is not replaced then this program shall look"+
			" for the "+goImports+" program and use that if it is found.",
		param.PostAction(paction.SetBool(&formatterSet, true)),
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.Add("formatter-args", psetter.StrList{Value: &formatterArgs},
		"the arguments to pass to the formatter command. Note that the"+
			" final argument will always be the name of the generated program.",
		param.Attrs(param.DontShowInStdUsage),
	)

	ps.AddFinalCheck(func() error {
		if runAsWebserver && runInReadLoop {
			errStr := "gosh cannot read from standard input" +
				" and run as a webserver." +
				" Parameters set at:"
			for _, p := range runAsWebserverSetters {
				for _, w := range p.WhereSet() {
					errStr += "\n\t" + w
				}
			}
			for _, p := range runInReadloopSetters {
				for _, w := range p.WhereSet() {
					errStr += "\n\t" + w
				}
			}
			return errors.New(errStr)
		}
		return nil
	})

	ps.AddFinalCheck(func() error {
		if err := check.StringSliceNoDups(imports); err != nil {
			return fmt.Errorf("bad list of imports: %s", err)
		}
		return nil
	})

	return nil
}

// addExamples adds some examples of how gosh might be used to the standard
// help message
func addExamples(ps *param.PSet) error {
	ps.AddExample("gosh -pln '\"Hello, World!\"'",
		`This will print Hello, World!`)
	ps.AddExample("gosh -pln 'math.Pi'",
		`This will print the value of Pi`)
	ps.AddExample(
		"gosh -n -b 'count := 0' -e 'count++' -a-pln 'count'",
		"This will read from the standard input and print"+
			" the number of lines read"+
			"\n"+
			"\n-n sets up the loop reading from standard input"+
			"\n-b 'count := 0' declares and initialises the counter"+
			" before the loop"+
			"\n-e 'count++' increments the counter inside the loop"+
			"\n-a-pln 'count' prints the counter using fmt.Println"+
			" after the loop.")
	ps.AddExample("gosh -n -b-p '\"Radius: \"'"+
		" -e 'r, _ := strconv.ParseFloat(line.Text(), 10)'"+
		" -pf '\"Area: %9.2f\\n\", r*r*math.Pi'"+
		" -p '\"Radius: \"'",
		"This will repeatedly prompt the user for a Radius and print"+
			" the Area of the corresponding circle"+
			"\n"+
			"\n-n sets up the loop reading from standard input"+
			"\n-b-p '\"Radius: \"' prints the first prompt"+
			" before the loop"+
			"\n-e 'r, _ := strconv.ParseFloat(line.Text(), 10)' sets"+
			" the radius from the text read from standard input,"+
			" ignoring errors"+
			"\n-pf '\"Area: %9.2f\\n\", r*r*math.Pi'' calculates and"+
			" prints the area using fmt.Printf"+
			"\n-p '\"Radius: \"' prints the next prompt.")

	return nil
}
