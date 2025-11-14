package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/injectors"
	"github.com/rom8726/chaoskit/validators"
)

// ContinuousWorkflowEngine runs workflows continuously with different patterns
type ContinuousWorkflowEngine struct {
	totalRuns     atomic.Int64
	successRuns   atomic.Int64
	failedRuns    atomic.Int64
	rollbackCount atomic.Int64
	maxRecursion  int
	currentDepth  atomic.Int32
}

func NewContinuousWorkflowEngine() *ContinuousWorkflowEngine {
	return &ContinuousWorkflowEngine{
		maxRecursion: 50,
	}
}

func (e *ContinuousWorkflowEngine) Name() string {
	return "continuous-workflow-engine"
}

func (e *ContinuousWorkflowEngine) Setup(ctx context.Context) error {
	return nil
}

func (e *ContinuousWorkflowEngine) Teardown(ctx context.Context) error {
	return nil
}

func (e *ContinuousWorkflowEngine) Execute(ctx context.Context) error {
	e.totalRuns.Add(1)

	// Randomly select workflow pattern
	patterns := []func(context.Context) error{
		e.executeSimple,
		e.executeComplex,
		e.executeNested,
		e.executeParallel,
	}

	pattern := patterns[rand.Intn(len(patterns))]
	err := pattern(ctx)

	if err != nil {
		e.failedRuns.Add(1)

		return err
	}

	e.successRuns.Add(1)

	return nil
}

func (e *ContinuousWorkflowEngine) executeSimple(ctx context.Context) error {
	steps := []func(context.Context) error{
		e.stepValidate,
		e.stepProcess,
		e.stepCommit,
	}

	for i, step := range steps {
		if err := step(ctx); err != nil {
			e.rollback(ctx, i)

			return err
		}
	}

	return nil
}

func (e *ContinuousWorkflowEngine) executeComplex(ctx context.Context) error {
	if err := e.stepValidate(ctx); err != nil {
		e.rollback(ctx, 0)

		return err
	}

	// Parallel processing
	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		errCh <- e.stepProcess(ctx)
	}()
	go func() {
		defer wg.Done()
		errCh <- e.stepProcess(ctx)
	}()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			e.rollback(ctx, 1)

			return err
		}
	}

	if err := e.stepCommit(ctx); err != nil {
		e.rollback(ctx, 2)

		return err
	}

	return nil
}

func (e *ContinuousWorkflowEngine) executeNested(ctx context.Context) error {
	return e.executeNestedRecursive(ctx, 0)
}

func (e *ContinuousWorkflowEngine) executeNestedRecursive(ctx context.Context, depth int) error {
	current := e.currentDepth.Add(1)
	defer e.currentDepth.Add(-1)

	chaoskit.RecordRecursionDepth(ctx, int(current))

	if current > int32(e.maxRecursion) {
		return fmt.Errorf("max recursion depth %d exceeded", e.maxRecursion)
	}

	if depth > 5 {
		return nil
	}

	if err := e.stepProcess(ctx); err != nil {
		e.rollback(ctx, depth)

		return err
	}

	// 30% chance of nested call
	if rand.Float64() < 0.3 {
		return e.executeNestedRecursive(ctx, depth+1)
	}

	return nil
}

func (e *ContinuousWorkflowEngine) executeParallel(ctx context.Context) error {
	workers := 3
	errCh := make(chan error, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := e.stepProcess(ctx); err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			e.rollback(ctx, 0)

			return err
		}
	}

	return nil
}

func (e *ContinuousWorkflowEngine) stepValidate(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		time.Sleep(time.Millisecond * time.Duration(3+rand.Intn(7)))
		if rand.Float64() < 0.05 {
			return fmt.Errorf("validation failed")
		}

		return nil
	}
}

func (e *ContinuousWorkflowEngine) stepProcess(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		time.Sleep(time.Millisecond * time.Duration(5+rand.Intn(15)))
		if rand.Float64() < 0.05 {
			return fmt.Errorf("processing failed")
		}

		return nil
	}
}

func (e *ContinuousWorkflowEngine) stepCommit(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		time.Sleep(time.Millisecond * time.Duration(3+rand.Intn(7)))
		if rand.Float64() < 0.03 {
			return fmt.Errorf("commit failed")
		}

		return nil
	}
}

func (e *ContinuousWorkflowEngine) rollback(ctx context.Context, fromStep int) {
	e.rollbackCount.Add(1)

	current := e.currentDepth.Add(1)
	defer e.currentDepth.Add(-1)

	chaoskit.RecordRecursionDepth(ctx, int(current))

	if current > int32(e.maxRecursion) {
		return
	}

	time.Sleep(time.Millisecond * time.Duration(fromStep*2))
}

