package chaoskit

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
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

// FailurePolicy defines how the executor handles failures
type FailurePolicy int

const (
	// FailFast stops execution on first failure
	FailFast FailurePolicy = iota
	// ContinueOnFailure continues execution even after failures
	ContinueOnFailure
)

// Executor runs scenarios
type Executor struct {
	metrics       *MetricsCollector
	reporter      *Reporter
	logger        Logger
	failurePolicy FailurePolicy
}

// ExecutorOption configures an Executor
type ExecutorOption func(*Executor)

// WithLogger sets a custom logger
func WithLogger(logger Logger) ExecutorOption {
	return func(e *Executor) {
		e.logger = logger
	}
}

// WithFailurePolicy sets the failure handling policy
func WithFailurePolicy(policy FailurePolicy) ExecutorOption {
	return func(e *Executor) {
		e.failurePolicy = policy
	}
}

// WithMetrics sets a custom metrics collector
func WithMetrics(metrics *MetricsCollector) ExecutorOption {
	return func(e *Executor) {
		e.metrics = metrics
	}
}

// WithReporter sets a custom reporter
func WithReporter(reporter *Reporter) ExecutorOption {
	return func(e *Executor) {
		e.reporter = reporter
	}
}

// NewExecutor creates a new executor with options
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		metrics:       NewMetricsCollector(),
		reporter:      NewReporter(),
		logger:        NewDefaultLogger(),
		failurePolicy: FailFast,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// internal event recorder that forwards to validators
type validatorEventRecorder struct{ validators []Validator }

func (r *validatorEventRecorder) RecordPanic() {
	for _, v := range r.validators {
		if pr, ok := v.(PanicRecorder); ok {
			pr.RecordPanic()
		}
	}
}

func (r *validatorEventRecorder) RecordRecursionDepth(depth int) {
	for _, v := range r.validators {
		if rr, ok := v.(RecursionRecorder); ok {
			rr.RecordRecursion(depth)
		}
	}
}

// getAllInjectors collects all injectors from scenario (both direct and from scopes)
func (e *Executor) getAllInjectors(scenario *Scenario) []Injector {
	allInjectors := make([]Injector, 0, len(scenario.injectors))

	// Add direct injectors
	allInjectors = append(allInjectors, scenario.injectors...)

	// Add injectors from scopes
	for _, scope := range scenario.scopes {
		e.log("Scope '%s' contains %d injector(s)", scope.name, len(scope.injectors))
		allInjectors = append(allInjectors, scope.injectors...)
	}

	return allInjectors
}

// Run executes a scenario
func (e *Executor) Run(ctx context.Context, scenario *Scenario) error {
	if scenario.target == nil {
		return fmt.Errorf("scenario %s has no target", scenario.name)
	}

	// Create a deterministic random generator if seed is set
	var rng *rand.Rand
	if scenario.seed != nil {
		rng = rand.New(rand.NewSource(*scenario.seed))
		e.log("Using deterministic seed: %d", *scenario.seed)
	} else {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}
	ctx = AttachRand(ctx, rng)

	// Setup target
	if err := scenario.target.Setup(ctx); err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}
	defer func() {
		if err := scenario.target.Teardown(ctx); err != nil {
			e.log("Teardown error: %v", err)
		}
	}()

	// Collect all injectors (from direct injectors and scopes)
	allInjectors := e.getAllInjectors(scenario)

	// Setup network injectors first (if they need proxy setup)
	networkInjectors := make([]Injector, 0)
	for _, inj := range allInjectors {
		if lifecycle, ok := inj.(NetworkInjectorLifecycle); ok {
			if err := lifecycle.SetupNetwork(ctx); err != nil {
				return fmt.Errorf("network setup failed for %s: %w", inj.Name(), err)
			}
			networkInjectors = append(networkInjectors, inj)
			e.log("Network injector %s setup completed", inj.Name())
		}
	}
	defer func() {
		for _, inj := range networkInjectors {
			if lifecycle, ok := inj.(NetworkInjectorLifecycle); ok {
				if err := lifecycle.TeardownNetwork(ctx); err != nil {
					e.log("Network teardown error for %s: %v", inj.Name(), err)
				}
			}
		}
	}()

	// Start injectors
	activeInjectors := make([]Injector, 0, len(allInjectors))
	for _, inj := range allInjectors {
		if err := inj.Inject(ctx); err != nil {
			e.log("Injector %s failed to start: %v", inj.Name(), err)
			// Stop already started injectors
			e.stopInjectors(ctx, activeInjectors)

			return fmt.Errorf("injector %s failed: %w", inj.Name(), err)
		}
		activeInjectors = append(activeInjectors, inj)
	}
	defer e.stopInjectors(ctx, activeInjectors)

	// Execute scenario
	if scenario.duration > 0 {
		return e.runForDuration(ctx, scenario)
	}

	return e.runRepeated(ctx, scenario)
}

