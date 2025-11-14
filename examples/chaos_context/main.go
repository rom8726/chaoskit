package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync/atomic"
	"time"

	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/injectors"
	"github.com/rom8726/chaoskit/validators"
)

// WorkflowEngineWithChaos demonstrates how to use chaos context inside user code
type WorkflowEngineWithChaos struct {
	executionCount atomic.Int64
	rollbackCount  atomic.Int64
	panicCount     atomic.Int64
	maxDepth       int
	currentDepth   atomic.Int32
}

func NewWorkflowEngineWithChaos() *WorkflowEngineWithChaos {
	return &WorkflowEngineWithChaos{
		maxDepth: 50,
	}
}

func (w *WorkflowEngineWithChaos) Name() string {
	return "workflow-engine-with-chaos"
}

func (w *WorkflowEngineWithChaos) Setup(ctx context.Context) error {
	log.Println("[Engine] Setting up workflow engine with chaos context support...")

	return nil
}

func (w *WorkflowEngineWithChaos) Teardown(ctx context.Context) error {
	log.Println("[Engine] Tearing down workflow engine...")
	stats := w.GetStats()
	log.Printf("[Engine] Final stats: %+v\n", stats)

	return nil
}

func (w *WorkflowEngineWithChaos) Execute(ctx context.Context) error {
	w.executionCount.Add(1)

	// Simulate workflow steps
	steps := []string{"validate", "process", "commit"}

	for i, step := range steps {
		select {
		case <-ctx.Done():
			w.rollback(ctx, i)

			return ctx.Err()
		default:
			if err := w.executeStep(ctx, step, i); err != nil {
				w.rollback(ctx, i)

				return err
			}
		}
	}

	return nil
}

func (w *WorkflowEngineWithChaos) executeStep(ctx context.Context, step string, depth int) error {
	// Track recursion depth
	current := w.currentDepth.Add(1)
	defer w.currentDepth.Add(-1)

	// Record recursion depth for validators
	chaoskit.RecordRecursionDepth(ctx, int(current))

	if current > int32(w.maxDepth) {
		return fmt.Errorf("max recursion depth exceeded: %d", current)
	}

	// CRITICAL: Call MaybePanic at critical points in your code
	// This allows ChaosKit to inject panics INSIDE your logic
	chaoskit.MaybePanic(ctx)

	// Simulate work with potential chaos delay
	// Option 1: Use MaybeDelay to inject delay from chaos context
	chaoskit.MaybeDelay(ctx)

	// Option 2: Use ShouldFail to simulate failures
	if chaoskit.ShouldFail(ctx, 0.1) {
		return fmt.Errorf("step %s failed (chaos-induced)", step)
	}

	// Normal work simulation
	time.Sleep(time.Millisecond * time.Duration(5+rand.Intn(10)))

	// Another critical point where panic might occur
	chaoskit.MaybePanic(ctx)

	return nil
}

func (w *WorkflowEngineWithChaos) rollback(ctx context.Context, fromStep int) {
	w.rollbackCount.Add(1)

	// Simulate rollback with potential recursion
	for i := fromStep; i >= 0; i-- {
		current := w.currentDepth.Add(1)
		defer w.currentDepth.Add(-1)

		// Record recursion depth
		chaoskit.RecordRecursionDepth(ctx, int(current))

		if current > int32(w.maxDepth) {
			// Prevent infinite rollback recursion
			return
		}

		// Chaos can happen during rollback too!
		chaoskit.MaybePanic(ctx)
		chaoskit.MaybeDelay(ctx)

		time.Sleep(time.Millisecond * 2)
	}
}

func (w *WorkflowEngineWithChaos) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"executions":    w.executionCount.Load(),
		"rollbacks":     w.rollbackCount.Load(),
		"panics":        w.panicCount.Load(),
		"current_depth": w.currentDepth.Load(),
	}
}

// RunWorkflow is the step function that executes the workflow
func RunWorkflow(ctx context.Context, target chaoskit.Target) error {
	engine, ok := target.(*WorkflowEngineWithChaos)
	if !ok {
		return fmt.Errorf("target is not a WorkflowEngineWithChaos")
	}

	return engine.Execute(ctx)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=== ChaosKit Context-Based Chaos Injection Example ===")
	log.Println("Demonstrates how to use chaoskit.MaybePanic() and chaoskit.MaybeDelay()")
	log.Println("inside your own code to enable chaos injection")
	log.Println()

	// Create a workflow engine
	engine := NewWorkflowEngineWithChaos()

	// Build scenario with chaos injectors
	scenario := chaoskit.NewScenario("chaos-context-demo").
		WithTarget(engine).
		Step("run-workflow", RunWorkflow).
		// Inject chaos - these will be available via context
		Inject("delay", injectors.RandomDelayWithInterval(
			10*time.Millisecond,
			30*time.Millisecond,
			50*time.Millisecond,
		)).
		Inject("panic", injectors.PanicProbability(0.05)). // 5% chance of panic
		// Validators
		Assert("no_goroutine_leak", validators.GoroutineLimit(200)).
		Assert("recursion_depth_below_100", validators.RecursionDepthLimit(100)).
		Assert("no_slow_iteration", validators.NoSlowIteration(5*time.Second)).
		Repeat(20).
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
	log.Println("1. chaoskit.MaybePanic(ctx) - Call at critical points to allow panic injection")
	log.Println("2. chaoskit.MaybeDelay(ctx) - Call to inject delays from chaos context")
	log.Println("3. chaoskit.ShouldFail(ctx, probability) - Use for conditional failures")
	log.Println("\nThis approach allows chaos to happen INSIDE your code execution,")
	log.Println("not just before/after steps, making tests more realistic!")

	_ = executor.Reporter().SaveJUnitXML(report, "chaos-context-report.xml")

	// Exit with verdict code
	os.Exit(report.Verdict.ExitCode())
}
