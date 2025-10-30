package injectors

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
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
