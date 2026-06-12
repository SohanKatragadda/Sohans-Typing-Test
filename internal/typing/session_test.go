package typing

import (
	"testing"
	"time"
)

func TestSegmentAdvanceAndBackspaceRestoresPreviousTypedContent(t *testing.T) {
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	session := NewSession([]string{"abc", "de"}, time.Minute)

	session.TypeRune('a', now)
	session.TypeRune('x', now)
	session.TypeRune('c', now)

	if session.Current != 1 {
		t.Fatalf("current segment = %d, want 1", session.Current)
	}

	session.Backspace()

	if session.Current != 0 {
		t.Fatalf("current segment after backspace = %d, want 0", session.Current)
	}
	if got := string(session.CurrentTyped()); got != "axc" {
		t.Fatalf("restored typed content = %q, want %q", got, "axc")
	}
}

func TestBackspaceAtBeginningOfFirstSegmentDoesNothing(t *testing.T) {
	session := NewSession([]string{"abc"}, time.Minute)

	session.Backspace()

	if session.Current != 0 {
		t.Fatalf("current segment = %d, want 0", session.Current)
	}
	if got := session.TypedChars(); got != 0 {
		t.Fatalf("typed chars = %d, want 0", got)
	}
}

func TestCompleteStopsInputAndCapsElapsed(t *testing.T) {
	start := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	session := NewSession([]string{"abcdef"}, 15*time.Second)

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

func TestWordStatesOnlyCompleteCorrectWords(t *testing.T) {
	target := []rune("cat dog")

	incorrect := WordStates(target, []rune("cot"))
	if incorrect[0].CompleteCorrect {
		t.Fatal("incorrect word should not be complete-correct")
	}

	corrected := WordStates(target, []rune("cat"))
	if !corrected[0].CompleteCorrect {
		t.Fatal("corrected word should be complete-correct")
	}
}
