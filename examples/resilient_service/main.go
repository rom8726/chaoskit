package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/injectors"
	"github.com/rom8726/chaoskit/validators"
)

// ResilientService demonstrates a service that handles chaos gracefully
type ResilientService struct {
	name            string
	requestCount    atomic.Int64
	successCount    atomic.Int64
	panicCount      atomic.Int64
	errorCount      atomic.Int64
	recoveredPanics atomic.Int64
}

func NewResilientService(name string) *ResilientService {
	return &ResilientService{
		name: name,
	}
}

func (s *ResilientService) Name() string {
	return s.name
}

func (s *ResilientService) Setup(ctx context.Context) error {
	log.Printf("[%s] Setting up service...", s.name)
	return nil
}

func (s *ResilientService) Teardown(ctx context.Context) error {
	log.Printf("[%s] Tearing down service...", s.name)
	stats := s.GetStats()
	requests := stats["requests"].(int64)
	success := stats["success"].(int64)
	panics := stats["panics"].(int64)
	errors := stats["errors"].(int64)
	recovered := stats["recovered"].(int64)
	log.Printf("[%s] Final stats: requests=%d, success=%d, panics=%d, errors=%d, recovered=%d",
		s.name, requests, success, panics, errors, recovered)
	return nil
}

func (s *ResilientService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"requests":     s.requestCount.Load(),
		"success":      s.successCount.Load(),
		"panics":       s.panicCount.Load(),
		"errors":       s.errorCount.Load(),
		"recovered":    s.recoveredPanics.Load(),
		"success_rate": float64(s.successCount.Load()) / float64(s.requestCount.Load()),
	}
}

// ProcessRequest processes a request with proper error and panic handling
func (s *ResilientService) ProcessRequest(ctx context.Context) error {
	s.requestCount.Add(1)

	// Use defer recover to handle any panics gracefully
	defer func() {
		if r := recover(); r != nil {
			s.panicCount.Add(1)
			s.recoveredPanics.Add(1)
			log.Printf("[%s] Recovered from panic: %v", s.name, r)
		}
	}()

	// Critical point 1: Potential panic injection
	chaoskit.MaybePanic(ctx)

	// Simulate some processing work
	chaoskit.MaybeDelay(ctx)

	// Critical point 2: Another potential panic
	chaoskit.MaybePanic(ctx)

	// Simulate potential error (but we handle it)
	if err := chaoskit.MaybeError(ctx); err != nil {
		// handle error
	}

	// Success case
	s.successCount.Add(1)

	return nil
}

// ProcessRequestWithRetry processes a request with retry logic
func (s *ResilientService) ProcessRequestWithRetry(ctx context.Context, maxRetries int) error {
	s.requestCount.Add(1)
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := func() (err error) {
			// Recover from panics in retry loop
			defer func() {
				if r := recover(); r != nil {
					s.panicCount.Add(1)
					s.recoveredPanics.Add(1)
					err = fmt.Errorf("recovered from panic: %v", r)
					log.Printf("[%s] Retry attempt %d: recovered from panic: %v", s.name, attempt+1, r)
				}
			}()

			// Potential chaos points
			chaoskit.MaybePanic(ctx)
			chaoskit.MaybeDelay(ctx)

			// Simulate work
			if err := chaoskit.MaybeError(ctx); err != nil { // 15% chance of error
				return fmt.Errorf("temporary error on attempt %d", attempt+1)
			}

			return nil
		}()

		if err == nil {
			s.successCount.Add(1)

			return nil
		}

		//s.errorCount.Add(1)
		if attempt < maxRetries-1 {
			// Wait before retry
			time.Sleep(10 * time.Millisecond)
		}
	}

	return fmt.Errorf("failed after %d retries", maxRetries)
}

