package model

import (
	"fmt"
	"math"
	"sort"
)

// stat records a statistic
type stat struct {
	count int
	sum   float64
	sumSq float64
	mins  []float64
	maxs  []float64
}

// NewStat constructs a new stat value and returns a pointer to it. An error
// is returned if the size is less than 1
func NewStat(size int) (*stat, error) {
	if size < 1 {
		return nil,
			fmt.Errorf(
				"the capacity of the min and max slices must be >= 1 (is %d)",
				size)
	}
	s := &stat{}
	s.mins = make([]float64, 0, size)
	s.maxs = make([]float64, 0, size)
	return s, nil
}

// NewStatOrPanic constructs a new stat value and returns a pointer to t. It
// will panic if there are any errors returned while creating it
func NewStatOrPanic(size int) *stat {
	s, err := NewStat(size)
	if err != nil {
		panic(err)
	}
	return s
}

type discardType int

const (
	DropFromStart discardType = iota
	DropFromEnd
)

// addVal adds a new value to the stat
func (s *stat) addVal(val float64) {
	maxIdx := cap(s.mins) - 1

	s.count++
	s.sum += val
	s.sumSq += (val * val)
	if s.count <= cap(s.mins) {
		s.mins = append(s.mins, val)
		s.maxs = append(s.maxs, val)
		sort.Float64s(s.mins)
		sort.Float64s(s.maxs)
	} else {
		if val < s.mins[maxIdx] { // smaller than the largest min value
			insert(val, s.mins, DropFromEnd)
		}
		if val > s.maxs[0] { // larger than the smallest max value
			insert(val, s.maxs, DropFromStart)
		}
	}
}

// insert inserts the val into the vals shifting the remaining values along
// and discarding from one end or the other according to the discard
// type. The vals slice is assumed to be sorted in ascending order.
func insert(val float64, vals []float64, discard discardType) {
	var i int
	var cmp float64

	for i, cmp = range vals {
		if cmp >= val {
			break
		}
	}

	if discard == DropFromEnd {
		if i+1 < len(vals) {
			copy(vals[i+1:], vals[i:len(vals)-1])
		}
	} else if discard == DropFromStart {
		if i > 0 {
			copy(vals[:i], vals[1:i+1])
		}
	}
	vals[i] = val
}

// merge combines the two slices and sorts them, it returns the combined
// slice
func merge(s1, s2 []float64) []float64 {
	agg := make([]float64, 0, len(s1)+len(s2))
	agg = append(agg, s1...)
	agg = append(agg, s2...)
	sort.Float64s(agg)
	return agg
}

// mergeVal combines the stats
func (s *stat) mergeVal(s2 *stat) {
	aggMins := merge(s.mins, s2.mins)
	end := cap(s.mins)
	if end > len(aggMins) {
		end = len(aggMins)
	}
	s.mins = append(s.mins[:0], aggMins[0:end]...)

	aggMaxs := merge(s.maxs, s2.maxs)
	start := len(aggMaxs) - cap(s.maxs)
	if start < 0 {
		start = 0
	}
	s.maxs = append(s.maxs[:0], aggMaxs[start:]...)

	s.count += s2.count
	s.sum += s2.sum
	s.sumSq += s2.sumSq
}

// calcMean will calculate the average value of the entries in the slice
// which must not be empty
func calcMean(s []float64) float64 {
	var sum float64
	for _, v := range s {
		sum += v
	}
	return sum / float64(len(s))
}

// vals returns the calculated values from the stat
func (s stat) vals() (min, avg, sd, max float64, count int) {
	if s.count == 0 {
		return
	}
	min = calcMean(s.mins)
	avg = s.sum / float64(s.count)
	sd = 0
	if s.count > 1 {
		sd = math.Sqrt((s.sumSq / float64(s.count-1)) - (avg * avg))
	}
	max = calcMean(s.maxs)
	count = s.count
	return
}
