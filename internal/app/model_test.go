package app

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"typing-test-tui/internal/typing"
)

func TestSpaceKeyInputsSpaceRune(t *testing.T) {
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.session = typing.NewSession("a b", time.Minute)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model = updated.(Model)

	if got := string(model.session.Typed); got != "a " {
		t.Fatalf("typed = %q, want %q", got, "a ")
	}
	if got := model.session.CorrectChars(); got != 2 {
		t.Fatalf("correct chars = %d, want 2", got)
	}
}
