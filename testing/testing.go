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
	thresholds     *chaoskit.SuccessThresholds
	skipVerdict    bool
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

// WithThresholds sets custom success thresholds for verdict calculation
func WithThresholds(thresholds *chaoskit.SuccessThresholds) ChaosTestOption {
	return func(c *chaosTestConfig) {
		c.thresholds = thresholds
	}
}

// WithDefaultThresholds uses default success thresholds (95% success rate)
func WithDefaultThresholds() ChaosTestOption {
	return func(c *chaosTestConfig) {
		c.thresholds = chaoskit.DefaultThresholds()
	}
}

// WithStrictThresholds uses strict success thresholds (100% success rate)
func WithStrictThresholds() ChaosTestOption {
	return func(c *chaosTestConfig) {
		c.thresholds = chaoskit.StrictThresholds()
	}
}

// WithRelaxedThresholds uses relaxed success thresholds (80% success rate)
func WithRelaxedThresholds() ChaosTestOption {
	return func(c *chaosTestConfig) {
		c.thresholds = chaoskit.RelaxedThresholds()
	}
}

// WithoutVerdict skips verdict calculation (only basic report)
func WithoutVerdict() ChaosTestOption {
	return func(c *chaosTestConfig) {
		c.skipVerdict = true
	}
}

// RunChaos creates a chaos test function that uses the full ChaosKit framework.
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
//	    chaoskit.RunChaos(t, "name", target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
//	        return s.
//	            Step("step1", func(ctx context.Context, target chaoskit.Target) error {
//	                // Your test logic
//	                return DoSomething()
//	            }).
//	            Inject("delay", injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond)).
//	            Assert("goroutines", validators.GoroutineLimit(100))
//	    },
//	        WithRepeat(10),
//	        WithDefaultThresholds(), // Enables verdict with 95% success rate
//	    )
//	}
func RunChaos(
	t TestingT,
	name string,
	target chaoskit.Target,
	builderFn func(*chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder,
	opts ...ChaosTestOption,
) {
	t.Helper()

	// Apply options
	config := &chaosTestConfig{
		repeat:        1,
		failurePolicy: chaoskit.FailFast,
		thresholds:    chaoskit.DefaultThresholds(), // Use default thresholds
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
		t.Errorf("chaos test execution failed: %v", err)

		// Print report on failure
		if !config.skipReport {
			printReport(t, executor, config)
		}

		t.FailNow()
		return
	}

	// Calculate verdict and print report
	if !config.skipReport || !config.skipVerdict {
		verdict := evaluateVerdict(t, executor, config)

		// Fail test if verdict is FAIL
		if verdict == chaoskit.VerdictFail {
			t.Errorf("chaos test verdict: FAIL")
			t.FailNow()
		}
	}
}

// printReport prints the test report
func printReport(t TestingT, executor *chaoskit.Executor, config *chaosTestConfig) {
	if config.skipVerdict {
		// Print simple report
		report := executor.Reporter().GenerateReport()
		if logger, ok := t.(interface{ Logf(string, ...interface{}) }); ok {
			logger.Logf("\n%s", report)
		}
	} else {
		// Print detailed report with verdict
		report, err := executor.Reporter().GetVerdict(config.thresholds)
		if err != nil {
			if logger, ok := t.(interface{ Logf(string, ...interface{}) }); ok {
				logger.Logf("\nFailed to generate verdict: %v", err)
				logger.Logf("\n%s", executor.Reporter().GenerateReport())
			}
			return
		}

		textReport := executor.Reporter().GenerateTextReport(report)
		if logger, ok := t.(interface{ Logf(string, ...interface{}) }); ok {
			logger.Logf("\n%s", textReport)
		}
	}
}

// evaluateVerdict evaluates the verdict and returns it
func evaluateVerdict(t TestingT, executor *chaoskit.Executor, config *chaosTestConfig) chaoskit.Verdict {
	if config.skipVerdict {
		return chaoskit.VerdictPass
	}

	// Get verdict
	report, err := executor.Reporter().GetVerdict(config.thresholds)
	if err != nil {
		t.Errorf("failed to generate verdict: %v", err)
		return chaoskit.VerdictFail
	}

	// Print report
	if !config.skipReport {
		textReport := executor.Reporter().GenerateTextReport(report)
		if logger, ok := t.(interface{ Logf(string, ...interface{}) }); ok {
			logger.Logf("\n%s", textReport)
		}
	}

	return report.Verdict
}

// RunChaosSimple is a simplified version that takes steps, injectors, and validators directly.
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
//	    chaoskit.RunChaosSimple(t, "name", target, steps, injectors, validators,
//	        WithRepeat(10),
//	        WithDefaultThresholds(),
//	    )
//	}
func RunChaosSimple(
	t TestingT,
	name string,
	target chaoskit.Target,
	steps []func(context.Context, chaoskit.Target) error,
	injectors []chaoskit.Injector,
	validators []chaoskit.Validator,
	opts ...ChaosTestOption,
) {
	RunChaos(t, name, target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
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
