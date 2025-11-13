// Package chaoskit provides a modular framework for chaos engineering.
//
// ChaosKit enables systematic testing of system reliability through
// controlled fault injection and invariant validation.
//
// # Basic Usage
//
//	scenario := chaoskit.NewScenario("test").
//		WithTarget(mySystem).
//		Inject("delay", injectors.RandomDelay(5*time.Millisecond, 25*time.Millisecond)).
//		Assert("goroutines", validators.GoroutineLimit(100)).
//		Build()
//
//	executor := chaoskit.NewExecutor()
//	if err := executor.Run(ctx, scenario); err != nil {
//		log.Fatal(err)
//	}
//
// # Architecture
//
// ChaosKit follows clean architecture principles with clear separation between:
// - Scenarios: Define what to test
// - Injectors: Introduce faults into the system
// - Validators: Verify system invariants
// - Executor: Orchestrates scenario execution
//
// # Extension Points
//
// Implement the Injector or Validator interfaces to create custom chaos behaviors.
package chaoskit

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// recorderKey is a private type for context key
type recorderKey struct{}
type loggerKey struct{}

// Target represents the system under test.
// Implement this interface to define the system that will be subject to chaos testing.
//
// Example:
//
//	type MySystem struct{}
//
//	func (s *MySystem) Name() string { return "my-system" }
//
//	func (s *MySystem) Setup(ctx context.Context) error {
//		// Initialize system resources
//		return nil
//	}
//
//	func (s *MySystem) Teardown(ctx context.Context) error {
//		// Clean up resources
//		return nil
//	}
type Target interface {
	Name() string
	Setup(ctx context.Context) error
	Teardown(ctx context.Context) error
}

// Step represents a single step in a scenario.
// Steps are executed sequentially and can use chaos injection functions
// like MaybeDelay() and MaybePanic() to interact with active injectors.
//
// Example:
//
//	step := &myStep{name: "process-order"}
//	scenario.Step("process", step.Execute)
type Step interface {
	Name() string
	Execute(ctx context.Context, target Target) error
}

// InjectorType defines how an injector applies its effects
type InjectorType int

const (
	// InjectorTypeGlobal applies effects globally (CPU, Memory, Network proxies)
	InjectorTypeGlobal InjectorType = iota
	// InjectorTypeContext applies effects through context (Delay, Panic)
	InjectorTypeContext
	// InjectorTypeStep applies effects before/after steps
	InjectorTypeStep
	// InjectorTypeHybrid can work in multiple modes
	InjectorTypeHybrid
)

// Injector introduces faults into the system.
// Implement this interface to create custom chaos injection behaviors.
//
// Inject() is called when the injector should start injecting faults.
// Stop() is called when the injector should stop and clean up.
//
// Example implementations:
//   - DelayInjector: Adds random delays
//   - PanicInjector: Triggers panics with probability
//   - NetworkInjector: Introduces network latency/drops
//
// See the injectors package for reference implementations.
type Injector interface {
	Name() string
	Inject(ctx context.Context) error
	Stop(ctx context.Context) error
}

// CategorizedInjector provides information about injector type
type CategorizedInjector interface {
	Injector
	Type() InjectorType
}

// GlobalInjector indicates that injector applies global effects
type GlobalInjector interface {
	Injector
	IsGlobal() bool
}

// StepInjector can inject faults before/after step execution
type StepInjector interface {
	Injector
	BeforeStep(ctx context.Context) error
	AfterStep(ctx context.Context, err error) error
}

// ChaosProvider is a universal interface for context-based chaos injection
type ChaosProvider interface {
	Name() string
	Apply(ctx context.Context) bool
}

// ChaosDelayProvider provides delay injection capability
type ChaosDelayProvider interface {
	Injector
	GetChaosDelay(ctx context.Context) (time.Duration, bool)
}

// ChaosPanicProvider provides panic injection capability
type ChaosPanicProvider interface {
	Injector
	ShouldChaosPanic() bool
	GetPanicProbability() float64
}

// ChaosNetworkProvider provides network chaos injection capability
type ChaosNetworkProvider interface {
	Injector
	ShouldApplyNetworkChaos(host string, port int) bool
	GetNetworkLatency(host string, port int) (time.Duration, bool)
	ShouldDropConnection(host string, port int) bool
}

// ChaosContextCancellationProvider provides context cancellation capability
type ChaosContextCancellationProvider interface {
	Injector
	GetChaosContext(parent context.Context) (context.Context, context.CancelFunc)
	GetCancellationProbability() float64
}

// NetworkInjectorLifecycle manages network proxy setup/teardown
type NetworkInjectorLifecycle interface {
	Injector
	SetupNetwork(ctx context.Context) error
	TeardownNetwork(ctx context.Context) error
}

// MetricsProvider allows injectors to expose metrics
type MetricsProvider interface {
	Injector
	GetMetrics() map[string]interface{}
}

