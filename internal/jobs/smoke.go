package jobs

import (
	"context"
	"github.com/msalomonn/documind/internal/documents"
	"github.com/msalomonn/documind/internal/pipeline"
	"io"
	"strings"
	"time"
)

const SmokePrefix = "DOCUMIND WORKER SMOKE\n"

// Smoke is an explicitly selected diagnostic, never a general extraction adapter.
type Smoke struct {
	Objects           documents.Objects
	Delay             time.Duration
	TransientFailures int
	Permanent         bool
}

func (s Smoke) Process(ctx context.Context, c Claim) (Outcome, error) {
	body, e := s.Objects.Open(ctx, c.ObjectKey)
	if e != nil {
		return Outcome{}, Failure{Code: "storage_unavailable", Transient: true}
	}
	defer body.Close()
	data, e := io.ReadAll(io.LimitReader(body, documents.MaxBytes+1))
	if e != nil {
		return Outcome{}, Failure{Code: "storage_unavailable", Transient: true}
	}
	if documents.Validate(c.Name, c.ContentType, data) != "" || !strings.HasPrefix(string(data), SmokePrefix) {
		return Outcome{}, Failure{Code: "invalid_input"}
	}
	timer := time.NewTimer(s.Delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return Outcome{}, ctx.Err()
	case <-timer.C:
	}
	if s.Permanent {
		return Outcome{}, Failure{Code: "smoke_failure"}
	}
	if c.Attempts <= s.TransientFailures {
		return Outcome{}, Failure{Code: "smoke_failure", Transient: true}
	}
	return Outcome{Result: pipeline.Result{SchemaVersion: 1, Title: "Worker diagnostic", Summary: "Synthetic worker lifecycle check completed. No AI extraction was performed.", KeyFacts: []pipeline.Fact{}}, Provider: "diagnostic", Model: "worker-smoke-v1"}, nil
}
