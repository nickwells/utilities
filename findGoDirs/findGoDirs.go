package main

import "github.com/nickwells/verbose.mod/verbose"

const (
	printAct    = "print"
	buildAct    = "build"
	installAct  = "install"
	testAct     = "test"
	generateAct = "generate"
	contentAct  = "content"
	filenameAct = "filename"
)

type (
	actionFunc      func(*prog, string)
	dirToContentMap map[string]contentMap
)

// prog holds the parameters and current status of the program
type prog struct {
	baseDirs      []string
	skipDirs      []string
	pkgNames      []string
	filesWanted   []string
	filesMissing  []string
	contentChecks checkMap
	dirContent    dirToContentMap

	noAction bool

	actions map[string]bool

	actionFuncs map[string]actionFunc

	generateArgs []string
	installArgs  []string
	buildArgs    []string
	testArgs     []string

	dbgStack *verbose.Stack
}

// newProg returns a properly initialised prog structure
func newProg() *prog {
	return &prog{
		contentChecks: make(checkMap),
		dirContent:    make(dirToContentMap),
		actions:       make(map[string]bool),
		actionFuncs: map[string]actionFunc{
			printAct:    doPrint,
			buildAct:    doBuild,
			installAct:  doInstall,
			testAct:     doTest,
			generateAct: doGenerate,
			contentAct:  doContent,
			filenameAct: doFilenames,
		},

		dbgStack: &verbose.Stack{},
	}
}
