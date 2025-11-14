package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/injectors"
	"github.com/rom8726/chaoskit/validators"
)

// Example functions for monkey patching
var (
	dbQueryFunc = func(ctx context.Context) error {
		fmt.Println("[DB] Querying database...")
		time.Sleep(50 * time.Millisecond)
		fmt.Println("[DB] Query completed")

		return nil
	}

	calculatePriceFunc = func(amount float64) float64 {
		fmt.Printf("[CALC] Calculating price for amount: %.2f\n", amount)
		price := amount * 1.2 // 20% markup
		fmt.Printf("[CALC] Price calculated: %.2f\n", price)

		return price
	}

	processPaymentFunc = func() error {
		fmt.Println("[PAYMENT] Processing payment...")
		time.Sleep(30 * time.Millisecond)
		fmt.Println("[PAYMENT] Payment processed")

		return nil
	}
)

// ExampleTarget demonstrates a system under test
type ExampleTarget struct {
	name string
}

func (t *ExampleTarget) Name() string {
	return t.name
}

func (t *ExampleTarget) Setup(ctx context.Context) error {
	fmt.Println("[TARGET] Setup")

	return nil
}

func (t *ExampleTarget) Teardown(ctx context.Context) error {
	fmt.Println("[TARGET] Teardown")

	return nil
}

// RunStep executes a step that uses various chaos-injectable functions
func RunStep(ctx context.Context, target chaoskit.Target) error {
	fmt.Println("[STEP] Starting execution...")

	// Use context cancellation chaos
	ctx, cancel := chaoskit.MaybeCancelContext(ctx)
	defer cancel()

	// Example 1: Database query with timeout injection
	fmt.Println("\n--- Database Query with Timeout ---")
	if err := dbQueryFunc(ctx); err != nil {
		fmt.Printf("[STEP] DB query failed: %v\n", err)

		return err
	}

	// Example 2: Calculate price with value corruption
	fmt.Println("\n--- Price Calculation with Value Corruption ---")
	amount := 100.0
	price := calculatePriceFunc(amount)
	fmt.Printf("[STEP] Final price: %.2f\n", price)

	// Example 3: Process payment with error injection
	fmt.Println("\n--- Payment Processing with Error Injection ---")
	if err := processPaymentFunc(); err != nil {
		fmt.Printf("[STEP] Payment failed: %v\n", err)

		return err
	}

	fmt.Println("[STEP] Execution completed successfully")

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=== ChaosKit New Injectors Example ===")
	log.Println("Demonstrates Timeout, Error, Value Corruption, and Context Cancellation injectors")
	log.Println()

	target := &ExampleTarget{name: "example-target"}

	// 1. Timeout Injector - interrupts function execution after timeout
	timeoutInjector := injectors.MonkeyPatchTimeout([]injectors.TimeoutPatchTarget{
		{
			Func:        &dbQueryFunc,
			Timeout:     30 * time.Millisecond, // Timeout shorter than function execution
			Probability: 0.5,                   // 50% chance of timeout
			FuncName:    "dbQueryFunc",
			ReturnError: errors.New("chaos: database query timeout"),
		},
	})

	// 2. Error Injector - replaces returned error with simulated error
	errorInjector := injectors.MonkeyPatchError([]injectors.ErrorPatchTarget{
		{
			Func:        &processPaymentFunc,
			Error:       errors.New("chaos: payment processing failed"),
			Probability: 0.3, // 30% chance of error
			FuncName:    "processPaymentFunc",
		},
	})

	// 3. Value Corruption Injector - modifies return values
	valueCorruptionInjector := injectors.MonkeyPatchValueCorruption([]injectors.ValueCorruptionPatchTarget{
		{
			Func: &calculatePriceFunc,
			CorruptFunc: func(orig float64) float64 {
				// Corrupt: multiply by 10 (wrong price)
				return orig * 10
			},
			Probability: 0.2, // 20% chance of corruption
			FuncName:    "calculatePriceFunc",
		},
	})

	// 4. Context Cancellation Injector - cancels contexts randomly
	contextCancellationInjector := injectors.NewContextCancellationInjector(0.4) // 40% chance

	// Build scenario
	scenario := chaoskit.NewScenario("new-injectors-demo").
		WithTarget(target).
		Step("run-step", RunStep).
		Inject("timeout", timeoutInjector).
		Inject("error", errorInjector).
		Inject("value-corruption", valueCorruptionInjector).
		Inject("context-cancellation", contextCancellationInjector).
		Assert("panic-recovery", validators.NoPanics(0)).
		Repeat(10).
		Build()

	// Run scenario with executor
	ctx := context.Background()
	executor := chaoskit.NewExecutor(
		chaoskit.WithFailurePolicy(chaoskit.ContinueOnFailure),
	)

	if err := executor.Run(ctx, scenario); err != nil {
		log.Printf("Scenario execution completed with errors: %v", err)
	}

	// Get verdict and generate report
	thresholds := chaoskit.DefaultThresholds()
	report, err := executor.Reporter().GetVerdict(thresholds)
	if err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	// Print detailed report
	log.Println("\n=== Chaos Test Report ===")
	log.Println(executor.Reporter().GenerateTextReport(report))

	log.Println("\nKey Points:")
	log.Println("1. TimeoutInjector - interrupts function execution after timeout")
	log.Println("2. ErrorInjector - replaces returned errors with simulated errors")
	log.Println("3. ValueCorruptionInjector - modifies return values")
	log.Println("4. ContextCancellationInjector - cancels contexts randomly")
	log.Println("\nNote: Monkey patching requires -gcflags=all=-l for correct operation")
	log.Println("Run: go run -gcflags=all=-l main.go")

	// Exit with verdict code
	os.Exit(report.Verdict.ExitCode())
}
