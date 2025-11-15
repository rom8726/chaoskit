package injectors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"

	"github.com/rom8726/chaoskit"
)

// ErrorInjector возвращает ошибку через context value
type ErrorInjector struct {
	name        string
	probability float64
	errorMsg    string
	errorCount  int64

	mu      sync.Mutex
	stopped bool

	rng *rand.Rand // Deterministic random generator from context
}

func ErrorWithProbability(errorMsg string, probability float64) *ErrorInjector {
	return &ErrorInjector{
		name:        "error_injector",
		probability: probability,
		errorMsg:    errorMsg,
	}
}

func (e *ErrorInjector) Name() string {
	return e.name
}

func (e *ErrorInjector) Inject(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stopped {
		return fmt.Errorf("injector already stopped")
	}

	// Store deterministic random generator from context
	e.rng = chaoskit.GetRand(ctx)

	return nil
}

func (e *ErrorInjector) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.stopped {
		e.stopped = true
		chaoskit.GetLogger(ctx).Info("delay injector stopped",
			slog.String("injector", e.name))
	}

	return nil
}

// BeforeStep injects a delay before step execution
func (e *ErrorInjector) BeforeStep(context.Context) error {
	return nil
}

// AfterStep is called after step execution (no-op for delay injector)
func (e *ErrorInjector) AfterStep(ctx context.Context, err error) error {
	if err != nil {
		e.mu.Lock()
		e.errorCount++
		e.mu.Unlock()
	}

	return nil
}

// Type implements CategorizedInjector
func (e *ErrorInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeContext
}

// GetMetrics implements MetricsProvider
func (e *ErrorInjector) GetMetrics() map[string]interface{} {
	e.mu.Lock()
	defer e.mu.Unlock()

	return map[string]interface{}{
		"probability": e.probability,
		"error_count": e.errorCount,
		"stopped":     e.stopped,
	}
}

func (e *ErrorInjector) ShouldReturnError() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.stopped {
		return nil
	}

	if e.rng.Float64() < e.probability {
		return errors.New(e.errorMsg)
	}

	return nil
}
