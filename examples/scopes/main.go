package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/injectors"
	"github.com/rom8726/chaoskit/validators"
)

// Simple target implementation
type simpleTarget struct {
	name string
}

func (t *simpleTarget) Setup(ctx context.Context) error {
	fmt.Printf("[Target] %s setup complete\n", t.name)

	return nil
}

func (t *simpleTarget) Teardown(ctx context.Context) error {
	fmt.Printf("[Target] %s teardown complete\n", t.name)

	return nil
}

func (t *simpleTarget) Name() string {
	return t.name
}

func main() {
	ctx := context.Background()

	// Create injectors for different scopes

	// Scenario with scopes for organizing injectors by system component
	scenario := chaoskit.NewScenario("payments").
		WithTarget(&simpleTarget{name: "payment-system"}).
		// Scope for database injectors
		Scope("db", func(s *chaoskit.ScopeBuilder) {
			s.Inject("delay", injectors.RandomDelay(50*time.Millisecond, 200*time.Millisecond)).
				Inject("error", injectors.PanicProbability(0.1))
		}).
		// Scope for API injectors
		Scope("api", func(s *chaoskit.ScopeBuilder) {
			s.Inject("panic", injectors.PanicProbability(0.02)).
				Inject("delay", injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond))
		}).
		// Scope for cache injectors
		Scope("cache", func(s *chaoskit.ScopeBuilder) {
			s.Inject("delay", injectors.RandomDelay(5*time.Millisecond, 20*time.Millisecond))
		}).
		// Direct injectors (not in a scope) - still supported for backward compatibility
		Inject("global-cpu", injectors.CPUStress(50)).
		// Step to execute
		Step("run", func(ctx context.Context, target chaoskit.Target) error {
			fmt.Println("[Step] Executing workflow...")

			// Simulate some work
			for i := 0; i < 3; i++ {
				// Potential chaos points
				chaoskit.MaybeDelay(ctx)
				chaoskit.MaybePanic(ctx)

				fmt.Printf("[Step] Iteration %d completed\n", i+1)
				time.Sleep(100 * time.Millisecond)
			}

			return nil
		}).
		// Validators
		Assert("no-panics", validators.NoPanics(0)).
		Build()

	// Create executor and run scenario
	executor := chaoskit.NewExecutor(
		chaoskit.WithLogger(log.New(log.Writer(), "[CHAOS] ", log.LstdFlags)),
		chaoskit.WithFailurePolicy(chaoskit.ContinueOnFailure),
	)

	fmt.Println("\n=== Running scenario with scopes ===")
	if err := executor.Run(ctx, scenario); err != nil {
		log.Printf("Scenario execution completed with errors: %v", err)
	}

	// Example 2: Mixed usage (scopes + direct injectors)
	fmt.Println("\n=== Example 2: Mixed usage ===")
	scenario2 := chaoskit.NewScenario("mixed-example").
		WithTarget(&simpleTarget{name: "mixed-system"}).
		// Some injectors in scopes
		Scope("db", func(s *chaoskit.ScopeBuilder) {
			s.Inject("db-delay", injectors.RandomDelay(100*time.Millisecond, 200*time.Millisecond))
		}).
		// Some injectors directly (backward compatibility)
		Inject("global-panic", injectors.PanicProbability(0.05)).
		Step("run", func(ctx context.Context, target chaoskit.Target) error {
			fmt.Println("[Step] Running with mixed injectors...")
			chaoskit.MaybeDelay(ctx)
			chaoskit.MaybePanic(ctx)

			return nil
		}).
		Build()

	if err := executor.Run(ctx, scenario2); err != nil {
		log.Printf("Mixed scenario completed with errors: %v", err)
	}

	// Get verdict and generate report for all scenarios
	thresholds := chaoskit.DefaultThresholds()
	report, err := executor.Reporter().GetVerdict(thresholds)
	if err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	// Print metrics
	metrics := executor.Metrics().Stats()
	fmt.Printf("\n=== Metrics ===\n%+v\n", metrics)

	// Print detailed report
	fmt.Println("\n=== Chaos Test Report ===")
	fmt.Println(executor.Reporter().GenerateTextReport(report))

	// Exit with verdict code
	os.Exit(report.Verdict.ExitCode())
}
