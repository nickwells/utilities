package main

import (
	"testing"
	"time"

	"github.com/nickwells/mathutil.mod/v2/mathutil"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestSleepCalc(t *testing.T) {
	now := time.Date(2020, time.June, 19, 1, 2, 3, 123000000, time.UTC)
	testCases := []struct {
		testhelper.ID
		durationSeconds  int64
		offset           int64
		expectedDuration time.Duration
	}{
		{
			ID:               testhelper.MkID("10 secs, no offset"),
			durationSeconds:  10,
			offset:           0,
			expectedDuration: time.Duration(6877000000),
		},
		{
			ID:               testhelper.MkID("10 secs, offset: 2"),
			durationSeconds:  10,
			offset:           2,
			expectedDuration: time.Duration(8877000000),
		},
		{
			ID:               testhelper.MkID("10 secs, offset: -2"),
			durationSeconds:  10,
			offset:           -2,
			expectedDuration: time.Duration(4877000000),
		},
		{
			ID: testhelper.MkID(
				"10 secs, offset: 22 (bigger than duration)"),
			durationSeconds:  10,
			offset:           22,
			expectedDuration: time.Duration(8877000000),
		},
		{
			ID: testhelper.MkID(
				"10 secs, offset: -22 (smaller than duration)"),
			durationSeconds:  10,
			offset:           -22,
			expectedDuration: time.Duration(4877000000),
		},
	}

	for _, tc := range testCases {
		actualDuration := sleepCalc(tc.durationSeconds, tc.offset, now)
		if !mathutil.AlmostEqual(
			float64(actualDuration),
			float64(tc.expectedDuration),
			1000000.0) {
			t.Log(tc.IDStr())
			t.Logf("\t: expected duration: %15d\n", tc.expectedDuration)
			t.Logf("\t:   actual duration: %15d\n", actualDuration)
			t.Errorf("\t: duration differs\n")
		}
	}
}
