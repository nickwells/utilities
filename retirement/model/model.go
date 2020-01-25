package model

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"time"
)

type timer struct {
	start time.Time
	D     time.Duration
}

// TimeIt sets the start time and returns a function to be called when the
// operation is complete. It is suggested that you should pass this to defer
// as follows.
//
//     defer t.TimeIt()()
func (t *timer) TimeIt() func() {
	t.start = time.Now()
	return func() {
		t.D = time.Since(t.start)
	}
}

type metrics struct {
	durCalcValues timer
	threadCount   int64
}

// M records the parameters of the model. These can be changed to
// see the impact on the outcomes
type M struct {
	rtnMeanPct   float64
	rtnSDPct     float64
	minGrowthPct float64

	inflationPct float64

	targetIncome          float64
	minIncome             float64
	drawingPeriodsPerYear int64

	initialPortfolio float64

	crashInterval int64
	crashPct      float64

	yearsDefered int64
	years        int64
	trials       int64

	yearsToShow int64

	showIntroText   bool
	showModelParams bool

	showModelMetrics bool
	modelMetrics     metrics

	extremeSetSize int64
}

// New returns a new model with the default values set
func New() *M {
	return &M{
		rtnMeanPct:            7,
		rtnSDPct:              3,
		inflationPct:          2.5,
		drawingPeriodsPerYear: 12,
		years:                 30,
		trials:                250000,
		yearsToShow:           1,
		extremeSetSize:        10,
	}
}

// initResults creates the results slice and initialises any necessary values
func (m M) initResults() []*AggResults {
	results := make([]*AggResults, m.years)
	for y, r := range results {
		r = NewAggResultsOrPanic(int(m.extremeSetSize))
		r.year = int64(y)
		if r.year < m.yearsDefered {
			r.withdrawalDefered = true
		}
		results[y] = r
	}

	return results
}

// mergeResults ...
func (m M) mergeResults(results []*AggResults, rc <-chan []*AggResults, dc chan<- bool) {
	for subR := range rc {
		for i, r := range subR {
			val := results[i]

			val.crash += r.crash
			val.bust += r.bust
			val.surplusAvailable += r.surplusAvailable
			val.minimalIncome += r.minimalIncome
			val.portfolioDown += r.portfolioDown
			(val.portfolio).mergeVal(r.portfolio)
			(val.income).mergeVal(r.income)

			results[i] = val
		}
	}
	dc <- true
}

// trialRunner runs the model trials times and when it is finished it passes
// the results on over the results channel
func (m *M) trialRunner(trials int64, rc chan<- []*AggResults, tc chan bool) {
	results := m.initResults()

	s := new(state)
	s.rand = rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	for ; trials > 0; trials-- {
		s.setState(m)
		for y := int64(0); y < m.years; y++ {
			r := results[y]

			s.year = y
			s.calcCurrentRtn(r)
			s.calcCurrentIncome(r)
			s.calcNewPortfolio(r)
			if s.portfolio <= 0 {
				s.bust = true
				for ; y < m.years; y++ {
					r := results[y]
					r.bust++
				}
				break
			}

			s.adjustForInflation()
		}
	}
	rc <- results
	tc <- true
}

// CalcValues creates an AggResults slice and runs the model to populate
// it. It runs the model calculations in a pool of goroutines each of which
// calculates a proportion of the trials and then passes its own results to a
// separate goroutine which merges them together. When the merging is
// complete this routine returns the merged results.
func (m *M) CalcValues() []*AggResults {
	defer m.modelMetrics.durCalcValues.TimeIt()()

	results := m.initResults()

	poolSize := int64(runtime.NumCPU() - 1)
	if poolSize <= 0 {
		poolSize = 1
	}
	if poolSize > m.trials {
		poolSize = m.trials
	}
	m.modelMetrics.threadCount = poolSize

	trialsPerRunner := int64(math.Ceil(float64(m.trials) / float64(poolSize)))

	resultsChan := make(chan []*AggResults, poolSize*2)
	resultsGathered := make(chan bool)
	trialsComplete := make(chan bool)

	go m.mergeResults(results, resultsChan, resultsGathered)

	for i := int64(0); i < poolSize-1; i++ {
		go m.trialRunner(trialsPerRunner, resultsChan, trialsComplete)
	}
	go m.trialRunner(m.trials-(poolSize-1)*trialsPerRunner,
		resultsChan, trialsComplete)

	var runnerCnt int64
	for range trialsComplete {
		runnerCnt++
		if runnerCnt >= poolSize {
			break
		}
	}

	close(resultsChan)
	<-resultsGathered

	return results
}

