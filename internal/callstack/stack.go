package callstack

import (
	"fmt"
	"strings"
	"time"

	"github.com/nickwells/timer.mod/timer"
	"github.com/nickwells/verbose.mod/verbose"
)

const maxStackWidth = 30

// Stack used in conjunction with the timer and verbose packages this
// will print out how long a function took to run
type Stack struct {
	ShowTimings bool
	stack       []string
}

// Start prints the Start message, starts a timer and returns the function
// to be called at the end.
func (s *Stack) Start(tag, msg string) func() {
	s.stack = append(s.stack, tag)
	if verbose.IsOn() {
		fmt.Println(s.Tag(), msg)
	} else if s.ShowTimings {
		fmt.Println(s.Tag(), "Start")
	} else {
		return func() { s.popStack() }
	}

	return timer.Start(tag, s)
}

// Tag returns a stacked tag reflecting the current stack depth and
// right-filled.
func (s *Stack) Tag() string {
	t := strings.Repeat("|    ", len(s.stack)-1) +
		s.stack[len(s.stack)-1]
	if len(t) < maxStackWidth {
		t += strings.Repeat(".", maxStackWidth-len(t))
	}
	t += ":"
	return t
}

// popStack removes the last stack entry
func (s *Stack) popStack() {
	s.stack = s.stack[:len(s.stack)-1]
}

// Act satisfies the action function interface for a timer. It prints out the
// tag and the duration in milliseconds if the program is in verbose mode
func (s *Stack) Act(_ string, d time.Duration) {
	tag := s.Tag()
	s.popStack()

	if verbose.IsOn() || s.ShowTimings {
		fmt.Printf("%s%12.3f msecs\n",
			tag, float64(d/time.Microsecond)/1000.0)
		//fmt.Printf("%s------------\n", strings.Repeat(" ", len(tag)))
	}
}
