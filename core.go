package chaoskit

import (
	"context"
	"fmt"
	"log"
	"time"
)

// recorderKey is a private type for context key
type recorderKey struct{}

// Target represents the system under test
type Target interface {
	Name() string
	Setup(ctx context.Context) error
	Teardown(ctx context.Context) error
}

// Step represents a single step in a scenario
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

// Injector introduces faults into the system
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
	GetChaosDelay() (time.Duration, bool)
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

// Validator checks system invariants
type Validator interface {
	Name() string
	Validate(ctx context.Context, target Target) error
}

// Resettable is implemented by validators that need to reset state between iterations
type Resettable interface {
	Reset()
}

// Logger is a simple logging interface
type Logger interface {
	Printf(format string, v ...any)
	Println(v ...any)
}

// --- Event recording & context helpers ---

// PanicRecorder is implemented by validators that can record panics during execution.
type PanicRecorder interface {
	RecordPanic()
}

// RecursionRecorder is implemented by validators that can record recursion/rollback depth.
type RecursionRecorder interface {
	RecordRecursion(depth int)
}

// EventRecorder provides a unified interface for recording runtime events from steps.
type EventRecorder interface {
	RecordPanic()
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
			r.RecordPanic()
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

// Run executes a scenario with default settings
func Run(ctx context.Context, scenario *Scenario) error {
	executor := NewExecutor()
	if err := executor.Run(ctx, scenario); err != nil {
		return err
	}

	fmt.Println(executor.Reporter().GenerateReport())

	return nil
}

// RunWithLogger executes a scenario with a custom logger
func RunWithLogger(ctx context.Context, scenario *Scenario, logger Logger) error {
	executor := NewExecutor(WithLogger(logger))
	if err := executor.Run(ctx, scenario); err != nil {
		return err
	}

	logger.Println(executor.Reporter().GenerateReport())

	return nil
}

// defaultLogger wraps standard log package
type defaultLogger struct{}

func (d *defaultLogger) Printf(format string, v ...any) {
	log.Printf(format, v...)
}

func (d *defaultLogger) Println(v ...any) {
	log.Println(v...)
}

// NewDefaultLogger creates a default logger
func NewDefaultLogger() Logger {
	return &defaultLogger{}
}
