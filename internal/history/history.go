package history

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

var header = []string{
	"timestamp",
	"language",
	"duration_seconds",
	"raw_wpm",
	"accuracy",
	"net_wpm",
	"correct_chars",
	"typed_chars",
}

type Run struct {
	Timestamp       time.Time
	Language        string
	DurationSeconds int
	RawWPM          float64
	Accuracy        float64
	NetWPM          float64
	CorrectChars    int
	TypedChars      int
}

func Append(path string, run Run) error {
	needsHeader := false
	if info, err := os.Stat(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		needsHeader = true
	} else if info.Size() == 0 {
		needsHeader = true
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if needsHeader {
		if err := writer.Write(header); err != nil {
			return err
		}
	}
	if err := writer.Write(encodeRun(run)); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func Load(path string) ([]Run, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	start := 0
	if sameHeader(records[0]) {
		start = 1
	}

	runs := make([]Run, 0, len(records)-start)
	for i, record := range records[start:] {
		run, err := decodeRun(record)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+start+1, err)
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func encodeRun(run Run) []string {
	return []string{
		run.Timestamp.Format(time.RFC3339),
		run.Language,
		strconv.Itoa(run.DurationSeconds),
		fmt.Sprintf("%.2f", run.RawWPM),
		fmt.Sprintf("%.4f", run.Accuracy),
		fmt.Sprintf("%.2f", run.NetWPM),
		strconv.Itoa(run.CorrectChars),
		strconv.Itoa(run.TypedChars),
	}
}

func decodeRun(record []string) (Run, error) {
	if len(record) != len(header) {
		return Run{}, fmt.Errorf("expected %d fields, got %d", len(header), len(record))
	}

	timestamp, err := time.Parse(time.RFC3339, record[0])
	if err != nil {
		return Run{}, err
	}
	duration, err := strconv.Atoi(record[2])
	if err != nil {
		return Run{}, err
	}
	rawWPM, err := strconv.ParseFloat(record[3], 64)
	if err != nil {
		return Run{}, err
	}
	accuracy, err := strconv.ParseFloat(record[4], 64)
	if err != nil {
		return Run{}, err
	}
	netWPM, err := strconv.ParseFloat(record[5], 64)
	if err != nil {
		return Run{}, err
	}
	correct, err := strconv.Atoi(record[6])
	if err != nil {
		return Run{}, err
	}
	typed, err := strconv.Atoi(record[7])
	if err != nil {
		return Run{}, err
	}

	return Run{
		Timestamp:       timestamp,
		Language:        record[1],
		DurationSeconds: duration,
		RawWPM:          rawWPM,
		Accuracy:        accuracy,
		NetWPM:          netWPM,
		CorrectChars:    correct,
		TypedChars:      typed,
	}, nil
}

func sameHeader(record []string) bool {
	if len(record) != len(header) {
		return false
	}
	for i := range header {
		if record[i] != header[i] {
			return false
		}
	}
	return true
}
