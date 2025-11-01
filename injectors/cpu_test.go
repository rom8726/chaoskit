package injectors

import (
	"context"
	"testing"
	"time"
)

func TestCPUStress_StartStopAndMetrics(t *testing.T) {
	cpu := CPUStress(1)
	if cpu.Name() == "" {
		t.Fatalf("expected non-empty name")
	}
	if cpu.Type() != 0 { // InjectorTypeGlobal = 0
		t.Fatalf("expected global type")
	}
	if !cpu.IsGlobal() {
		t.Fatalf("expected IsGlobal true")
	}

	if err := cpu.Inject(context.Background()); err != nil {
		t.Fatalf("inject err: %v", err)
	}
	// Let worker spin a tiny bit
	time.Sleep(10 * time.Millisecond)
	if err := cpu.Stop(context.Background()); err != nil {
		t.Fatalf("stop err: %v", err)
	}
	m := cpu.GetMetrics()
	if stopped, ok := m["stopped"].(bool); !ok || !stopped {
		t.Fatalf("expected stopped=true in metrics")
	}
}
