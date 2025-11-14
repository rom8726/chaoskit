package injectors

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/rom8726/chaoskit"
)

// PanicInjector injects panics with a given probability
type PanicInjector struct {
	name        string
	probability float64
	mu          sync.Mutex
	stopped     bool
	rng         *rand.Rand // Deterministic random generator from context
}

// PanicProbability creates a new panic injector
func PanicProbability(probability float64) *PanicInjector {
	return &PanicInjector{
		name:        fmt.Sprintf("panic_injector_%.2f", probability),
		probability: probability,
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

	// Store deterministic random generator from context
	p.rng = chaoskit.GetRand(ctx)

	return nil
}

func (p *PanicInjector) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopped = true

	return nil
}

// ShouldChaosPanic returns true if panic should be triggered based on probability
func (p *PanicInjector) ShouldChaosPanic() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return false
	}

	// Use stored generator (should be set during Inject)
	rng := p.rng
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	return rng.Float64() < p.probability
}

// GetPanicProbability returns the configured panic probability
func (p *PanicInjector) GetPanicProbability() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.probability
}

// Type implements CategorizedInjector
func (p *PanicInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeContext // Works via MaybePanic() in user code
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
