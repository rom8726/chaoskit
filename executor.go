package chaoskit

import (
	"context"
	"fmt"
	"time"
)

// ExecutionResult contains the result of a scenario execution
type ExecutionResult struct {
	ScenarioName  string
	Success       bool
	Error         error
	Duration      time.Duration
	StepsExecuted int
	Timestamp     time.Time
}

// Executor runs scenarios
type Executor struct {
	metrics  *MetricsCollector
	reporter *Reporter
}

// NewExecutor creates a new executor
func NewExecutor() *Executor {
	return &Executor{
		metrics:  NewMetricsCollector(),
		reporter: NewReporter(),
	}
}

// Run executes a scenario
func (e *Executor) Run(ctx context.Context, scenario *Scenario) error {
	if scenario.target == nil {
		return fmt.Errorf("scenario %s has no target", scenario.name)
	}

	// Setup target
	if err := scenario.target.Setup(ctx); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}
	defer scenario.target.Teardown(ctx)

	// Start injectors
	for _, inj := range scenario.injectors {
		if err := inj.Inject(ctx); err != nil {
			return fmt.Errorf("injector %s failed: %w", inj.Name(), err)
		}
		defer inj.Stop(ctx)
	}

	// Execute scenario
	if scenario.duration > 0 {
		return e.runForDuration(ctx, scenario)
	}

	return e.runRepeated(ctx, scenario)
}

func (e *Executor) runRepeated(ctx context.Context, scenario *Scenario) error {
	for i := 0; i < scenario.repeat; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result := e.executeOnce(ctx, scenario)
		e.metrics.RecordExecution(result)
		e.reporter.AddResult(result)

		if result.Error != nil {
			return fmt.Errorf("execution %d failed: %w", i+1, result.Error)
		}
	}

	return nil
}

func (e *Executor) runForDuration(ctx context.Context, scenario *Scenario) error {
	ctx, cancel := context.WithTimeout(ctx, scenario.duration)
	defer cancel()

	iteration := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		result := e.executeOnce(ctx, scenario)
		e.metrics.RecordExecution(result)
		e.reporter.AddResult(result)

		if result.Error != nil {
			return fmt.Errorf("execution %d failed: %w", iteration+1, result.Error)
		}
		iteration++
	}
}

func (e *Executor) executeOnce(ctx context.Context, scenario *Scenario) ExecutionResult {
	start := time.Now()
	result := ExecutionResult{
		ScenarioName: scenario.name,
		Success:      true,
		Timestamp:    start,
	}

	// Execute steps
	for i, step := range scenario.steps {
		if err := step.Execute(ctx, scenario.target); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("step %s failed: %w", step.Name(), err)
			result.StepsExecuted = i
			result.Duration = time.Since(start)

			return result
		}
	}
	result.StepsExecuted = len(scenario.steps)

	// Run validators
	for _, val := range scenario.validators {
		if err := val.Validate(ctx, scenario.target); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("validator %s failed: %w", val.Name(), err)
			result.Duration = time.Since(start)
			return result
		}
	}

	result.Duration = time.Since(start)

	return result
}

// Metrics returns the metrics collector
func (e *Executor) Metrics() *MetricsCollector {
	return e.metrics
}

// Reporter returns the reporter
func (e *Executor) Reporter() *Reporter {
	return e.reporter
}
