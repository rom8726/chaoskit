package injectors

import (
	"context"
	"testing"
	"time"
)

func TestContextCancellation_ProbabilityClamp(t *testing.T) {
	c1 := NewContextCancellationInjector(-1)
	if got := c1.GetCancellationProbability(); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
	c2 := NewContextCancellationInjector(2)
	if got := c2.GetCancellationProbability(); got != 1 {
		t.Fatalf("expected 1, got %v", got)
	}
}

func TestContextCancellation_CancelsChildAtProbOne(t *testing.T) {
	inj := NewContextCancellationInjector(1.0)
	if err := inj.Inject(context.Background()); err != nil {
		t.Fatalf("inject err: %v", err)
	}

	parent := context.Background()
	ctx, cancel := inj.GetChaosContext(parent)
	defer cancel()

	// Wait for async cancellation (~10ms in implementation)
	select {
	case <-ctx.Done():
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected context to be cancelled by injector")
	}

	if inj.GetCancelCount() == 0 {
		t.Fatalf("expected cancel count > 0")
	}

	_ = inj.Stop(context.Background())
}

func TestContextCancellation_StopCancelsActivesAndMetrics(t *testing.T) {
	inj := NewContextCancellationInjector(0.0)
	_ = inj.Inject(context.Background())

	parent := context.Background()
	ctx, cancel := inj.GetChaosContext(parent)

	// Ensure not cancelled yet
	select {
	case <-ctx.Done():
		t.Fatalf("should not be cancelled yet")
	default:
	}

	// Stop should cancel active contexts
	_ = inj.Stop(context.Background())

	select {
	case <-ctx.Done():
		// ok
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("expected context to be cancelled on Stop")
	}

	// Metrics should reflect stopped
	m := inj.GetMetrics()
	if stopped, ok := m["stopped"].(bool); !ok || !stopped {
		t.Fatalf("expected stopped=true in metrics")
	}

	cancel() // no-op if already cancelled
}
