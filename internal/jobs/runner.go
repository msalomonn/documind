package jobs

import (
	"context"
	"errors"
	"time"
)

type Processor interface {
	Process(context.Context, Claim) (Outcome, error)
}
type Runner struct {
	Queue      *Queue
	Processor  Processor
	DocumentID string
}

func (r Runner) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		e := r.Once(ctx)
		if e != nil && !errors.Is(e, ErrEmpty) && !errors.Is(e, ErrLost) {
			return e
		}
		timer := time.NewTimer(r.Queue.Config.Poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
func (r Runner) Once(ctx context.Context) error {
	c, e := r.Queue.Claim(ctx, r.DocumentID)
	if e != nil {
		return e
	}
	// Reserve time inside the fenced deadline to persist a timeout/retry safely.
	grace := min(5*time.Second, r.Queue.Config.Attempt/5)
	attempt, cancel := context.WithDeadline(ctx, c.Deadline.Add(-grace))
	defer cancel()
	heartbeat, stop := context.WithCancel(attempt)
	done := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(r.Queue.Config.Heartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeat.Done():
				done <- nil
				return
			case <-ticker.C:
				if e := r.Queue.Renew(heartbeat, c); e != nil {
					if heartbeat.Err() != nil {
						done <- nil
						return
					}
					cancel()
					done <- e
					return
				}
			}
		}
	}()
	outcome, processErr := r.Processor.Process(attempt, c)
	attemptErr := attempt.Err()
	stop()
	renewErr := <-done
	if ctx.Err() != nil {
		return ctx.Err()
	} // shutdown leaves the lease for recovery
	if renewErr != nil {
		return renewErr
	} // no mutation after losing ownership
	if attemptErr != nil {
		processErr = attemptErr
	}
	if processErr == nil {
		e = r.Queue.Complete(ctx, c, outcome)
		if !errors.Is(e, ErrInvalidResult) {
			return e
		}
		return r.Queue.Fail(ctx, c, Failure{Code: "invalid_result"})
	}
	failure := Failure{Code: "processor_failed"}
	if !errors.As(processErr, &failure) && errors.Is(processErr, context.DeadlineExceeded) {
		failure = Failure{Code: "timeout", Transient: true}
	}
	return r.Queue.Fail(ctx, c, failure)
}
