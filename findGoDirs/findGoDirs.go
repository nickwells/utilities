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
	actionFunc      func(*findGoDirs, string)
	dirToContentMap map[string]contentMap
)

// findGoDirs holds the parameters and current status of the program
type findGoDirs struct {
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

func newFindGoDirs() *findGoDirs {
	fgd := &findGoDirs{
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

	return fgd
}
