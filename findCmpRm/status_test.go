package main

import (
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

const (
	testDataDir        = "testdata"
	statusReportSubDir = "statusReport"
)

var gfc = testhelper.GoldenFileCfg{
	DirNames:               []string{testDataDir, statusReportSubDir},
	Sfx:                    "txt",
	UpdFlagName:            "upd-status-report-files",
	KeepBadResultsFlagName: "keep-bad-status-report-results",
}

func init() {
	gfc.AddUpdateFlag()
	gfc.AddKeepBadResultsFlag()
}

func TestStatusReport(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		setStatus func(*Status)
	}{
		{
			ID: testhelper.MkID("no-files"),
		},
		{
			ID:        testhelper.MkID("cmp-files"),
			setStatus: func(s *Status) { s.cmpFile.total = 1 },
		},
		{
			ID:        testhelper.MkID("dup-files"),
			setStatus: func(s *Status) { s.dupFile.total = 1 },
		},
		{
			ID:        testhelper.MkID("bad-files"),
			setStatus: func(s *Status) { s.badFile.total = 1 },
		},
		{
			ID: testhelper.MkID("all-types"),
			setStatus: func(s *Status) {
				s.cmpFile.total = 10
				s.cmpFile.cmpErrs = 3
				s.cmpFile.compared = 3
				s.cmpFile.deleted = 4
				s.cmpFile.reverted = 1
				s.dupFile.total = 10
				s.badFile.total = 10
			},
		},
	}

	for _, tc := range testCases {
		s := InitStatus()

		if tc.setStatus != nil {
			tc.setStatus(&s)
		}

		fakeIO, err := testhelper.NewStdioFromString("")
		if err != nil {
			t.Log(tc.IDStr())
			t.Log("\t: creating Fake std I/O")
			t.Fatal(err)
		}

		s.Report()

		stdout, stderr, err := fakeIO.Done()
		if err != nil {
			t.Log(tc.IDStr())
			t.Log("\t: collecting output")
			t.Fatal(err)
		}

		gfc.Check(t, tc.IDStr()+" [stdout]", tc.Name+"-stdout", stdout)
		gfc.Check(t, tc.IDStr()+" [stderr]", tc.Name+"-stderr", stderr)
	}
}