// ExecuteStep is the main step function
func ExecuteStep(ctx context.Context, target chaoskit.Target) error {
	service, ok := target.(*ResilientService)
	if !ok {
		return fmt.Errorf("target is not a ResilientService")
	}

	// Process multiple requests to demonstrate resilience
	for i := 0; i < 5; i++ {
		// Use retry logic for better resilience
		if err := service.ProcessRequestWithRetry(ctx, 10); err != nil {
			// Log error but continue processing - this is expected behavior
			log.Printf("[%s] Request %d failed after retries: %v", service.name, i+1, err)
		}
	}

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=== ChaosKit Resilient Service Example ===")
	log.Println("Demonstrates a service that handles chaos gracefully")
	log.Println("All panics and errors are handled, resulting in VerdictPass")
	log.Println()

	// Create resilient service
	service := NewResilientService("resilient-api")

	// Build scenario with chaos injectors
	scenario := chaoskit.NewScenario("resilient-service-test").
		WithTarget(service).
		Step("process-requests", ExecuteStep).
		// Inject panics with moderate probability (will be recovered)
		Inject("panic", injectors.PanicProbability(0.15)). // 15% chance
		// Inject delays
		Inject("delay", injectors.RandomDelay(5*time.Millisecond, 20*time.Millisecond)).
		// Inject errors with 15% probability
		Inject("error", injectors.ErrorWithProbability("temporary error", 0.15)).
		// Validators - allow some panics since we recover from them
		Assert("panic_recovery", validators.NoPanics(20)).                      // Allow up to 20 panics (we have 25 iterations * 5 requests * 0.15 = ~19 expected)
		Assert("goroutine_limit", validators.GoroutineLimit(500)).              // High limit to allow for retries
		Assert("recursion_depth", validators.RecursionDepthLimit(50)).          // Reasonable recursion limit
		Assert("no_slow_iteration", validators.NoSlowIteration(2*time.Second)). // Prevent slow iterations
		Assert("no_errors", validators.MaxErrors(0)).
		Repeat(25). // 25 iterations to get good statistics
		Build()

	// Run scenario with executor
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	executor := chaoskit.NewExecutor(
		chaoskit.WithFailurePolicy(chaoskit.ContinueOnFailure), // Continue even if some iterations fail
		chaoskit.WithSlogLogger(logger),                        // Info level for cleaner output
	)

	log.Println("Starting chaos test...")
	if err := executor.Run(ctx, scenario); err != nil {
		log.Printf("Scenario execution completed with errors: %v", err)
	}

	// Get verdict and generate report
	thresholds := chaoskit.DefaultThresholds()
	// Adjust thresholds to be more lenient for chaos testing
	thresholds.MinSuccessRate = 0.5     // Allow 50% success rate (since we're injecting chaos)
	thresholds.MaxFailedIterations = 15 // Allow up to 15 failed iterations out of 25

	report, err := executor.Reporter().GetVerdict(thresholds)
	if err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	// Print detailed report
	log.Println("\n=== Chaos Test Report ===")
	log.Println(executor.Reporter().GenerateTextReport(report))

	// Print service statistics
	log.Println("\n=== Service Statistics ===")
	stats := service.GetStats()
	requests := stats["requests"].(int64)
	success := stats["success"].(int64)
	panics := stats["panics"].(int64)
	errors := stats["errors"].(int64)
	recovered := stats["recovered"].(int64)
	log.Printf("Total requests: %d", requests)
	log.Printf("Successful: %d", success)
	log.Printf("Panics (recovered): %d", panics)
	log.Printf("Errors (handled): %d", errors)
	log.Printf("Recovered panics: %d", recovered)
	if requests > 0 {
		successRate := float64(success) / float64(requests) * 100
		log.Printf("Success rate: %.2f%%", successRate)
	}

	log.Printf("\n=== Verdict: %s ===", report.Verdict.String())
	if report.Verdict == chaoskit.VerdictPass {
		log.Println("✅ All tests passed! Service demonstrated resilience to chaos.")
	} else {
		log.Printf("⚠️  Verdict: %s - Check report above for details", report.Verdict.String())
	}

	// Exit with verdict code
	os.Exit(report.Verdict.ExitCode())
}
