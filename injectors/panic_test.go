package injectors

import (
	"context"
	"testing"
	"time"

	"github.com/rom8726/chaoskit"
)

func TestPanicInjector_BeforeStepPanicsAtProbabilityOne(t *testing.T) {
	p := PanicProbability(1.0)

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic, got none")
		}
	}()

	// BeforeStep should panic almost surely when probability=1
	_ = p.BeforeStep(context.Background())
}

func TestPanicInjector_NameTypeMetrics(t *testing.T) {
	p := PanicProbability(0.3)
	if got := p.Name(); got == "" {
		t.Fatalf("expected non-empty name")
	}
	// Type should be hybrid
	if got := p.Type(); got != chaoskit.InjectorTypeHybrid {
		t.Fatalf("unexpected type: %v", got)
	}
	if prob := p.GetPanicProbability(); prob != 0.3 {
		t.Fatalf("expected prob=0.3, got %v", prob)
	}

	// Start and Stop should mark stopped and not error
	if err := p.Inject(context.Background()); err != nil {
		t.Fatalf("inject err: %v", err)
	}
	if err := p.Stop(context.Background()); err != nil {
		t.Fatalf("stop err: %v", err)
	}

	// Metrics should include keys and stopped true
	m := p.GetMetrics()
	if m["probability"].(float64) != 0.3 {
		t.Fatalf("metrics: unexpected probability: %v", m["probability"])
	}
	if stopped, ok := m["stopped"].(bool); !ok || !stopped {
		t.Fatalf("expected stopped=true in metrics, got %v", m["stopped"])
	}
}

func TestPanicInjector_ShouldChaosPanicRespectsStopped(t *testing.T) {
	p := PanicProbability(1.0)
	// Start and then stop
	_ = p.Inject(context.Background())
	_ = p.Stop(context.Background())
	// Give goroutine a tick to observe stop
	time.Sleep(10 * time.Millisecond)
	if p.ShouldChaosPanic() {
		t.Fatalf("should not panic when stopped")
	}
}