func (e *Executor) stopInjectors(ctx context.Context, injectors []Injector) {
	for _, inj := range injectors {
		if err := inj.Stop(ctx); err != nil {
			e.log("Injector %s failed to stop: %v", inj.Name(), err)
		}
	}
}

func (e *Executor) runRepeated(ctx context.Context, scenario *Scenario) error {
	var firstError error

	for i := 0; i < scenario.repeat; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Reset validators before each iteration
		e.resetValidators(scenario.validators)

		result := e.executeOnce(ctx, scenario)
		e.metrics.RecordExecution(result)
		e.reporter.AddResult(result)

		if result.Error != nil {
			if firstError == nil {
				firstError = fmt.Errorf("execution %d failed: %w", i+1, result.Error)
			}

			if e.failurePolicy == FailFast {
				return firstError
			}
			// Continue on failure - just log it
			e.log("Execution %d failed (continuing): %v", i+1, result.Error)
		}
	}

	return firstError
}

func (e *Executor) runForDuration(ctx context.Context, scenario *Scenario) error {
	ctx, cancel := context.WithTimeout(ctx, scenario.duration)
	defer cancel()

	iteration := 0
	var firstError error

	for {
		select {
		case <-ctx.Done():
			// Timeout is expected, not an error
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return firstError
			}

			return ctx.Err()
		default:
		}

		// Reset validators before each iteration
		e.resetValidators(scenario.validators)

		result := e.executeOnce(ctx, scenario)
		e.metrics.RecordExecution(result)
		e.reporter.AddResult(result)

		if result.Error != nil {
			if firstError == nil {
				firstError = fmt.Errorf("execution %d failed: %w", iteration+1, result.Error)
			}

			if e.failurePolicy == FailFast {
				return firstError
			}
			// Continue on failure - just log it
			e.log("Execution %d failed (continuing): %v", iteration+1, result.Error)
		}
		iteration++
	}
}

func (e *Executor) resetValidators(validators []Validator) {
	for _, val := range validators {
		if resettable, ok := val.(Resettable); ok {
			resettable.Reset()
		}
	}
}

