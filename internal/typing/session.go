package typing

import (
	"time"
	"unicode"

	"typing-test-tui/internal/metrics"
)

type Session struct {
	Segments [][]rune
	Typed    [][]rune
	Duration time.Duration
	Started  bool
	Done     bool

	Current int

	startedAt time.Time
	endedAt   time.Time
}

type WordState struct {
	Start           int
	End             int
	Current         bool
	CompleteCorrect bool
}

func NewSession(segments []string, duration time.Duration) Session {
	session := Session{
		Segments: make([][]rune, 0, len(segments)),
		Typed:    make([][]rune, 0, len(segments)),
		Duration: duration,
	}
	session.AppendSegments(segments)
	return session
}

func (s *Session) AppendSegments(segments []string) {
	for _, segment := range segments {
		runes := []rune(segment)
		if len(runes) == 0 {
			continue
		}
		s.Segments = append(s.Segments, runes)
		s.Typed = append(s.Typed, make([]rune, 0, len(runes)))
	}
}

func (s *Session) TypeRune(r rune, now time.Time) bool {
	if s.Done || len(s.Segments) == 0 {
		return false
	}
	if !s.Started {
		s.Start(now)
	}

	s.Typed[s.Current] = append(s.Typed[s.Current], r)
	if len(s.Typed[s.Current]) < len(s.Segments[s.Current]) {
		return false
	}

	if s.Current < len(s.Segments)-1 {
		s.Current++
		return true
	}
	return false
}

func (s *Session) Start(now time.Time) {
	if s.Started {
		return
	}
	s.Started = true
	s.startedAt = now
}

func (s *Session) Backspace() {
	if s.Done || len(s.Segments) == 0 {
		return
	}
	if len(s.Typed[s.Current]) > 0 {
		s.Typed[s.Current] = s.Typed[s.Current][:len(s.Typed[s.Current])-1]
		return
	}
	if s.Current > 0 {
		s.Current--
	}
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

func (s Session) CurrentSegment() []rune {
	if len(s.Segments) == 0 {
		return nil
	}
	return s.Segments[s.Current]
}

func (s Session) CurrentTyped() []rune {
	if len(s.Typed) == 0 {
		return nil
	}
	return s.Typed[s.Current]
}

func (s Session) SegmentStrings() []string {
	segments := make([]string, 0, len(s.Segments))
	for _, segment := range s.Segments {
		segments = append(segments, string(segment))
	}
	return segments
}

func (s Session) NeedsMoreSegments() bool {
	return len(s.Segments)-s.Current <= 2
}

func (s Session) CorrectChars() int {
	correct := 0
	for segmentIndex, typed := range s.Typed {
		target := s.Segments[segmentIndex]
		for i, r := range typed {
			if i < len(target) && r == target[i] {
				correct++
			}
		}
	}
	return correct
}

func (s Session) TypedChars() int {
	typed := 0
	for _, segment := range s.Typed {
		typed += len(segment)
	}
	return typed
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

func WordStates(target []rune, typed []rune) []WordState {
	states := make([]WordState, 0)
	cursor := len(typed)

	for i := 0; i < len(target); {
		for i < len(target) && unicode.IsSpace(target[i]) {
			i++
		}
		if i >= len(target) {
			break
		}

		start := i
		for i < len(target) && !unicode.IsSpace(target[i]) {
			i++
		}
		end := i
		state := WordState{
			Start:   start,
			End:     end,
			Current: cursor >= start && cursor < end,
		}
		if len(typed) >= end {
			state.CompleteCorrect = true
			for j := start; j < end; j++ {
				if j >= len(typed) || typed[j] != target[j] {
					state.CompleteCorrect = false
					break
				}
			}
		}
		states = append(states, state)
	}

	return states
}
