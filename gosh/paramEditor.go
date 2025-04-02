package main

import (
	"errors"
	"fmt"
	"strings"
)

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
	"print":   "fmt.Fprint(_w, ",
	"p":       "fmt.Fprint(_w, ",
	"printf":  "fmt.Fprintf(_w, ",
	"pf":      "fmt.Fprintf(_w, ",
	"println": "fmt.Fprintln(_w, ",
	"pln":     "fmt.Fprintln(_w, ",
}

var webPrintMap = map[string]string{
	"print":   "fmt.Fprint(_rw, ",
	"p":       "fmt.Fprint(_rw, ",
	"printf":  "fmt.Fprintf(_rw, ",
	"pf":      "fmt.Fprintf(_rw, ",
	"println": "fmt.Fprintln(_rw, ",
	"pln":     "fmt.Fprintln(_rw, ",
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
		return "", errors.New("the parameter value must not be empty")
	}

	return callName + paramVal + ")", nil
}
