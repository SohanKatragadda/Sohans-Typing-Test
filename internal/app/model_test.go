package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"typing-test-tui/internal/history"
	"typing-test-tui/internal/typing"
)

func TestSplashAndCountdownTransitions(t *testing.T) {
	base := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.splashStarted = base

	updated, _ := model.Update(tickMsg(base.Add(time.Second)))
	model = updated.(Model)
	if model.screen != screenHome {
		t.Fatalf("screen after splash = %v, want home", model.screen)
	}

	model.prepareCountdown(base)
	updated, _ = model.Update(tickMsg(base.Add(4 * time.Second)))
	model = updated.(Model)
	if model.screen != screenRunning {
		t.Fatalf("screen after countdown = %v, want running", model.screen)
	}
	if !model.session.Started {
		t.Fatal("session should start when countdown completes")
	}
}

func TestTestsTabNavigationAndConfirmationFlow(t *testing.T) {
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.screen = screenHome

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(Model)
	if model.durationIndex != 3 {
		t.Fatalf("duration index = %d, want 3", model.durationIndex)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.focus != focusLanguage || !model.durationConfirmed {
		t.Fatalf("expected confirmed duration and language focus, got focus=%v confirmed=%v", model.focus, model.durationConfirmed)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = updated.(Model)
	if model.languageIndex != 1 {
		t.Fatalf("language index = %d, want 1", model.languageIndex)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.focus != focusStart || !model.languageConfirmed {
		t.Fatalf("expected start focus and confirmed language, got focus=%v confirmed=%v", model.focus, model.languageConfirmed)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.focus != focusLanguage || model.languageConfirmed {
		t.Fatalf("escape from start should return to language selection, got focus=%v confirmed=%v", model.focus, model.languageConfirmed)
	}
}

func TestHomeTabsRenderAsBoxesWithShortcutLabels(t *testing.T) {
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.screen = screenHome

	view := model.homeView()

	for _, want := range []string{"Tests (t)", "Past Results (p)", lipgloss.NormalBorder().TopLeft} {
		if !strings.Contains(view, want) {
			t.Fatalf("home view missing %q:\n%s", want, view)
		}
	}
}

func TestFocusedTestsOptionRendersWithBlink(t *testing.T) {
	if !focusStyle.GetBlink() {
		t.Fatal("focused option style should blink")
	}
}

func TestCountdownIgnoresTyping(t *testing.T) {
	base := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.prepareCountdown(base)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)

	if got := model.session.TypedChars(); got != 0 {
		t.Fatalf("typed chars during countdown = %d, want 0", got)
	}
}

func TestRunningTabKeyInputsFourSpacesForCodingLanguages(t *testing.T) {
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.screen = screenRunning
	model.languageIndex = 1
	model.session = typing.NewSession([]string{"    "}, time.Minute)
	model.session.Start(time.Now())

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)

	if got := string(model.session.Typed[0]); got != "    " {
		t.Fatalf("typed = %q, want four spaces", got)
	}
	if got := model.session.CorrectChars(); got != 4 {
		t.Fatalf("correct chars = %d, want 4", got)
	}
}

func TestRunningTabKeyDoesNothingForEnglish(t *testing.T) {
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.screen = screenRunning
	model.languageIndex = 0
	model.session = typing.NewSession([]string{"    "}, time.Minute)
	model.session.Start(time.Now())

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)

	if got := model.session.TypedChars(); got != 0 {
		t.Fatalf("typed chars = %d, want 0", got)
	}
}

func TestRenderSegmentShowsSpacesForTabRune(t *testing.T) {
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.session = typing.NewSession([]string{"a\tb"}, time.Minute)

	got := model.renderSegment(20, 3)

	if strings.Contains(got, "\\t") {
		t.Fatalf("rendered segment should not show tab marker, got %q", got)
	}
	if !strings.Contains(got, "    ") {
		t.Fatalf("rendered segment should show spaces for tab, got %q", got)
	}
}

func TestRunningInputLockingAndPlainQR(t *testing.T) {
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.screen = screenRunning
	model.tab = tabTests
	model.session = typing.NewSession([]string{"tpqrasd"}, time.Minute)
	model.session.Start(time.Now())

	for _, r := range []rune("tpqrasd") {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model = updated.(Model)
	}

	if model.screen != screenRunning {
		t.Fatalf("plain q/r should not leave running screen, got %v", model.screen)
	}
	if got := model.session.TypedChars(); got != 7 {
		t.Fatalf("typed chars = %d, want 7", got)
	}
}

func TestRunningArrowKeysAreIgnored(t *testing.T) {
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.screen = screenRunning
	model.session = typing.NewSession([]string{"abc"}, time.Minute)
	model.session.Start(time.Now())

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(Model)

	if got := model.session.TypedChars(); got != 0 {
		t.Fatalf("typed chars after arrow = %d, want 0", got)
	}
	if model.screen != screenRunning {
		t.Fatalf("screen = %v, want running", model.screen)
	}
}

func TestSecondToLastSegmentGeneratesMoreContent(t *testing.T) {
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.screen = screenRunning
	model.session = typing.NewSession([]string{"a", "b"}, time.Minute)
	model.session.Start(time.Now())

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = updated.(Model)

	if len(model.session.Segments) <= 2 {
		t.Fatalf("expected appended segments, got %d", len(model.session.Segments))
	}
}

func TestEnsureUpcomingSegmentsAvoidsExistingPrompts(t *testing.T) {
	model := New(filepath.Join(t.TempDir(), "history.csv"))
	model.languageIndex = 0
	model.session = typing.NewSession([]string{"calm focus turns a small practice session into a reliable habit."}, time.Minute)

	model.ensureUpcomingSegments()

	seen := map[string]bool{}
	for _, segment := range model.session.SegmentStrings() {
		if seen[segment] {
			t.Fatalf("duplicate prompt appended: %q", segment)
		}
		seen[segment] = true
	}
}

func TestCtrlQAbortDoesNotSaveAndCtrlRRestartsCountdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.csv")
	model := New(path)
	model.screen = screenRunning
	model.session = typing.NewSession([]string{"abc"}, time.Minute)
	now := time.Now()
	model.session.Start(now)
	model.session.TypeRune('a', now)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	model = updated.(Model)
	if model.screen != screenHome {
		t.Fatalf("screen after ctrl+q = %v, want home", model.screen)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("abort should not create history file, stat err=%v", err)
	}

	model.screen = screenRunning
	model.session = typing.NewSession([]string{"abc"}, time.Minute)
	model.session.Start(time.Now())
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	model = updated.(Model)
	if model.screen != screenCountdown {
		t.Fatalf("screen after ctrl+r = %v, want countdown", model.screen)
	}
	if got := model.session.TypedChars(); got != 0 {
		t.Fatalf("typed chars after restart = %d, want 0", got)
	}
}

func TestCompletedTimedAttemptIsSaved(t *testing.T) {
	base := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "history.csv")
	model := New(path)
	model.screen = screenRunning
	model.session = typing.NewSession([]string{"abc"}, time.Second)
	model.session.Start(base)
	model.session.TypeRune('a', base)

	updated, _ := model.Update(tickMsg(base.Add(time.Second)))
	model = updated.(Model)
	if model.screen != screenHome || model.tab != tabPastResults {
		t.Fatalf("expected home past-results after completion, got screen=%v tab=%v", model.screen, model.tab)
	}

	runs, err := history.Load(path)
	if err != nil {
		t.Fatalf("load history: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("history runs = %d, want 1", len(runs))
	}
}
