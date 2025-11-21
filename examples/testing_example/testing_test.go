package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/injectors"
	chaostest "github.com/rom8726/chaoskit/testing"
	"github.com/rom8726/chaoskit/validators"

	"github.com/stretchr/testify/require"
)

// TestTarget is a simple target implementation for testing
type TestTarget struct {
	mu          sync.Mutex
	counter     int
	initialized bool
}

func (t *TestTarget) Name() string {
	return "test-target"
}

func (t *TestTarget) Setup(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.initialized = true
	t.counter = 0
	return nil
}

func (t *TestTarget) Teardown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.initialized = false
	return nil
}

func (t *TestTarget) Increment() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counter++
}

func (t *TestTarget) GetCounter() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.counter
}

// TestWithChaosFullAPI demonstrates the new WithChaos API with full framework integration
func TestWithChaosFullAPI(t *testing.T) {
	target := &TestTarget{}

	// Use the new WithChaos API with ScenarioBuilder
	chaostest.WithChaos(t, "full", target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
		return s.
			Step("increment", func(ctx context.Context, target chaoskit.Target) error {
				// Use chaos context functions
				chaoskit.MaybeDelay(ctx)

				// Execute business logic
				testTarget := target.(*TestTarget)
				testTarget.Increment()

				return nil
			}).
			Step("verify", func(ctx context.Context, target chaoskit.Target) error {
				testTarget := target.(*TestTarget)
				counter := testTarget.GetCounter()

				if counter <= 0 {
					return fmt.Errorf("counter should be positive, got %d", counter)
				}

				return nil
			}).
			Inject("delay", injectors.RandomDelay(5*time.Millisecond, 20*time.Millisecond)).
			Assert("goroutines", validators.GoroutineLimit(50))
	}, chaostest.WithRepeat(5))()
}

// TestWithChaosSimpleAPI demonstrates the simplified API
func TestWithChaosSimpleAPI(t *testing.T) {
	target := &TestTarget{}

	steps := []func(context.Context, chaoskit.Target) error{
		func(ctx context.Context, target chaoskit.Target) error {
			testTarget := target.(*TestTarget)
			testTarget.Increment()
			return nil
		},
		func(ctx context.Context, target chaoskit.Target) error {
			testTarget := target.(*TestTarget)
			require.Greater(t, testTarget.GetCounter(), 0)
			return nil
		},
	}

	injs := []chaoskit.Injector{
		injectors.RandomDelayWithProbability(5*time.Millisecond, 15*time.Millisecond, 0.5),
	}

	vals := []chaoskit.Validator{
		validators.GoroutineLimit(50),
	}

	chaostest.WithChaosSimple(t, "simple", target, steps, injs, vals, chaostest.WithRepeat(10))()
}

// TestWithChaosComplexScenario demonstrates a more complex scenario
func TestWithChaosComplexScenario(t *testing.T) {
	target := &TestTarget{}

	chaostest.WithChaos(t, "complex", target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
		return s.
			Step("init", func(ctx context.Context, target chaoskit.Target) error {
				testTarget := target.(*TestTarget)
				testTarget.Increment()
				return nil
			}).
			Step("process", func(ctx context.Context, target chaoskit.Target) error {
				// Use chaos functions
				chaoskit.MaybeDelay(ctx)

				testTarget := target.(*TestTarget)
				for i := 0; i < 5; i++ {
					testTarget.Increment()
					time.Sleep(1 * time.Millisecond)
				}
				return nil
			}).
			Step("validate", func(ctx context.Context, target chaoskit.Target) error {
				testTarget := target.(*TestTarget)
				counter := testTarget.GetCounter()

				// Should have at least 6 increments (1 + 5)
				if counter < 6 {
					return fmt.Errorf("expected at least 6 increments, got %d", counter)
				}
				return nil
			}).
			Inject("delay", injectors.RandomDelay(1*time.Millisecond, 10*time.Millisecond)).
			Inject("panic", injectors.PanicProbability(0.05)).
			Assert("goroutines", validators.GoroutineLimit(100)).
			Assert("panic-recovery", validators.NoPanics(5))
	},
		chaostest.WithRepeat(20),
		chaostest.WithFailurePolicy(chaoskit.ContinueOnFailure),
	)()
}
