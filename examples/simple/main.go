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

// WorkflowEngine simulates a saga-based workflow engine with rollback capability
type WorkflowEngine struct {
	executionCount atomic.Int64
	rollbackCount  atomic.Int64
	maxDepth       int
	currentDepth   atomic.Int32
}

func NewWorkflowEngine() *WorkflowEngine {
	return &WorkflowEngine{
		maxDepth: 50,
	}
}

func (w *WorkflowEngine) Name() string {
	return "workflow-engine"
}

func (w *WorkflowEngine) Setup(ctx context.Context) error {
	log.Println("[Engine] Setting up workflow engine...")

	return nil
}

func (w *WorkflowEngine) Teardown(ctx context.Context) error {
	log.Println("[Engine] Tearing down workflow engine...")
	stats := w.GetStats()
	log.Printf("[Engine] Final stats: %+v\n", stats)

	return nil
}

func (w *WorkflowEngine) Execute(ctx context.Context) error {
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

func (w *WorkflowEngine) executeStep(ctx context.Context, step string, depth int) error {
	// Track recursion depth
	current := w.currentDepth.Add(1)
	defer w.currentDepth.Add(-1)

	// Record recursion depth for validators
	chaoskit.RecordRecursionDepth(ctx, int(current))

	if current > int32(w.maxDepth) {
		return fmt.Errorf("max recursion depth exceeded: %d", current)
	}

	// Simulate work
	time.Sleep(time.Millisecond * time.Duration(5+rand.Intn(10)))

	// Random failures for testing (10% chance)
	if rand.Float64() < 0.1 {
		return fmt.Errorf("step %s failed randomly", step)
	}

	return nil
}

func (w *WorkflowEngine) rollback(ctx context.Context, fromStep int) {
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

		time.Sleep(time.Millisecond * 2)
	}
}

func (w *WorkflowEngine) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"executions":    w.executionCount.Load(),
		"rollbacks":     w.rollbackCount.Load(),
		"current_depth": w.currentDepth.Load(),
	}
}

// RunWorkflow is the step function that executes the workflow
func RunWorkflow(ctx context.Context, target chaoskit.Target) error {
	engine, ok := target.(*WorkflowEngine)
	if !ok {
		return fmt.Errorf("target is not a WorkflowEngine")
	}

	return engine.Execute(ctx)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=== ChaosKit Simple Example ===")
	log.Println("Testing workflow engine with chaos injectors and validators")
	log.Println()

	// Create workflow engine
	engine := NewWorkflowEngine()

	// Build scenario
	scenario := chaoskit.NewScenario("workflow-reliability-test").
		WithTarget(engine).
		Step("run-workflow", RunWorkflow).
		// Inject random delays
		Inject("delay", injectors.RandomDelay(5*time.Millisecond, 25*time.Millisecond)).
		// Inject random panics (low probability)
		Inject("panic", injectors.PanicProbability(0.01)).
		// Validators
		Assert("no_goroutine_leak", validators.GoroutineLimit(200)).
		Assert("recursion_depth_below_100", validators.RecursionDepthLimit(100)).
		Assert("no_slow_iteration", validators.NoSlowIteration(5*time.Second)).
		Repeat(50).
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

	// Exit with verdict code
	os.Exit(report.Verdict.ExitCode())
}
