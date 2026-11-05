package jobs

import (
	"encoding/json"
	"errors"
	"github.com/msalomonn/documind/internal/pipeline"
	"math/rand/v2"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrEmpty = errors.New("no_eligible_job")
var ErrLost = errors.New("claim_lost")
var ErrInvalidResult = errors.New("invalid_result")

type Config struct{ Lease, Heartbeat, Attempt, Poll time.Duration }

func Defaults() Config {
	return Config{Lease: 30 * time.Second, Heartbeat: 10 * time.Second, Attempt: 60 * time.Second, Poll: time.Second}
}
func (c Config) Valid() bool {
	return c.Lease >= 100*time.Millisecond && c.Lease <= time.Minute && c.Heartbeat >= 10*time.Millisecond && c.Heartbeat < c.Lease/2 && c.Attempt >= c.Lease && c.Attempt <= 5*time.Minute && c.Poll >= 10*time.Millisecond && c.Poll <= time.Minute
}

type Outcome struct {
	Result          pipeline.Result
	Provider, Model string
}
type Failure struct {
	Code       string
	Transient  bool
	RetryAfter time.Duration
}

func (f Failure) Error() string { return f.Code }
func RetryDelay(attempt int, after time.Duration) time.Duration {
	base := 5 * time.Second
	if attempt >= 2 {
		base = 30 * time.Second
	}
	delay := base + time.Duration(rand.Int64N(int64(base/5)+1))
	if after > delay {
		delay = after
	}
	if delay > time.Minute {
		delay = time.Minute
	}
	return delay
}
func safeCode(s string) string {
	switch s {
	case "timeout", "rate_limited", "provider_unavailable", "storage_unavailable", "invalid_input", "invalid_result", "lease_expired_exhausted", "smoke_failure":
		return s
	default:
		return "processor_failed"
	}
}
func validOutcome(o Outcome) bool {
	r := o.Result
	bounded := func(s string, n int) bool { return utf8.ValidString(s) && len(s) <= n && strings.TrimSpace(s) != "" }
	if r.SchemaVersion != 1 || !bounded(r.Title, 512) || !bounded(r.Summary, 8000) || r.KeyFacts == nil || len(r.KeyFacts) > 50 || !bounded(o.Provider, 80) || !bounded(o.Model, 120) {
		return false
	}
	for _, f := range r.KeyFacts {
		if !bounded(f.Text, 2000) || !bounded(f.SourceExcerpt, 4000) {
			return false
		}
	}
	b, e := json.Marshal(r)
	return e == nil && len(b) <= 64*1024
}
