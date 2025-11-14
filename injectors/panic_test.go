package injectors

import (
	"context"
	"testing"

	"github.com/rom8726/chaoskit"
)

func TestPanicInjector_MaybePanicWorks(t *testing.T) {
	p := PanicProbability(1.0)
	_ = p.Inject(context.Background())

	// ShouldChaosPanic should return true when probability=1
	if !p.ShouldChaosPanic() {
		t.Fatalf("expected ShouldChaosPanic to return true when probability=1")
	}
}

func TestPanicInjector_NameTypeMetrics(t *testing.T) {
	p := PanicProbability(0.3)
	if got := p.Name(); got == "" {
		t.Fatalf("expected non-empty name")
	}
	// Type should be context-based
	if got := p.Type(); got != chaoskit.InjectorTypeContext {
		t.Fatalf("unexpected type: %v, want InjectorTypeContext", got)
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
	if p.ShouldChaosPanic() {
		t.Fatalf("should not panic when stopped")
	}
}
