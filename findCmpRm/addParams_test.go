package main

import (
	"errors"
	"os"
	"testing"

	"github.com/nickwells/errutil.mod/errutil"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/param.mod/v6/paramtest"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

// cmpProgStruct compares the value with the expected value and returns
// an error if they differ
func cmpProgStruct(iVal, iExpVal any) error {
	val, ok := iVal.(*prog)
	if !ok {
		return errors.New("Bad value: not a pointer to a Prog struct")
	}

	expVal, ok := iExpVal.(*prog)
	if !ok {
		return errors.New("Bad expected value: not a pointer to a Prog struct")
	}

	return testhelper.DiffVals(val, expVal)
}

// mkTestParser populates and returns a paramtest.Parser ready to be added to
// the testcases.
func mkTestParser(
	errs errutil.ErrMap, id testhelper.ID,
	progSetter func(prog *prog),
	preFunc, postFunc func() error,
	args ...string,
) paramtest.Parser {
	actVal := newProg()
	ps := paramset.NewNoHelpNoExitNoErrRptOrPanic(
		addParams(actVal),
	)

	expVal := newProg()
	if progSetter != nil {
		progSetter(expVal)
	}

	return paramtest.Parser{
		ID:             id,
		ExpParseErrors: errs,
		Val:            actVal,
		Ps:             ps,
		ExpVal:         expVal,
		Args:           args,
		CheckFunc:      cmpProgStruct,
		Pre:            preFunc,
		Post:           postFunc,
	}
}

// TestParseParams will use the paramtest.Parser to make sure the
// behaviour of the parameter setting is as expected.
func TestParseParamsCmdProg(t *testing.T) {
	testCases := []paramtest.Parser{}

	// no params; no change
	testCases = append(testCases,
		mkTestParser(nil, testhelper.MkID("good: no params, no change"),
			nil, nil, nil))

	{
		parseErrs := errutil.ErrMap{}
		parseErrs.AddError(
			paramNameCmpAction,
			errors.New(`value not allowed: "nonesuch"`+"\n"+
				"At: [command line]:"+
				` Supplied Parameter:2: "-comparable-action" "nonesuch"`))

		testCases = append(testCases,
			mkTestParser(parseErrs, testhelper.MkID("bad: cmp-action"),
				nil, nil, nil,
				"-"+paramNameCmpAction, "nonesuch"))
	}
	{
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID("good: cmp-action"),
				func(prog *prog) { prog.cmpAction = caKeepAll }, nil, nil,
				"-"+paramNameCmpAction, string(caKeepAll)))
	}
	{
		parseErrs := errutil.ErrMap{}
		parseErrs.AddError(
			paramNameDupAction,
			errors.New(`value not allowed: "nonesuch"`+"\n"+
				"At: [command line]:"+
				` Supplied Parameter:2: "-duplicate-action" "nonesuch"`))

		testCases = append(testCases,
			mkTestParser(parseErrs, testhelper.MkID("bad: dup-action"),
				nil, nil, nil,
				"-"+paramNameDupAction, "nonesuch"))
	}
	{
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID("good: dup-action"),
				func(prog *prog) { prog.dupAction = daKeep }, nil, nil,
				"-"+paramNameDupAction, string(daKeep)))
	}
	{
		const tmpDirTest = "_tmpdir.test"

		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID("good: dir"),
				func(prog *prog) { prog.searchDir = tmpDirTest },
				func() error { return os.Mkdir(tmpDirTest, 0o700) },
				func() error { return os.Remove(tmpDirTest) },
				"-"+paramNameDir, tmpDirTest))
	}
	{
		parseErrs := errutil.ErrMap{}
		parseErrs.AddError(
			paramNameDir,
			errors.New(`path: "nonesuch": should exist but does not;`+
				` "." exists but "nonesuch" does not`+
				"\n"+
				"At: [command line]:"+
				` Supplied Parameter:2: "-dir" "nonesuch"`))

		testCases = append(testCases,
			mkTestParser(parseErrs, testhelper.MkID("bad: dir"),
				nil, nil, nil,
				"-"+paramNameDir, "nonesuch"))
	}
	{
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID("good: dont-recurse"),
				func(prog *prog) { prog.searchSubDirs = false }, nil, nil,
				"-"+paramNameNoRecurse))
	}
	{
		testCases = append(testCases,
			mkTestParser(nil, testhelper.MkID("good: extension"),
				func(prog *prog) { prog.fileExtension = ".pre" }, nil, nil,
				"-"+paramNameExtension, ".pre"))
	}
	{
		parseErrs := errutil.ErrMap{}
		parseErrs.AddError(
			paramNameExtension,
			errors.New("the length of the string (0) is incorrect:"+
				" the value (0) must be greater than 0\n"+
				"At: [command line]:"+
				` Supplied Parameter:2: "-extension" ""`))

		testCases = append(testCases,
			mkTestParser(parseErrs, testhelper.MkID("bad: extension"),
				nil, nil, nil,
				"-"+paramNameExtension, ""))
	}

	for _, tc := range testCases {
		_ = tc.Test(t)
	}
}
