package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/injectors"
	"github.com/rom8726/chaoskit/validators"
)

// ProcessingService simulates a service that processes tasks
// Some tasks may contain infinite loops that need to be detected
type ProcessingService struct {
	processedCount int
	failedCount    int
}

func NewProcessingService() *ProcessingService {
	return &ProcessingService{}
}

func (p *ProcessingService) Name() string {
	return "processing-service"
}

func (p *ProcessingService) Setup(ctx context.Context) error {
	log.Println("[Service] Setting up processing service...")
	return nil
}

func (p *ProcessingService) Teardown(ctx context.Context) error {
	log.Println("[Service] Tearing down processing service...")
	log.Printf("[Service] Processed: %d, Failed: %d\n", p.processedCount, p.failedCount)
	return nil
}

// ProcessTask simulates processing a task
// Returns true if task should cause infinite loop (for testing)
func (p *ProcessingService) ProcessTask(ctx context.Context, taskID int, shouldLoop bool) error {
	if shouldLoop {
		// Simulate infinite loop - this should be detected by validator
		log.Printf("[Service] Task %d: Starting (will loop infinitely)\n", taskID)
		for {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Normal processing
	log.Printf("[Service] Task %d: Processing...\n", taskID)
	processingTime := time.Duration(50+rand.Intn(100)) * time.Millisecond
	time.Sleep(processingTime)
	p.processedCount++
	log.Printf("[Service] Task %d: Completed in %v\n", taskID, processingTime)
	return nil
}

// Step functions for the scenario

// ProcessNormalTask processes a normal task that completes successfully
func ProcessNormalTask(ctx context.Context, target chaoskit.Target) error {
	service, ok := target.(*ProcessingService)
	if !ok {
		return fmt.Errorf("target is not a ProcessingService")
	}

	// Random task ID
	taskID := rand.Intn(1000)
	return service.ProcessTask(ctx, taskID, false)
}

// ProcessInfiniteLoopTask processes a task that will loop infinitely
// This should be detected by the infinite loop validator
func ProcessInfiniteLoopTask(ctx context.Context, target chaoskit.Target) error {
	service, ok := target.(*ProcessingService)
	if !ok {
		return fmt.Errorf("target is not a ProcessingService")
	}

	// This task will loop infinitely (unless context is cancelled)
	taskID := 9999
	return service.ProcessTask(ctx, taskID, true)
}

// ProcessRandomTask randomly processes either normal or infinite loop task
func ProcessRandomTask(ctx context.Context, target chaoskit.Target) error {
	// 90% chance of normal task, 10% chance of infinite loop
	if rand.Float64() < 0.1 {
		return ProcessInfiniteLoopTask(ctx, target)
	}

	return ProcessNormalTask(ctx, target)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=== ChaosKit Infinite Loop Detection Example ===")
	log.Println("This example demonstrates detection of infinite loops in step execution")
	log.Println()

	// Create processing service
	service := NewProcessingService()

	// Build scenario with infinite loop detection
	// The NoInfiniteLoop validator will wrap each step with a timeout
	// If a step exceeds the timeout, it's considered an infinite loop
	scenario := chaoskit.NewScenario("infinite-loop-detection").
		WithTarget(service).
		// Add steps - some may loop infinitely
		Step("process-normal", ProcessNormalTask).
		Step("process-random", ProcessRandomTask).
		// Inject some chaos
		Inject("delay", injectors.RandomDelay(5*time.Millisecond, 15*time.Millisecond)).
		// Add infinite loop validator with 200ms timeout
		// Any step taking longer than 200ms will be detected as infinite loop
		Assert("no_infinite_loop", validators.NoInfiniteLoop(200*time.Millisecond)).
		// Also add other validators
		Assert("no_goroutine_leak", validators.GoroutineLimit(100)).
		Repeat(10).
		Build()

	// Run scenario with executor
	ctx := context.Background()
	executor := chaoskit.NewExecutor(
		chaoskit.WithFailurePolicy(chaoskit.ContinueOnFailure),
	)

	log.Println("Running scenario...")
	if err := executor.Run(ctx, scenario); err != nil {
		log.Printf("Scenario execution completed with errors: %v\n", err)
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

	// Note: Infinite loop detections are logged during execution
	// and will appear in the report if steps exceed the timeout

	// Exit with verdict code
	os.Exit(report.Verdict.ExitCode())
}
