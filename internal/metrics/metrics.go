package metrics

import "time"

type Result struct {
	RawWPM   float64
	Accuracy float64
	NetWPM   float64
}

func Calculate(typedChars int, correctChars int, elapsed time.Duration) Result {
	if typedChars <= 0 || elapsed <= 0 {
		return Result{}
	}

	minutes := elapsed.Minutes()
	raw := (float64(typedChars) / 5.0) / minutes
	accuracy := float64(correctChars) / float64(typedChars)

	return Result{
		RawWPM:   raw,
		Accuracy: accuracy,
		NetWPM:   raw * accuracy,
	}
}
