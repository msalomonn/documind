package jobs

import (
	"github.com/msalomonn/documind/internal/pipeline"
	"testing"
	"time"
)

func TestRetryBounds(t *testing.T) {
	for _, tc := range []struct {
		attempt         int
		after, min, max time.Duration
	}{{1, 0, 5 * time.Second, 6 * time.Second}, {2, 0, 30 * time.Second, 36 * time.Second}, {1, time.Hour, 60 * time.Second, 60 * time.Second}} {
		for range 100 {
			d := RetryDelay(tc.attempt, tc.after)
			if d < tc.min || d > tc.max {
				t.Fatalf("delay %v outside [%v,%v]", d, tc.min, tc.max)
			}
		}
	}
}
func TestOutcomeValidation(t *testing.T) {
	o := Outcome{Result: pipeline.Result{SchemaVersion: 1, Title: "Synthetic", Summary: "Diagnostic only", KeyFacts: []pipeline.Fact{}}, Provider: "diagnostic", Model: "worker-smoke-v1"}
	if !validOutcome(o) {
		t.Fatal("valid outcome rejected")
	}
	o.Result.SchemaVersion = 2
	if validOutcome(o) {
		t.Fatal("unknown schema accepted")
	}
	o.Result.SchemaVersion = 1
	o.Result.Title = ""
	if validOutcome(o) {
		t.Fatal("empty result accepted")
	}
}