// AggResults records the aggregate results over all the trials for each year
type AggResults struct {
	year              int64
	withdrawalDefered bool
	surplusAvailable  int
	minimalIncome     int
	crash             int
	bust              int
	portfolioDown     int
	portfolio         *stat
	income            *stat
}

// NewAggResults constructs a new AggResults value and returns a pointer to
// it. An error is returned if the size is less than 1
func NewAggResults(size int) (*AggResults, error) {
	if size < 1 {
		return nil,
			fmt.Errorf(
				"the size to be used for the stat members must be >= 1 (is %d)",
				size)
	}
	ar := &AggResults{
		portfolio: NewStatOrPanic(size),
		income:    NewStatOrPanic(size),
	}
	return ar, nil
}

// NewAggResultsOrPanic constructs a new AggResults value and returns a
// pointer to it.
func NewAggResultsOrPanic(size int) *AggResults {
	ar, err := NewAggResults(size)
	if err != nil {
		panic(err)
	}
	return ar
}

// state holds the current state of the model
type state struct {
	model *M
	rand  *rand.Rand

	year int64

	portfolio        float64
	initialPortfolio float64
	bust             bool

	currentRtn float64
	rtnMean    float64
	rtnSD      float64
	minGrowth  float64
	crashProp  float64

	currentIncome float64
	targetIncome  float64
	minIncome     float64

	inflationAdjustment float64
	yearlyInflation     float64
}

// setState sets the state to its initial values from the model parameters
// supplied
func (s *state) setState(m *M) {
	s.model = m

	s.portfolio = m.initialPortfolio
	s.initialPortfolio = m.initialPortfolio
	s.bust = false

	s.currentRtn = m.rtnMeanPct / 100
	s.rtnMean = m.rtnMeanPct / 100
	s.rtnSD = m.rtnSDPct / 100
	s.minGrowth = (m.inflationPct + m.minGrowthPct) / 100
	s.crashProp = m.crashPct / 100

	s.currentIncome = m.targetIncome
	s.targetIncome = m.targetIncome
	s.minIncome = m.minIncome

	s.inflationAdjustment = 1
	s.yearlyInflation = 1 + (m.inflationPct / 100)
}

// calcCurrentIncome sets the income to be taken in the forthcoming year. It
// assumes that the next year will have the same return as last year and from
// that works out the available income. Then it calculates the growth we want
// to see each year (inflation plus the minimum growth) and subtracts that
// amount from the available income. Lastly it ensures that the income we
// take will be between the target and the minimum.
func (s *state) calcCurrentIncome(r *AggResults) {
	if r.withdrawalDefered {
		s.currentIncome = 0
		return
	}

	r.income.addVal(s.currentIncome / s.inflationAdjustment)

	availableInc := s.portfolio * s.currentRtn
	desiredGrowth := s.portfolio * s.minGrowth
	s.currentIncome = availableInc - desiredGrowth

	if s.currentIncome > s.targetIncome {
		s.currentIncome = s.targetIncome
		r.surplusAvailable++
	} else if s.currentIncome < s.minIncome {
		s.currentIncome = s.minIncome
		r.minimalIncome++
	}
}

// calcCurrentRtn calculates the return for the coming year. Each year there
// is a 1 in crashInterval chance that the market will 'crash' meaning that
// the return is set to the crash proportion
func (s *state) calcCurrentRtn(r *AggResults) {
	s.currentRtn = s.rtnMean + (s.rand.NormFloat64() * s.rtnSD)

	if s.model.crashInterval > 0 &&
		s.rand.Float64() < 1/float64(s.model.crashInterval) {
		s.currentRtn = -1 * s.crashProp

		r.crash++
	}
}

// adjustForInflation adjusts the values for inflation
func (s *state) adjustForInflation() {
	s.inflationAdjustment *= s.yearlyInflation
	s.targetIncome *= s.yearlyInflation
	s.minIncome *= s.yearlyInflation
}

// calcNewPortfolio set the end-of-year portfolio value according to the
// model after income is taken out and the growth has taken place
func (s *state) calcNewPortfolio(r *AggResults) {
	ppy := float64(s.model.drawingPeriodsPerYear)
	periodMult := math.Pow(1+s.currentRtn, 1.0/ppy)
	periodIncome := s.currentIncome / ppy
	if s.model.yearsDefered > s.year {
		periodIncome = 0
	}

	for i := 0; i < int(s.model.drawingPeriodsPerYear); i++ {
		s.portfolio -= periodIncome
		if s.portfolio < 0 {
			s.portfolio = 0
			break
		}
		s.portfolio *= periodMult
	}

	if s.portfolio/s.inflationAdjustment < s.initialPortfolio {
		r.portfolioDown++
	}

	r.portfolio.addVal(s.portfolio / s.inflationAdjustment)
}
