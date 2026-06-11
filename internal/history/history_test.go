package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendCreatesCSVWithHeaderAndLoadReadsRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "typing_history.csv")
	now := time.Date(2026, 6, 11, 12, 30, 0, 0, time.UTC)

	want := Run{
		Timestamp:       now,
		Language:        "python",
		DurationSeconds: 60,
		RawWPM:          70.126,
		Accuracy:        0.95,
		NetWPM:          66.61875,
		CorrectChars:    333,
		TypedChars:      350,
	}

	if err := Append(path, want); err != nil {
		t.Fatalf("append: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if got, wantHeader := string(content[:len(header[0])]), "timestamp"; got != wantHeader {
		t.Fatalf("expected header prefix %q, got %q", wantHeader, got)
	}

	runs, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	got := runs[0]
	if got.Language != want.Language || got.DurationSeconds != want.DurationSeconds || got.CorrectChars != want.CorrectChars || got.TypedChars != want.TypedChars {
		t.Fatalf("loaded wrong run: %+v", got)
	}
	if got.RawWPM != 70.13 || got.Accuracy != 0.95 || got.NetWPM != 66.62 {
		t.Fatalf("loaded rounded metrics incorrectly: %+v", got)
	}
}

func TestLoadMissingFileReturnsEmptyRuns(t *testing.T) {
	runs, err := Load(filepath.Join(t.TempDir(), "missing.csv"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected no runs, got %d", len(runs))
	}
}
