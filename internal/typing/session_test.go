package typing

import (
	"testing"
	"time"
)

func TestTypingCorrectIncorrectAndBackspace(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	session := NewSession("abc", time.Minute)

	session.TypeRune('a', now)
	session.TypeRune('x', now)

	if got := session.CorrectChars(); got != 1 {
		t.Fatalf("correct chars = %d, want 1", got)
	}

	session.Backspace()
	session.TypeRune('b', now)

	if got := session.CorrectChars(); got != 2 {
		t.Fatalf("correct chars = %d, want 2", got)
	}
	if got := session.TypedChars(); got != 2 {
		t.Fatalf("typed chars = %d, want 2", got)
	}
}

func TestCompleteStopsInputAndCapsElapsed(t *testing.T) {
	start := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	session := NewSession("abcdef", 15*time.Second)

	session.TypeRune('a', start)
	session.Complete(start.Add(30 * time.Second))
	session.TypeRune('b', start.Add(31*time.Second))
	session.Backspace()

	if got := session.TypedChars(); got != 1 {
		t.Fatalf("typed chars after done = %d, want 1", got)
	}
	if got := session.Elapsed(start.Add(time.Minute)); got != 15*time.Second {
		t.Fatalf("elapsed = %v, want 15s", got)
	}
}