func (e *Executor) executeOnce(ctx context.Context, scenario *Scenario) ExecutionResult {
	start := time.Now()
	result := ExecutionResult{
		ScenarioName: scenario.name,
		Success:      true,
		Timestamp:    start,
	}

	// Ensure rand generator is attached (in case executeOnce is called directly)
	if ctx.Value(randKey{}) == nil {
		var rng *rand.Rand
		if scenario.seed != nil {
			rng = rand.New(rand.NewSource(*scenario.seed))
		} else {
			rng = rand.New(rand.NewSource(rand.Int63()))
		}
		ctx = AttachRand(ctx, rng)
	}

	// Attach event recorder to context for steps to use
	recorder := &validatorEventRecorder{validators: scenario.validators}
	ctx = AttachRecorder(ctx, recorder)

	// Collect all injectors (from direct injectors and scopes)
	allInjectors := e.getAllInjectors(scenario)

	// Attach chaos context for user code to use
	chaosCtx := e.buildChaosContext(allInjectors)
	ctx = AttachChaos(ctx, chaosCtx)

	// Execute steps with panic recovery
	for i, step := range scenario.steps {
		stepErr := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					// record panic and convert to error
					recorder.RecordPanic()
					err = fmt.Errorf("panic in step %s: %v", step.Name(), r)
				}
			}()

			// Apply injectors before step
			for _, inj := range allInjectors {
				if stepInj, ok := inj.(StepInjector); ok {
					if err := stepInj.BeforeStep(ctx); err != nil {
						return fmt.Errorf("injector %s before step failed: %w", inj.Name(), err)
					}
				}
			}

			// Execute step
			stepErr := step.Execute(ctx, scenario.target)

			// Apply injectors after step
			for _, inj := range allInjectors {
				if stepInj, ok := inj.(StepInjector); ok {
					if err := stepInj.AfterStep(ctx, stepErr); err != nil {
						return fmt.Errorf("injector %s after step failed: %w", inj.Name(), err)
					}
				}
			}

			return stepErr
		}()

		if stepErr != nil {
			result.Success = false
			result.Error = fmt.Errorf("step %s failed: %w", step.Name(), stepErr)
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

func (e *Executor) buildChaosContext(injectors []Injector) *ChaosContext {
	chaos := &ChaosContext{
		providers: make(map[string]ChaosProvider),
	}

	// Find delay injector
	for _, inj := range injectors {
		if delayProvider, ok := inj.(ChaosDelayProvider); ok {
			chaos.delayFunc = func() bool {
				delay, ok := delayProvider.GetChaosDelay()
				if ok && delay > 0 {
					fmt.Printf("[CHAOS] Delay injected in user code: %v\n", delay)
					time.Sleep(delay)

					return true
				}

				return false
			}
		}

		if panicProvider, ok := inj.(ChaosPanicProvider); ok {
			chaos.panicFunc = func() bool {
				if panicProvider.ShouldChaosPanic() {
					fmt.Printf("[CHAOS] Panic triggered in user code (probability: %.2f)\n",
						panicProvider.GetPanicProbability())

					return true
				}

				return false
			}
			chaos.panicProbability = panicProvider.GetPanicProbability()
		}

		// Find network injector
		if networkProvider, ok := inj.(ChaosNetworkProvider); ok {
			chaos.networkFunc = func(host string, port int) bool {
				if !networkProvider.ShouldApplyNetworkChaos(host, port) {
					return false
				}

				// Apply latency if configured
				if latency, hasLatency := networkProvider.GetNetworkLatency(host, port); hasLatency && latency > 0 {
					fmt.Printf("[CHAOS] Network latency injected: %s:%d -> %v\n", host, port, latency)
					time.Sleep(latency)

					return true
				}

				// Check for connection drop
				if networkProvider.ShouldDropConnection(host, port) {
					fmt.Printf("[CHAOS] Network connection drop simulated: %s:%d\n", host, port)

					return true
				}

				return false
			}
		}

		// Find context cancellation injector
		if cancellationProvider, ok := inj.(ChaosContextCancellationProvider); ok {
			chaos.cancellationFunc = func(parent context.Context) (context.Context, context.CancelFunc) {
				return cancellationProvider.GetChaosContext(parent)
			}
		}

		// Register universal providers
		if universalProvider, ok := inj.(ChaosProvider); ok {
			chaos.RegisterProvider(universalProvider)
		}

		// Collect metrics if available
		if metricsProvider, ok := inj.(MetricsProvider); ok {
			metrics := metricsProvider.GetMetrics()
			e.metrics.RecordInjectorMetrics(inj.Name(), metrics)
		}
	}

	return chaos
}

func (e *Executor) log(format string, v ...any) {
	if e.logger != nil {
		e.logger.Printf(format, v...)
	}
}

// Metrics returns the metrics collector
func (e *Executor) Metrics() *MetricsCollector {
	return e.metrics
}

// Reporter returns the reporter
func (e *Executor) Reporter() *Reporter {
	return e.reporter
}
