package metrics

import (
	"math"
	"testing"
	"time"
)

func TestCalculate(t *testing.T) {
	got := Calculate(250, 225, time.Minute)

	assertClose(t, got.RawWPM, 50)
	assertClose(t, got.Accuracy, 0.9)
	assertClose(t, got.NetWPM, 45)
}

func TestCalculateHandlesEmptyOrZeroElapsed(t *testing.T) {
	for _, got := range []Result{
		Calculate(0, 0, time.Minute),
		Calculate(10, 10, 0),
	} {
		if got != (Result{}) {
			t.Fatalf("expected zero result, got %+v", got)
		}
	}
}

func assertClose(t *testing.T, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("got %.4f, want %.4f", got, want)
	}
}
