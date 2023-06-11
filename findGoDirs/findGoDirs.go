package main

import "github.com/nickwells/utilities/internal/callstack"

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
	actionFunc      func(*Prog, string)
	dirToContentMap map[string]contentMap
)

// Prog holds the parameters and current status of the program
type Prog struct {
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

	dbgStack *callstack.Stack
}

func NewProg() *Prog {
	return &Prog{
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

		dbgStack: &callstack.Stack{},
	}
}
