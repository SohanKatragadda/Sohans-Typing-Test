package typing

import (
	"time"

	"typing-test-tui/internal/metrics"
)

type Session struct {
	Target   []rune
	Typed    []rune
	Duration time.Duration
	Started  bool
	Done     bool

	startedAt time.Time
	endedAt   time.Time
}

func NewSession(target string, duration time.Duration) Session {
	return Session{
		Target:   []rune(target),
		Duration: duration,
		Typed:    make([]rune, 0, len([]rune(target))),
	}
}

func (s *Session) TypeRune(r rune, now time.Time) {
	if s.Done {
		return
	}
	if !s.Started {
		s.Start(now)
	}
	s.Typed = append(s.Typed, r)
}

func (s *Session) Start(now time.Time) {
	if s.Started {
		return
	}
	s.Started = true
	s.startedAt = now
}

func (s *Session) Backspace() {
	if s.Done || len(s.Typed) == 0 {
		return
	}
	s.Typed = s.Typed[:len(s.Typed)-1]
}

func (s *Session) Complete(now time.Time) {
	if s.Done {
		return
	}
	if !s.Started {
		s.Start(now)
	}
	s.Done = true
	s.endedAt = now
}

func (s Session) CorrectChars() int {
	correct := 0
	for i, typed := range s.Typed {
		if i < len(s.Target) && typed == s.Target[i] {
			correct++
		}
	}
	return correct
}

func (s Session) TypedChars() int {
	return len(s.Typed)
}

func (s Session) Elapsed(now time.Time) time.Duration {
	if !s.Started {
		return 0
	}
	end := now
	if s.Done {
		end = s.endedAt
	}
	elapsed := end.Sub(s.startedAt)
	if elapsed < 0 {
		return 0
	}
	if elapsed > s.Duration {
		return s.Duration
	}
	return elapsed
}

func (s Session) Remaining(now time.Time) time.Duration {
	remaining := s.Duration - s.Elapsed(now)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (s Session) Metrics(now time.Time) metrics.Result {
	return metrics.Calculate(s.TypedChars(), s.CorrectChars(), s.Elapsed(now))
}
