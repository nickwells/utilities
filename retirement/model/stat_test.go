package model

import (
	"testing"

	"github.com/nickwells/testhelper.mod/testhelper"
)

// differs compares the two slices and returns true if they differ
func differs(a, b []float64) bool {
	if len(a) != len(b) {
		return true
	}

	for i, v := range a {
		if v != b[i] {
			return true
		}
	}

	return false
}

func TestInsert(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		val       float64
		valSlc    []float64
		discard   discardType
		expResult []float64
	}{
		{
			ID:        testhelper.MkID("from start, one entry"),
			val:       11,
			valSlc:    []float64{10},
			discard:   DropFromStart,
			expResult: []float64{11},
		},
		{
			ID:        testhelper.MkID("from end, one entry"),
			val:       8,
			valSlc:    []float64{9},
			discard:   DropFromEnd,
			expResult: []float64{8},
		},
		{
			ID:        testhelper.MkID("from start, biggest"),
			val:       11,
			valSlc:    []float64{9, 10},
			discard:   DropFromStart,
			expResult: []float64{10, 11},
		},
		{
			ID:        testhelper.MkID("from end, smallest"),
			val:       8,
			valSlc:    []float64{9, 10},
			discard:   DropFromEnd,
			expResult: []float64{8, 9},
		},
		{
			ID:        testhelper.MkID("from start, dup biggest"),
			val:       11,
			valSlc:    []float64{9, 10, 11},
			discard:   DropFromStart,
			expResult: []float64{10, 11, 11},
		},
		{
			ID:        testhelper.MkID("from end, dup smallest"),
			val:       9,
			valSlc:    []float64{9, 10, 11},
			discard:   DropFromEnd,
			expResult: []float64{9, 9, 10},
		},
		{
			ID:        testhelper.MkID("from start, dup middle"),
			val:       10,
			valSlc:    []float64{9, 10, 11},
			discard:   DropFromStart,
			expResult: []float64{10, 10, 11},
		},
		{
			ID:        testhelper.MkID("from end, dup middle"),
			val:       10,
			valSlc:    []float64{9, 10, 11},
			discard:   DropFromEnd,
			expResult: []float64{9, 10, 10},
		},
	}

	for _, tc := range testCases {
		insert(tc.val, tc.valSlc, tc.discard)
		if differs(tc.valSlc, tc.expResult) {
			t.Log(tc.IDStr())
			t.Log("\t: Expected:", tc.expResult)
			t.Log("\t:      Got:", tc.valSlc)
			t.Errorf("\t: insert failed\n")
		}
	}
}

func TestMerge(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		s1, s2, expResult []float64
	}{
		{
			ID:        testhelper.MkID("overlap"),
			s1:        []float64{1, 2, 3},
			s2:        []float64{2, 3, 4},
			expResult: []float64{1, 2, 2, 3, 3, 4},
		},
		{
			ID:        testhelper.MkID("s1 empty"),
			s1:        []float64{},
			s2:        []float64{2, 3, 4},
			expResult: []float64{2, 3, 4},
		},
		{
			ID:        testhelper.MkID("s2 empty"),
			s1:        []float64{1, 2, 3},
			s2:        []float64{},
			expResult: []float64{1, 2, 3},
		},
		{
			ID:        testhelper.MkID("both empty"),
			s1:        []float64{},
			s2:        []float64{},
			expResult: []float64{},
		},
		{
			ID:        testhelper.MkID("s1 nil"),
			s2:        []float64{2, 3, 4},
			expResult: []float64{2, 3, 4},
		},
		{
			ID:        testhelper.MkID("s2 nil"),
			s1:        []float64{1, 2, 3},
			expResult: []float64{1, 2, 3},
		},
		{
			ID:        testhelper.MkID("both nil"),
			expResult: []float64{},
		},
	}

	for _, tc := range testCases {
		result := merge(tc.s1, tc.s2)
		if differs(result, tc.expResult) {
			t.Log(tc.IDStr())
			t.Log("\t: Expected:", tc.expResult)
			t.Log("\t:      Got:", result)
			t.Errorf("\t: merge failed\n")
		}
	}
}

func statDiffers(s1, s2 stat) bool {
	if s1.count != s2.count {
		return true
	}
	if s1.sum != s2.sum {
		return true
	}
	if s1.sumSq != s2.sumSq {
		return true
	}
	if differs(s1.mins, s2.mins) {
		return true
	}
	if differs(s1.maxs, s2.maxs) {
		return true
	}
	return false
}

func TestAddVal(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		size    int
		vals    []float64
		expStat stat
	}{
		{
			ID:   testhelper.MkID("size 3, 4 vals"),
			size: 3,
			vals: []float64{10, 10, 10, 5},
			expStat: stat{
				count: 4,
				sum:   35,
				sumSq: 325,
				mins:  []float64{5, 10, 10},
				maxs:  []float64{10, 10, 10},
			},
		},
		{
			ID:   testhelper.MkID("size 3, 2 vals"),
			size: 3,
			vals: []float64{10, 5},
			expStat: stat{
				count: 2,
				sum:   15,
				sumSq: 125,
				mins:  []float64{5, 10},
				maxs:  []float64{5, 10},
			},
		},
	}

	for _, tc := range testCases {
		s := NewStatOrPanic(tc.size)
		for _, v := range tc.vals {
			s.addVal(v)
		}
		if statDiffers(*s, tc.expStat) {
			t.Log(tc.IDStr())
			t.Errorf("\t: addVal failed\n")
		}
	}
}

func TestMergeVal(t *testing.T) {
	stat1 := *NewStatOrPanic(5)

	testCases := []struct {
		testhelper.ID
		s1     stat
		s2     stat
		expVal stat
	}{
		{
			ID: testhelper.MkID("..."),
			s1: stat{
				count: 2,
				sum:   15,
				sumSq: 125,
				mins:  []float64{5, 10},
				maxs:  []float64{5, 10},
			},
			s2: stat{
				count: 2,
				sum:   15,
				sumSq: 125,
				mins:  []float64{5, 10},
				maxs:  []float64{5, 10},
			},
			expVal: stat{
				count: 4,
				sum:   30,
				sumSq: 250,
				mins:  []float64{5, 5},
				maxs:  []float64{10, 10},
			},
		},

		{
			ID: testhelper.MkID("empty stat1 with excess capacity"),
			s1: stat1,
			s2: stat{
				count: 2,
				sum:   15,
				sumSq: 125,
				mins:  []float64{5, 10},
				maxs:  []float64{5, 10},
			},
			expVal: stat{
				count: 2,
				sum:   15,
				sumSq: 125,
				mins:  []float64{5, 10},
				maxs:  []float64{5, 10},
			},
		},
	}

	for _, tc := range testCases {
		tc.s1.mergeVal(&tc.s2)
		if statDiffers(tc.s1, tc.expVal) {
			t.Log(tc.IDStr())
			t.Errorf("\t: mergeVal failed\n")
		}
	}
}
