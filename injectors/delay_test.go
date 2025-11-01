package injectors

import (
	"context"
	"testing"
	"time"
)

func TestDelay_ProbabilityMode_GetChaosDelay(t *testing.T) {
	di := RandomDelayWithProbability(5*time.Millisecond, 5*time.Millisecond, 1.0)
	if err := di.Inject(context.Background()); err != nil {
		t.Fatalf("inject err: %v", err)
	}
	d, ok := di.GetChaosDelay()
	if !ok {
		t.Fatalf("expected delay to be applied")
	}
	if d != 5*time.Millisecond {
		t.Fatalf("expected 5ms, got %v", d)
	}
	if di.GetDelayCount() == 0 {
		t.Fatalf("expected delay count > 0")
	}
	if err := di.Stop(context.Background()); err != nil {
		t.Fatalf("stop err: %v", err)
	}
	if d2, ok2 := di.GetChaosDelay(); ok2 || d2 != 0 {
		t.Fatalf("expected no delay after stop, got %v %v", d2, ok2)
	}
}

func TestDelay_IntervalMode_AppliesDelay(t *testing.T) {
	di := RandomDelayWithInterval(3*time.Millisecond, 3*time.Millisecond, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := di.Inject(ctx); err != nil {
		t.Fatalf("inject err: %v", err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	var applied time.Duration
	for time.Now().Before(deadline) {
		if d, ok := di.GetChaosDelay(); ok {
			applied = d
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if applied == 0 {
		t.Fatalf("expected some delay to be applied in interval mode")
	}
	if applied != 3*time.Millisecond {
		t.Fatalf("expected 3ms delay, got %v", applied)
	}
	// Stop and ensure no further delays
	_ = di.Stop(context.Background())
	if d, ok := di.GetChaosDelay(); ok || d != 0 {
		t.Fatalf("expected no delay after stop, got %v %v", d, ok)
	}
}
