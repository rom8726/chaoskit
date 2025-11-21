package main

import (
	"context"
	"testing"
	"time"

	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/injectors"
	chaostest "github.com/rom8726/chaoskit/testing"
	"github.com/rom8726/chaoskit/validators"
)

// TestVerdictPass demonstrates a test that passes all thresholds
func TestVerdictPass(t *testing.T) {
	target := &TestTarget{}

	chaostest.RunChaos(t, "verdict-pass", target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
		return s.
			Step("work", func(ctx context.Context, target chaoskit.Target) error {
				testTarget := target.(*TestTarget)
				testTarget.Increment()
				return nil
			}).
			Inject("delay", injectors.RandomDelay(1*time.Millisecond, 5*time.Millisecond)).
			Assert("goroutines", validators.GoroutineLimit(50))
	},
		chaostest.WithRepeat(10),
		chaostest.WithDefaultThresholds(), // 95% success rate required
	)()
}

// TestVerdictUnstable demonstrates a test with warnings (commented out to not fail CI)
// Uncomment to see UNSTABLE verdict in action
/*
func TestVerdictUnstable(t *testing.T) {
	target := &TestTarget{}

	// Custom thresholds with warning validators
	thresholds := &chaoskit.SuccessThresholds{
		MinSuccessRate: 0.90,
		CriticalValidators: []string{
			chaoskit.ValidatorGoroutineLimit,
		},
		WarningValidators: []string{
			chaoskit.ValidatorPanicRecovery,
		},
	}

	chaostest.RunChaos(t, "verdict-unstable", target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
		return s.
			Step("work", func(ctx context.Context, target chaoskit.Target) error {
				testTarget := target.(*TestTarget)
				testTarget.Increment()
				chaoskit.MaybePanic(ctx) // May trigger panic
				return nil
			}).
			Inject("panic", injectors.PanicProbability(0.1)). // 10% panic probability
			Assert("goroutines", validators.GoroutineLimit(50)).
			Assert("panics", validators.NoPanics(3)) // Allow up to 3 panics
	},
		chaostest.WithRepeat(20),
		chaostest.WithThresholds(thresholds),
		chaostest.WithFailurePolicy(chaoskit.ContinueOnFailure),
	)()
}
*/

// TestVerdictFail demonstrates a test that fails thresholds (commented out to not fail CI)
// Uncomment to see FAIL verdict in action
/*
func TestVerdictFail(t *testing.T) {
	target := &TestTarget{}

	chaostest.RunChaos(t, "verdict-fail", target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
		failCount := 0
		return s.
			Step("work", func(ctx context.Context, target chaoskit.Target) error {
				testTarget := target.(*TestTarget)
				testTarget.Increment()

				// Intentionally fail some iterations
				failCount++
				if failCount%2 == 0 {
					return fmt.Errorf("intentional failure")
				}
				return nil
			}).
			Assert("goroutines", validators.GoroutineLimit(50))
	},
		chaostest.WithRepeat(10),
		chaostest.WithStrictThresholds(), // 100% success rate required - will fail
	)()
}
*/

// TestVerdictWithCustomThresholds demonstrates custom threshold configuration
func TestVerdictWithCustomThresholds(t *testing.T) {
	target := &TestTarget{}

	// Custom thresholds allowing some failures
	thresholds := &chaoskit.SuccessThresholds{
		MinSuccessRate:      0.85, // 85% success rate
		MaxFailedIterations: 3,    // Allow max 3 failures
		CriticalValidators: []string{
			chaoskit.ValidatorGoroutineLimit,
			chaoskit.ValidatorInfiniteLoop,
		},
	}

	chaostest.RunChaos(t, "custom-thresholds", target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
		return s.
			Step("work", func(ctx context.Context, target chaoskit.Target) error {
				testTarget := target.(*TestTarget)
				testTarget.Increment()
				return nil
			}).
			Inject("delay", injectors.RandomDelay(1*time.Millisecond, 10*time.Millisecond)).
			Assert("goroutines", validators.GoroutineLimit(100))
	},
		chaostest.WithRepeat(20),
		chaostest.WithThresholds(thresholds),
		chaostest.WithFailurePolicy(chaoskit.ContinueOnFailure),
	)()
}

// TestVerdictWithoutReport demonstrates running without report output
func TestVerdictWithoutReport(t *testing.T) {
	target := &TestTarget{}

	chaostest.RunChaos(t, "no-report", target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
		return s.
			Step("work", func(ctx context.Context, target chaoskit.Target) error {
				testTarget := target.(*TestTarget)
				testTarget.Increment()
				return nil
			}).
			Inject("delay", injectors.RandomDelay(1*time.Millisecond, 5*time.Millisecond))
	},
		chaostest.WithRepeat(5),
		chaostest.WithoutReport(), // No report printed
	)()
}

// TestVerdictWithoutVerdictCalculation demonstrates running without verdict
func TestVerdictWithoutVerdictCalculation(t *testing.T) {
	target := &TestTarget{}

	chaostest.RunChaos(t, "no-verdict", target, func(s *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
		return s.
			Step("work", func(ctx context.Context, target chaoskit.Target) error {
				testTarget := target.(*TestTarget)
				testTarget.Increment()
				return nil
			}).
			Inject("delay", injectors.RandomDelay(1*time.Millisecond, 5*time.Millisecond))
	},
		chaostest.WithRepeat(5),
		chaostest.WithoutVerdict(), // No verdict calculation
	)()
}
