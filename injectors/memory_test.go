package injectors

import (
	"context"
	"testing"
)

func TestMemoryPressure_AllocStopMetrics(t *testing.T) {
	m := MemoryPressure(1) // 1MB
	if m.Name() == "" {
		t.Fatalf("expected non-empty name")
	}
	if m.Type() != 0 || !m.IsGlobal() {
		t.Fatalf("expected global injector")
	}

	if err := m.Inject(context.Background()); err != nil {
		t.Fatalf("inject err: %v", err)
	}
	// expect allocated not nil before stop
	metrics := m.GetMetrics()
	if released, ok := metrics["released"].(bool); !ok || released {
		t.Fatalf("expected released=false before stop")
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("stop err: %v", err)
	}
	metrics = m.GetMetrics()
	if released, ok := metrics["released"].(bool); !ok || !released {
		t.Fatalf("expected released=true after stop")
	}
}