// Validator checks system invariants.
// Implement this interface to verify that the system maintains expected properties
// during chaos testing.
//
// Validate() is called after each scenario execution to check invariants.
// Return an error if the invariant is violated.
//
// Example implementations:
//   - GoroutineLimit: Ensures goroutine count stays below threshold
//   - RecursionDepthLimit: Verifies recursion depth doesn't exceed limit
//   - NoInfiniteLoop: Detects infinite loops
//
// See the validators package for reference implementations.
type Validator interface {
	Name() string
	Validate(ctx context.Context, target Target) error
}

// Resettable is implemented by validators that need to reset state between iterations
type Resettable interface {
	Reset()
}

// Logger is deprecated. Use *slog.Logger instead.
// This type is kept for backward compatibility but will be removed in a future version.
type Logger interface {
	Printf(format string, v ...any)
	Println(v ...any)
}

// --- Event recording & context helpers ---

// PanicRecorder is implemented by validators that can record panics during execution.
type PanicRecorder interface {
	RecordPanic(ctx context.Context)
}

// RecursionRecorder is implemented by validators that can record recursion/rollback depth.
type RecursionRecorder interface {
	RecordRecursion(depth int)
}

// EventRecorder provides a unified interface for recording runtime events from steps.
type EventRecorder interface {
	RecordPanic(ctx context.Context)
	RecordRecursionDepth(depth int)
}

// AttachRecorder attaches an EventRecorder to context.
func AttachRecorder(ctx context.Context, r EventRecorder) context.Context {
	return context.WithValue(ctx, recorderKey{}, r)
}

// RecordPanic records a panic via context-attached recorder (no-op if absent).
func RecordPanic(ctx context.Context) {
	if v := ctx.Value(recorderKey{}); v != nil {
		if r, ok := v.(EventRecorder); ok {
			r.RecordPanic(ctx)
		}
	}
}

// RecordRecursionDepth records recursion depth via context-attached recorder (no-op if absent).
func RecordRecursionDepth(ctx context.Context, depth int) {
	if v := ctx.Value(recorderKey{}); v != nil {
		if r, ok := v.(EventRecorder); ok {
			r.RecordRecursionDepth(depth)
		}
	}
}

// AttachLogger attaches a logger to context.
func AttachLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// GetLogger retrieves logger from context, or returns slog.Default() if not found.
func GetLogger(ctx context.Context) *slog.Logger {
	if v := ctx.Value(loggerKey{}); v != nil {
		if logger, ok := v.(*slog.Logger); ok {
			return logger
		}
	}
	return slog.Default()
}

// Run executes a scenario with default settings.
// This is a convenience function that creates an executor, runs the scenario,
// and prints the report.
//
// Example:
//
//	scenario := chaoskit.NewScenario("test").WithTarget(mySystem).Build()
//	if err := chaoskit.Run(ctx, scenario); err != nil {
//		log.Fatal(err)
//	}
func Run(ctx context.Context, scenario *Scenario) error {
	executor := NewExecutor()
	if err := executor.Run(ctx, scenario); err != nil {
		return err
	}

	fmt.Println(executor.Reporter().GenerateReport())

	return nil
}

// RunWithLogger executes a scenario with a custom logger (deprecated, use RunWithSlogLogger)
func RunWithLogger(ctx context.Context, scenario *Scenario, logger Logger) error {
	// Convert old Logger to slog.Logger for backward compatibility
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	executor := NewExecutor(WithSlogLogger(slogLogger))
	if err := executor.Run(ctx, scenario); err != nil {
		return err
	}

	logger.Println(executor.Reporter().GenerateReport())

	return nil
}

// RunWithSlogLogger executes a scenario with a structured logger.
// Use this function when you want to use structured logging with slog.
//
// Example:
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	scenario := chaoskit.NewScenario("test").WithTarget(mySystem).Build()
//	if err := chaoskit.RunWithSlogLogger(ctx, scenario, logger); err != nil {
//		log.Fatal(err)
//	}
func RunWithSlogLogger(ctx context.Context, scenario *Scenario, logger *slog.Logger) error {
	executor := NewExecutor(WithSlogLogger(logger))
	if err := executor.Run(ctx, scenario); err != nil {
		return err
	}

	logger.Info(executor.Reporter().GenerateReport())

	return nil
}

// defaultLogger wraps standard log package (deprecated)
type defaultLogger struct{}

func (d *defaultLogger) Printf(format string, v ...any) {
	slog.Default().Info(fmt.Sprintf(format, v...))
}

func (d *defaultLogger) Println(v ...any) {
	slog.Default().Info(fmt.Sprint(v...))
}

// NewDefaultLogger creates a default logger (deprecated, use slog.Default())
func NewDefaultLogger() Logger {
	return &defaultLogger{}
}
