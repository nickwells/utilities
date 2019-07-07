package model

import "math"

// stat records a statistic
type stat struct {
	count int
	sum   float64
	sumSq float64
	min   float64
	max   float64
}

// addVal adds a new value to the stat
func (s *stat) addVal(val float64) {
	s.count++
	s.sum += val
	s.sumSq += (val * val)
	if s.count == 1 {
		s.min = val
		s.max = val
		return
	}
	if val < s.min {
		s.min = val
	}
	if val > s.max {
		s.max = val
	}
}

// mergeVal combines the stats
func (s *stat) mergeVal(s2 stat) {
	if s.count == 0 {
		s.min = s2.min
		s.max = s2.max
	} else {
		if s2.min < s.min {
			s.min = s2.min
		}
		if s2.max > s.max {
			s.max = s2.max
		}
	}

	s.count += s2.count
	s.sum += s2.sum
	s.sumSq += s2.sumSq
}

// vals returns the calculated values from the stat
func (s stat) vals() (min, avg, sd, max float64, count int) {
	if s.count == 0 {
		return
	}
	min = s.min
	avg = s.sum / float64(s.count)
	sd = 0
	if s.count > 1 {
		sd = math.Sqrt((s.sumSq / float64(s.count-1)) - (avg * avg))
	}
	max = s.max
	count = s.count
	return
}