func (e *ContinuousWorkflowEngine) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_runs":    e.totalRuns.Load(),
		"success_runs":  e.successRuns.Load(),
		"failed_runs":   e.failedRuns.Load(),
		"rollbacks":     e.rollbackCount.Load(),
		"current_depth": e.currentDepth.Load(),
	}
}

func RunWorkflow(ctx context.Context, target chaoskit.Target) error {
	engine, ok := target.(*ContinuousWorkflowEngine)
	if !ok {
		return fmt.Errorf("target is not a ContinuousWorkflowEngine")
	}

	return engine.Execute(ctx)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=== ChaosKit Continuous Testing ===")
	log.Println("Running continuous chaos tests on workflow engine")
	log.Println("Press Ctrl+C to stop")
	log.Println()

	engine := NewContinuousWorkflowEngine()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Statistics reporter
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := engine.GetStats()
				log.Printf("[Stats] Total=%d Success=%d Failed=%d Rollbacks=%d Depth=%d",
					stats["total_runs"], stats["success_runs"], stats["failed_runs"],
					stats["rollbacks"], stats["current_depth"])
			}
		}
	}()

	// Run scenarios continuously
	scenarioCount := 0
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			log.Println("\n[Main] Shutting down...")
			cancel()

			// Wait a bit for scenarios to finish
			time.Sleep(1 * time.Second)

			// Print final stats
			stats := engine.GetStats()
			log.Printf("\n=== Final Statistics ===")
			log.Printf("Total scenarios: %d", scenarioCount)
			log.Printf("Total workflow runs: %d", stats["total_runs"])
			log.Printf("Successful runs: %d", stats["success_runs"])
			log.Printf("Failed runs: %d", stats["failed_runs"])
			log.Printf("Rollbacks: %d", stats["rollbacks"])

			// Note: In continuous mode, we don't have a single executor
			// Each scenario runs in its own goroutine, so we can't aggregate verdicts
			// This is a limitation of the continuous example design
			log.Println("\nNote: Continuous mode runs scenarios independently.")
			log.Println("For verdict reporting, use a single scenario with RunFor() duration.")

			return

		case <-ticker.C:
			scenarioCount++

			// Create and run scenario
			go func(num int) {
				scenario := createRandomScenario(engine, num)

				expCtx, expCancel := context.WithTimeout(ctx, 10*time.Second)
				defer expCancel()

				executor := chaoskit.NewExecutor()
				if err := executor.Run(expCtx, scenario); err != nil {
					log.Printf("[Scenario %d] Error: %v", num, err)
				}
			}(scenarioCount)
		}
	}
}

func createRandomScenario(engine *ContinuousWorkflowEngine, num int) *chaoskit.Scenario {
	scenarios := []struct {
		name       string
		injectors  []chaoskit.Injector
		validators []chaoskit.Validator
		repeat     int
	}{
		{
			name:      fmt.Sprintf("quick-test-%d", num),
			injectors: []chaoskit.Injector{},
			validators: []chaoskit.Validator{
				validators.RecursionDepthLimit(100),
			},
			repeat: 5,
		},
		{
			name: fmt.Sprintf("delayed-test-%d", num),
			injectors: []chaoskit.Injector{
				injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond),
			},
			validators: []chaoskit.Validator{
				validators.RecursionDepthLimit(100),
				validators.GoroutineLimit(150),
			},
			repeat: 3,
		},
		{
			name: fmt.Sprintf("stress-test-%d", num),
			injectors: []chaoskit.Injector{
				injectors.RandomDelay(5*time.Millisecond, 20*time.Millisecond),
				injectors.PanicProbability(0.02),
			},
			validators: []chaoskit.Validator{
				validators.RecursionDepthLimit(100),
				validators.GoroutineLimit(200),
				validators.NoSlowIteration(3 * time.Second),
			},
			repeat: 10,
		},
	}

	selected := scenarios[rand.Intn(len(scenarios))]

	builder := chaoskit.NewScenario(selected.name).
		WithTarget(engine).
		Step("run-workflow", RunWorkflow).
		Repeat(selected.repeat)

	for _, inj := range selected.injectors {
		builder = builder.Inject(inj.Name(), inj)
	}

	for _, val := range selected.validators {
		builder = builder.Assert(val.Name(), val)
	}

	return builder.Build()
}
