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

// Example target functions to patch
var (
	criticalFunction = func() {
		fmt.Println("[Target] Executing critical function")
	}

	databaseQuery = func(query string) (result string, err error) {
		fmt.Printf("[Target] Executing database query: %s\n", query)

		return "result", nil
	}

	networkCall = func(host string, port int) error {
		fmt.Printf("[Target] Making network call to %s:%d\n", host, port)

		return nil
	}
)

// ExampleTarget demonstrates monkey patching injection
type ExampleTarget struct {
	callCount int
}

func NewExampleTarget() *ExampleTarget {
	return &ExampleTarget{}
}

func (t *ExampleTarget) Name() string {
	return "monkey-patch-example"
}

func (t *ExampleTarget) Setup(ctx context.Context) error {
	log.Println("[Target] Setting up...")

	return nil
}

func (t *ExampleTarget) Teardown(ctx context.Context) error {
	log.Printf("[Target] Tearing down (total calls: %d)", t.callCount)

	return nil
}

// Execute runs the target logic with patched functions
func (t *ExampleTarget) Execute(ctx context.Context) error {
	t.callCount++

	fmt.Println("\n=== Executing target functions ===")

	// These calls will trigger monkey patched versions
	// With probability, they will panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[Target] Recovered from panic: %v\n", r)
			}
		}()
		criticalFunction()
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[Target] Recovered from panic: %v\n", r)
			}
		}()
		_, _ = databaseQuery("SELECT * FROM users")
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[Target] Recovered from panic: %v\n", r)
			}
		}()
		_ = networkCall("api.example.com", 443)
	}()

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=== ChaosKit Monkey Patch Example ===")
	log.Println("WARNING: Monkey patching requires -gcflags=all=-l")
	log.Println("Run with: go run -gcflags=all=-l main.go")
	log.Println()

	// Create target
	target := NewExampleTarget()

	// Create monkey patch panic injector
	monkeyPanicInjector := injectors.MonkeyPatchPanic([]injectors.PatchTarget{
		{
			Func:         &criticalFunction,
			Probability:  0.3, // 30% chance of panic
			FuncName:     "criticalFunction",
			PanicMessage: "chaos: critical function panic",
		},
		{
			Func:         &databaseQuery,
			Probability:  0.2, // 20% chance of panic
			FuncName:     "databaseQuery",
			PanicMessage: "chaos: database query panic",
		},
		{
			Func:         &networkCall,
			Probability:  0.15, // 15% chance of panic
			FuncName:     "networkCall",
			PanicMessage: "chaos: network call panic",
		},
	})

	// Build scenario
	scenario := chaoskit.NewScenario("monkey-patch-demo").
		WithTarget(target).
		Step("execute", func(ctx context.Context, t chaoskit.Target) error {
			if ex, ok := t.(*ExampleTarget); ok {
				return ex.Execute(ctx)
			}

			return fmt.Errorf("invalid target type")
		}).
		Inject("monkey-patch-panic", monkeyPanicInjector).
		Inject("monkey-patch-delay", injectors.MonkeyPatchDelay([]injectors.DelayPatchTarget{
			{
				Func:        &databaseQuery,
				Probability: 0.5, // 50% chance of delay
				MinDelay:    20 * time.Millisecond,
				MaxDelay:    100 * time.Millisecond,
				DelayBefore: true,
				FuncName:    "databaseQuery",
			},
			{
				Func:        &networkCall,
				Probability: 0.4, // 40% chance of delay
				MinDelay:    10 * time.Millisecond,
				MaxDelay:    50 * time.Millisecond,
				DelayBefore: false, // delay after the call
				FuncName:    "networkCall",
			},
		})).
		Assert("panic_recovery", validators.NoPanics(3)).
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
	log.Println("1. Monkey patching intercepts function calls at runtime")
	log.Println("2. Panic injector: Functions are replaced with versions that may panic")
	log.Println("3. Delay injector: Functions are replaced with versions that may add delays")
	log.Println("4. Original functions are restored when injector stops")
	log.Println("5. Requires -gcflags=all=-l to disable compiler inlining")
	log.Println("6. Delay can be applied before or after function execution")

	// Exit with verdict code
	os.Exit(report.Verdict.ExitCode())
}
