package testing

import (
	"context"
	"fmt"

	"github.com/rom8726/chaoskit"
)

// TestingT is an interface that matches testing.T and similar types
type TestingT interface {
	Errorf(format string, args ...interface{})
	FailNow()
	Helper()
}

// ChaosTestOption configures chaos testing
type ChaosTestOption func(*chaosTestConfig)

type chaosTestConfig struct {
	repeat         int
	failurePolicy  chaoskit.FailurePolicy
	executorOpts   []chaoskit.ExecutorOption
	skipReport     bool
	reportToStderr bool
}

// WithRepeat sets the number of times to repeat the test scenario
func WithRepeat(n int) ChaosTestOption {
	return func(c *chaosTestConfig) {
		c.repeat = n
	}
}

// WithFailurePolicy sets how to handle failures (FailFast or ContinueOnFailure)
func WithFailurePolicy(policy chaoskit.FailurePolicy) ChaosTestOption {
	return func(c *chaosTestConfig) {
		c.failurePolicy = policy
	}
}

// WithExecutorOptions passes options to the underlying executor
func WithExecutorOptions(opts ...chaoskit.ExecutorOption) ChaosTestOption {
	return func(c *chaosTestConfig) {
		c.executorOpts = append(c.executorOpts, opts...)
	}
}

// WithoutReport skips printing the report after execution
func WithoutReport() ChaosTestOption {
	return func(c *chaosTestConfig) {
		c.skipReport = true
	}
}

// WithReportToStderr prints the report to stderr instead of stdout
func WithReportToStderr() ChaosTestOption {
	return func(c *chaosTestConfig) {
		c.reportToStderr = true
	}
}

// WithChaos creates a chaos test function that uses the full ChaosKit framework.
// It creates a scenario using ScenarioBuilder, runs it with an Executor, and validates results.
//
// The builderFn receives a pre-initialized ScenarioBuilder with the target already set.
// You should add steps, injectors, and validators to the builder.
//
// Usage:
//
//	func TestWithChaos(t *testing.T) {
//	    target := &MyTarget{}
//
//	    chaoskit.WithChaos(t, "name", target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
//	        return s.
//	            Step("step1", func(ctx context.Context, target chaoskit.Target) error {
//	                // Your test logic
//	                return DoSomething()
//	            }).
//	            Inject("delay", injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond)).
//	            Assert("goroutines", validators.GoroutineLimit(100))
//	    }, WithRepeat(10))()
//	}
func WithChaos(
	t TestingT,
	name string,
	target chaoskit.Target,
	builderFn func(*chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder,
	opts ...ChaosTestOption,
) func() {
	return func() {
		t.Helper()

		// Apply options
		config := &chaosTestConfig{
			repeat:        1,
			failurePolicy: chaoskit.FailFast,
		}
		for _, opt := range opts {
			opt(config)
		}

		// Create scenario builder
		builder := chaoskit.NewScenario(name).WithTarget(target)

		// Let user configure the scenario
		builder = builderFn(builder)

		// Set repeat count
		builder = builder.Repeat(config.repeat)

		// Build scenario
		scenario := builder.Build()

		// Create executor with options
		executorOpts := append(
			[]chaoskit.ExecutorOption{chaoskit.WithFailurePolicy(config.failurePolicy)},
			config.executorOpts...,
		)
		executor := chaoskit.NewExecutor(executorOpts...)

		// Run scenario
		ctx := context.Background()
		if err := executor.Run(ctx, scenario); err != nil {
			t.Errorf("chaos test failed: %v", err)

			// Print report on failure
			if !config.skipReport {
				report := executor.Reporter().GenerateReport()
				if logger, ok := t.(interface{ Logf(string, ...interface{}) }); ok {
					logger.Logf("\n%s", report)
				}
			}

			t.FailNow()
			return
		}

		// Print report on success (if not skipped)
		if !config.skipReport {
			report := executor.Reporter().GenerateReport()
			if logger, ok := t.(interface{ Logf(string, ...interface{}) }); ok {
				logger.Logf("\n%s", report)
			}
		}
	}
}

// WithChaosSimple is a simplified version that takes steps, injectors, and validators directly.
// This is useful when you don't need the full builder flexibility.
//
// Usage:
//
//	func TestSimpleChaos(t *testing.T) {
//	    target := &MyTarget{}
//	    steps := []chaoskit.StepFunc{
//	        func(ctx context.Context, target chaoskit.Target) error {
//	            return DoSomething()
//	        },
//	    }
//	    injectors := []chaoskit.Injector{
//	        injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond),
//	    }
//	    validators := []chaoskit.Validator{
//	        validators.GoroutineLimit(100),
//	    }
//
//	    chaoskit.WithChaosSimple(t, name, target, steps, injectors, validators, WithRepeat(10))()
//	}
func WithChaosSimple(
	t TestingT,
	name string,
	target chaoskit.Target,
	steps []func(context.Context, chaoskit.Target) error,
	injectors []chaoskit.Injector,
	validators []chaoskit.Validator,
	opts ...ChaosTestOption,
) func() {
	return WithChaos(t, name, target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
		// Add steps
		for i, stepFn := range steps {
			s = s.Step(fmt.Sprintf("step-%d", i+1), stepFn)
		}

		// Add injectors
		for i, inj := range injectors {
			s = s.Inject(fmt.Sprintf("injector-%d", i+1), inj)
		}

		// Add validators
		for i, val := range validators {
			s = s.Assert(fmt.Sprintf("validator-%d", i+1), val)
		}

		return s
	}, opts...)
}
