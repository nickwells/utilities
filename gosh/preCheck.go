package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nickwells/gogen.mod/gogen"
	"github.com/nickwells/twrap.mod/twrap"
)

const (
	preChkStdIndent  = 4
	preChkListIndent = 8
)

// PreCheck will test the availability of the various components that gosh
// needs and make recommendations as to how to fix any missing components.
func preCheck(g *gosh) {
	if !g.preCheck {
		return
	}

	var problemsFound bool

	exitStatus := 0
	twc := twrap.NewTWConfOrPanic()

	if goCmdBad(twc) {
		problemsFound = true
		exitStatus = goshExitStatusPreCheck
	}

	if importersBad(g, twc) {
		problemsFound = true
		exitStatus = goshExitStatusPreCheck
	}

	if snippetsBad(g, twc) {
		problemsFound = true
		exitStatus = goshExitStatusPreCheck
	}

	if problemsFound {
		fmt.Print("Setting parameters in configuration files\n\n")
		twc.Wrap("Parameters can be set through the command line but also"+
			" through entries in the gosh configuration files. These are"+
			" processed before the command line parameters so any value"+
			" given in a file can be superseded by a command-line entry."+
			"\n\n"+
			"The advantage of this is that a parameter value set like"+
			" this is applied every time gosh is run. You don't need to"+
			" set the parameter again."+
			"\n\n"+
			"To find the configuration files look at the help pages which"+
			" list alternative sources.",
			preChkStdIndent)
	}

	os.Exit(exitStatus)
}

// goCmdBad checks for problems with the Go command. If it is available and
// executable it returns false. Otherwise it reports the problem, describes
// potential remedies and returns true.
func goCmdBad(twc *twrap.TWConf) bool {
	goCmd := gogen.GetGoCmdName()

	_, err := exec.LookPath(goCmd)
	if err == nil {
		return false
	}

	fmt.Print("The Go command\n\n")

	twc.Wrap("'"+goCmd+"' is not executable or is not"+
		" found in your PATH. You should either",
		preChkStdIndent)
	twc.ListItem(preChkListIndent,
		"Change your PATH value to include the directory containing"+
			" the Go command",
		"Give the full pathname to the Go command explicitly using"+
			" the '"+paramNameSetGoCmd+"' parameter.")
	twc.Wrap("If the Go command is not installed on your computer you"+
		" will need to install it for gosh to work",
		preChkStdIndent)
	fmt.Println()

	return true
}

// importersBad checks for problems with the importers. If a valid importer is
// available it returns false. Otherwise it reports the problem, describes
// potential remedies and returns true.
//
// It will skip the check if the dontPopulateImports flag is set.
func importersBad(g *gosh, twc *twrap.TWConf) bool {
	if g.dontPopulateImports {
		return false
	}

	if g.importPopulatorSet {
		_, err := exec.LookPath(g.importPopulator)
		if err == nil {
			return false
		}
	} else if _, _, ok := findImporter(g); ok {
		return false
	}

	fmt.Print("The import populator\n\n")

	twc.Wrap("There is no command to automatically populate the import"+
		" statements. Although gosh can work without an import populator"+
		" you will then need to give all the packages to be imported on"+
		" the command line with the '"+paramNameImport+"' parameter.",
		preChkStdIndent)

	fmt.Println()

	if g.importPopulatorSet {
		twc.Wrap("You have set the import-populator command"+
			" as '"+g.importPopulator+"' but this"+
			" cannot be found. Either you have entered the command"+
			" incorrectly or else it cannot be found in your PATH. You"+
			" should either",
			preChkStdIndent)
		twc.ListItem(preChkListIndent,
			"Change the value of your PATH to include the directory"+
				" containing '"+g.importPopulator+"'",
			"Give the full pathname to the import-populator command.")
		fmt.Println()
	} else {
		twc.Wrap("If you have a program that can fill in the import"+
			" statements then you can give the full pathname to this"+
			" program explicitly using the '"+paramNameImporter+"'"+
			" parameter. Note that if you do this you will possibly also"+
			" need to specify the parameters that this"+
			" program uses. You can do this using"+
			" the '"+paramNameImporterArgs+"' parameter. The"+
			" import-populator program is called with the given arguments"+
			" and the name of the Go file as the last argument.",
			preChkStdIndent)
		fmt.Println()
	}

	twc.Wrap("By default gosh will search for one of the following"+
		" commands: "+importerPrograms()+". If these are already"+
		" installed and you want to use them you should either",
		preChkStdIndent)
	twc.ListItem(preChkListIndent,
		"Change the value of your PATH to include the directory"+
			" containing one of these commands",
		"Set the name of the import-populator command explicitly as"+
			" described above.")
	fmt.Println()
	twc.Wrap("If none of the default import-populators are installed"+
		" they can be installed with the following commands (only one"+
		" is needed):",
		preChkStdIndent)

	iInstallCmds := make([]string, 0, len(importers))

	for _, i := range importers {
		iInstallCmds = append(iInstallCmds, i.installCmd)
	}

	twc.List(iInstallCmds, preChkListIndent)

	fmt.Println()

	return true
}

// snippetsBad checks for problems with the snippets. If some snippets are
// available and all the directories can be searched it returns
// false. Otherwise it reports the problem, describes potential remedies and
// returns true.
func snippetsBad(g *gosh, twc *twrap.TWConf) bool {
	snippetCount := 0

	var snippetErrs error

	for _, dir := range g.snippetDirs {
		count, err := countSnippets(0, dir)
		snippetCount += count
		snippetErrs = errors.Join(snippetErrs, err)
	}

	if snippetErrs == nil && snippetCount > 0 {
		return false
	}

	fmt.Print("Snippets\n\n")

	if snippetErrs != nil {
		twc.Wrap("There is a problem with your snippet directories: "+
			snippetErrs.Error(), preChkStdIndent)
	} else if snippetCount == 0 {
		twc.Wrap("You have no snippets installed. You can use 'gosh'"+
			" without snippets but you may find them useful. Use the"+
			" 'gosh.snippet'"+
			" program to install the standard snippets. You will need to"+
			" install this program in the same way you installed gosh."+
			" You can install the snippets in one of these directories:",
			preChkStdIndent)
		twc.List(g.snippetDirs, preChkListIndent)
		twc.Wrap("Choose one of these to install the standard snippets"+
			" and then run the gosh.snippet program as follows:"+
			"\n\n"+
			"    gosh.snippet -to <dir> -install"+
			"\n\n"+
			"Alternatively if you have snippets in another directory or"+
			" you have the standard snippets alredy installed elsewhere"+
			" you can add to the list of directories that gosh will"+
			" search by using the '"+paramNameSnippetDir+"' parameter.",
			preChkStdIndent)
	}

	fmt.Println()

	return true
}

// countSnippets returns the number of snippet files found and any errors
// detected. If one of the entries in a directory is itself a directory then
// it will recursively descend into the directory.
func countSnippets(depth int, dir string) (int, error) {
	const maxSnippetDepth = 10

	depth++
	if depth > maxSnippetDepth {
		return 0,
			fmt.Errorf("the snippet directory %q is too deep (> %d levels)",
				dir, maxSnippetDepth)
	}

	count := 0

	var err error

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		} else {
			err = fmt.Errorf("bad snippets directory: %q: %w", dir, err)
		}

		return count, err
	}

	for _, de := range dirEntries {
		if de.IsDir() {
			dirCount, dirErr := countSnippets(
				depth,
				filepath.Join(dir, de.Name()))
			count += dirCount
			err = errors.Join(err, dirErr)

			continue
		}

		count++
	}

	return count, err
}
