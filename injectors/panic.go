package injectors

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

// PanicInjector injects panics with a given probability
type PanicInjector struct {
	name        string
	probability float64
	mu          sync.Mutex
	stopCh      chan struct{}
	stopped     bool
}

// PanicProbability creates a new panic injector
func PanicProbability(probability float64) *PanicInjector {
	return &PanicInjector{
		name:        fmt.Sprintf("panic_injector_%.2f", probability),
		probability: probability,
		stopCh:      make(chan struct{}),
	}
}

func (p *PanicInjector) Name() string {
	return p.name
}

func (p *PanicInjector) Inject(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return fmt.Errorf("injector already stopped")
	}

	// Start background panic injection
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-p.stopCh:
				return
			case <-ticker.C:
				if rand.Float64() < p.probability {
					// TODO: use gofail
					fmt.Printf("[CHAOS] Panic injected (probability: %.2f)\n", p.probability)
				}
			}
		}
	}()

	return nil
}

func (p *PanicInjector) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.stopped {
		close(p.stopCh)
		p.stopped = true
	}

	return nil
}

// BeforeStep injects a panic before step execution based on probability
func (p *PanicInjector) BeforeStep(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return nil
	}

	if rand.Float64() < p.probability {
		fmt.Printf("[CHAOS] Injecting panic (probability: %.2f)\n", p.probability)
		panic("chaos: injected panic")
	}

	return nil
}

// AfterStep is called after step execution (no-op for panic injector)
func (p *PanicInjector) AfterStep(ctx context.Context, err error) error {
	return nil
}

// ShouldChaosPanic returns true if panic should be triggered based on probability
func (p *PanicInjector) ShouldChaosPanic() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return false
	}

	return rand.Float64() < p.probability
}

// GetPanicProbability returns the configured panic probability
func (p *PanicInjector) GetPanicProbability() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.probability
}

// Type implements CategorizedInjector
func (p *PanicInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid // Works both as StepInjector and ChaosPanicProvider
}

// GetMetrics implements MetricsProvider
func (p *PanicInjector) GetMetrics() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	return map[string]interface{}{
		"probability": p.probability,
		"stopped":     p.stopped,
	}
}
