# ChaosKit Tutorial

This tutorial provides a progressive guide to using ChaosKit, from basic scenarios to advanced patterns.

## Table of Contents

1. [Part 1: Basic Scenario](#part-1-basic-scenario)
2. [Part 2: Adding Validators](#part-2-adding-validators)
3. [Part 3: Multiple Injectors](#part-3-multiple-injectors)
4. [Part 4: Context-Based Chaos](#part-4-context-based-chaos)
5. [Part 5: Custom Injectors and Providers](#part-5-custom-injectors-and-providers)
6. [Part 6: Production Usage Patterns](#part-6-production-usage-patterns)
7. [Part 7: Verdicts and Quality Gates](#part-7-verdicts-and-quality-gates)
8. [Part 8: Metrics and Exporters](#part-8-metrics-and-exporters)
9. [Part 9: Go Testing Integration](#part-9-go-testing-integration)

---

## Part 1: Basic Scenario

Let's start with the simplest possible scenario: a basic system with a single step and no chaos injection.

### Step 1.1: Define Your Target

First, implement the `Target` interface for your system:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/rom8726/chaoskit"
)

type MySystem struct {
    name string
}

func (s *MySystem) Name() string {
    return s.name
}

func (s *MySystem) Setup(ctx context.Context) error {
    fmt.Println("Setting up system...")
    return nil
}

func (s *MySystem) Teardown(ctx context.Context) error {
    fmt.Println("Tearing down system...")
    return nil
}
```

### Step 1.2: Create a Simple Step

A step is a function that executes your system logic:

```go
func ExecuteSystem(ctx context.Context, target chaoskit.Target) error {
    system := target.(*MySystem)
    fmt.Printf("Executing system: %s\n", system.Name())
    
    // Your system logic here
    // ...
    
    return nil
}
```

### Step 1.3: Build and Run a Scenario

Now create a scenario and run it:

```go
func main() {
    system := &MySystem{name: "my-system"}
    
    scenario := chaoskit.NewScenario("basic-test").
        WithTarget(system).
        Step("execute", ExecuteSystem).
        Repeat(5).
        Build()
    
    ctx := context.Background()
    if err := chaoskit.Run(ctx, scenario); err != nil {
        log.Fatal(err)
    }
}
```

**Output:**
```
Setting up system...
Executing system: my-system
Executing system: my-system
Executing system: my-system
Executing system: my-system
Executing system: my-system
Tearing down system...

ChaosKit Execution Report
========================
Total Executions: 5
Success: 5
Failed: 0
Success Rate: 100.00%
Average Duration: 1ms
```

---

## Part 2: Adding Validators

Validators check that your system maintains invariants during chaos testing. Each validator reports a severity (`CRITICAL`, `WARNING`, or `INFO`) that feeds into verdict calculation later.

### Step 2.1: Add a Goroutine Leak Validator

```go
import "github.com/rom8726/chaoskit/validators"

scenario := chaoskit.NewScenario("with-validators").
    WithTarget(system).
    Step("execute", ExecuteSystem).
    Assert("no-goroutine-leak", validators.GoroutineLimit(100)).
    Repeat(10).
    Build()
```

### Step 2.2: Record Events for Validators

Some validators need you to record events in your code. The executor attaches an event recorder to the context, so you can log measurements without wiring anything manually:

```go
func ExecuteSystem(ctx context.Context, target chaoskit.Target) error {
    // Record recursion depth for RecursionDepthValidator
    chaoskit.RecordRecursionDepth(ctx, currentDepth)
    
    // Propagate domain errors to ErrorValidator / MaxErrors
    if err := performOperation(); err != nil {
        chaoskit.RecordError(ctx)
        return err
    }
    
    // Your logic that might recurse
    if shouldRecurse {
        return ExecuteSystem(ctx, target) // Recursive call
    }
    
    return nil
}
```

### Step 2.3: Multiple Validators

Add multiple validators to check different invariants and severities:

```go
scenario := chaoskit.NewScenario("comprehensive-validation").
    WithTarget(system).
    Step("execute", ExecuteSystem).
    Assert("goroutines", validators.GoroutineLimit(200)).
    Assert("recursion", validators.RecursionDepthLimit(50)).
    Assert("no-infinite-loop", validators.NoSlowIteration(5*time.Second)).
    Assert("memory", validators.MemoryUnderLimit(512*1024*1024)). // 512MB
    Assert("max-errors", validators.MaxErrors(25)).                // Warning severity
    Repeat(100).
    Build()
```

**What happens:**
- If any validator fails, the scenario stops (or continues, depending on failure policy)
- Validators log warnings when approaching limits (80% threshold)
- Validator severity determines whether the verdict fails or marks the run as unstable
- All validation results are included in the final report

---

## Part 3: Multiple Injectors

Injectors introduce faults into your system. Let's add multiple types of chaos.

### Step 3.1: Basic Injectors

```go
import (
    "io"
    "time"

    "github.com/rom8726/chaoskit/injectors"
    "github.com/rom8726/chaoskit/validators"
)

scenario := chaoskit.NewScenario("multiple-injectors").
    WithTarget(system).
    Step("execute", ExecuteSystem).
    // Delay injection: random delays between 10-50ms
    Inject("delay", injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond)).
    // Panic injection: 1% chance of panic
    Inject("panic", injectors.PanicProbability(0.01)).
    // Error injection: return io.ErrUnexpectedEOF 5% of the time via MaybeError()
    Inject("errors", injectors.ErrorWithProbability(io.ErrUnexpectedEOF, 0.05)).
    Assert("no-panics", validators.NoPanics(5)). // Allow up to 5 panics
    Repeat(100).
    Build()
```

Inside your step, surface the injected error with `chaoskit.MaybeError(ctx)`:

```go
func ExecuteSystem(ctx context.Context, target chaoskit.Target) error {
    if err := chaoskit.MaybeError(ctx); err != nil {
        return err
    }

    // ... normal logic ...
    return nil
}
```

### Step 3.2: Interval-Based Delays

For more predictable chaos, use interval-based delays:

```go
// Delay injector that periodically blocks MaybeDelay() calls
Inject("interval-delay", injectors.RandomDelayWithInterval(
    20*time.Millisecond,  // min delay
    100*time.Millisecond, // max delay
    200*time.Millisecond, // interval between injections
))
```

### Step 3.3: Composite Injector

Combine multiple injectors into one:

```go
composite := injectors.Composite("chaos-combo",
    injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond),
    injectors.PanicProbability(0.02),
)

scenario := chaoskit.NewScenario("composite-chaos").
    WithTarget(system).
    Step("execute", ExecuteSystem).
    Inject("combo", composite).
    Repeat(50).
    Build()
```

---

## Part 4: Context-Based Chaos

Context-based chaos allows you to inject faults **inside** your code execution, not just before/after steps.

### Step 4.1: Instrument Your Code

Add chaos injection points in your code:

```go
func ProcessOrder(ctx context.Context, target chaoskit.Target) error {
    // Critical point: might panic here
    chaoskit.MaybePanic(ctx)
    
    // Propagate chaos-triggered errors (e.g., ErrorWithProbability injector)
    if err := chaoskit.MaybeError(ctx); err != nil {
        return err
    }

    // Network call: might have latency
    chaoskit.MaybeNetworkChaos(ctx, "api.example.com", 443)
    
    // Processing: might be delayed
    chaoskit.MaybeDelay(ctx)
    
    // Optional contextual chaos provider
    if chaoskit.ApplyChaos(ctx, "force-retry") {
        // Custom provider injected retry logic
    }
    
    // Your actual logic
    return processOrderLogic()
}
```

### Step 4.2: Configure Context Injectors

```go
scenario := chaoskit.NewScenario("context-chaos").
    WithTarget(system).
    Step("process-order", ProcessOrder).
    // These injectors provide chaos via context
    Inject("delay", injectors.RandomDelayWithInterval(
        10*time.Millisecond,
        30*time.Millisecond,
        50*time.Millisecond,
    )).
    Inject("panic", injectors.PanicProbability(0.05)). // 5% chance
    Inject("errors", injectors.ErrorWithProbability(io.ErrUnexpectedEOF, 0.02)).
    Repeat(20).
    Build()
```

> Note: import `io` alongside `time` to use `io.ErrUnexpectedEOF`.

### Step 4.3: Network Chaos via Context

For network-level chaos:

```go
// In your code
func MakeAPICall(ctx context.Context, host string, port int) error {
    // Check if network chaos should be applied
    chaoskit.MaybeNetworkChaos(ctx, host, port)
    
    // Your network call
    return httpCall(host, port)
}

// In scenario
networkInjector := injectors.NewContextualNetworkInjector(
    toxiproxyClient,
    proxyConfig,
    0.3, // 30% apply rate
)

scenario := chaoskit.NewScenario("network-chaos").
    WithTarget(system).
    Step("api-call", MakeAPICall).
    Inject("network", networkInjector).
    Build()
```

### Step 4.4: Context Cancellation

Test how your system handles context cancellation:

```go
func ProcessWithContext(ctx context.Context, target chaoskit.Target) error {
    // Create child context that might be cancelled
    childCtx, cancel := chaoskit.MaybeCancelContext(ctx)
    defer cancel()
    
    // Use childCtx in your logic
    return processWithContext(childCtx)
}

scenario := chaoskit.NewScenario("cancellation-chaos").
    WithTarget(system).
    Step("process", ProcessWithContext).
    Inject("cancellation", injectors.NewContextCancellationInjector(0.2)). // 20% chance
    Build()
```

---

## Part 5: Custom Injectors and Providers

Create your own injectors or universal chaos providers for custom behaviors.

### Step 5.1: Basic Custom Injector

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/rom8726/chaoskit"
)

type CustomDelayInjector struct {
    name     string
    duration time.Duration
    active   bool
}

func NewCustomDelayInjector(duration time.Duration) *CustomDelayInjector {
    return &CustomDelayInjector{
        name:     fmt.Sprintf("custom-delay-%v", duration),
        duration: duration,
    }
}

func (c *CustomDelayInjector) Name() string {
    return c.name
}

func (c *CustomDelayInjector) Inject(ctx context.Context) error {
    c.active = true
    return nil
}

func (c *CustomDelayInjector) Stop(ctx context.Context) error {
    c.active = false
    return nil
}

// Implement ChaosDelayProvider to work with MaybeDelay()
func (c *CustomDelayInjector) GetChaosDelay() (time.Duration, bool) {
    if c.active {
        return c.duration, true
    }
    return 0, false
}
```

### Step 5.2: Use Your Custom Injector

```go
customInjector := NewCustomDelayInjector(100 * time.Millisecond)

scenario := chaoskit.NewScenario("custom-injector").
    WithTarget(system).
    Step("execute", ExecuteSystem).
    Inject("custom", customInjector).
    Build()
```

### Step 5.3: Custom Validator

Create a validator for your specific invariants:

```go
type CustomStateValidator struct {
    name      string
    checkFunc func(ctx context.Context, target chaoskit.Target) error
}

func NewCustomStateValidator(
    name string,
    checkFunc func(ctx context.Context, target chaoskit.Target) error,
) *CustomStateValidator {
    return &CustomStateValidator{
        name:      name,
        checkFunc: checkFunc,
    }
}

func (c *CustomStateValidator) Name() string {
    return c.name
}

func (c *CustomStateValidator) Validate(ctx context.Context, target chaoskit.Target) error {
    return c.checkFunc(ctx, target)
}
```

Usage:

```go
scenario := chaoskit.NewScenario("custom-validator").
    WithTarget(system).
    Step("execute", ExecuteSystem).
    Assert("state", NewCustomStateValidator("state-check", func(ctx context.Context, target chaoskit.Target) error {
        // Your validation logic
        system := target.(*MySystem)
        if system.IsInValidState() {
            return nil
        }
        return fmt.Errorf("system in invalid state")
    })).
    Build()
```

### Step 5.4: Custom Chaos Provider

Create an injector that also implements `chaoskit.ChaosProvider` so you can trigger custom behavior via `ApplyChaos()`:

```go
type FeatureToggleProvider struct {
    name        string
    probability float64
}

func NewFeatureToggleProvider(name string, probability float64) *FeatureToggleProvider {
    return &FeatureToggleProvider{name: name, probability: probability}
}

// Injector contract (no background work required here)
func (p *FeatureToggleProvider) Name() string                 { return p.name }
func (p *FeatureToggleProvider) Inject(context.Context) error { return nil }
func (p *FeatureToggleProvider) Stop(context.Context) error   { return nil }

// ChaosProvider contract
func (p *FeatureToggleProvider) Apply(ctx context.Context) bool {
    return chaoskit.GetRand(ctx).Float64() < p.probability
}
```

Register the provider alongside other injectors:

```go
provider := NewFeatureToggleProvider("force-retry", 0.30)

scenario := chaoskit.NewScenario("toggle-chaos").
    WithTarget(system).
    Step("process", ProcessOrder).
    Inject("force-retry", provider).
    Build()
```

In your step, call `chaoskit.ApplyChaos(ctx, "force-retry")` to branch logic when the provider fires.

---

## Part 6: Production Usage Patterns

Best practices for using ChaosKit in production environments.

### Step 6.1: Structured Logging

Use structured logging for better observability:

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

executor := chaoskit.NewExecutor(
    chaoskit.WithSlogLogger(logger),
    chaoskit.WithFailurePolicy(chaoskit.ContinueOnFailure),
)
```

### Step 6.2: Continuous Testing

Run long-duration tests to discover edge cases:

```go
scenario := chaoskit.NewScenario("continuous-test").
    WithTarget(system).
    Step("execute", ExecuteSystem).
    Inject("delay", injectors.RandomDelay(10*time.Millisecond, 100*time.Millisecond)).
    Inject("panic", injectors.PanicProbability(0.01)).
    Assert("stability", validators.GoroutineLimit(500)).
    Assert("recursion", validators.RecursionDepthLimit(100)).
    RunFor(24 * time.Hour). // Run for 24 hours
    Build()

executor.Run(ctx, scenario)
```

### Step 6.3: Deterministic Testing

Use seeds for reproducible tests:

```go
scenario := chaoskit.NewScenario("deterministic-test").
    WithTarget(system).
    Step("execute", ExecuteSystem).
    Inject("delay", injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond)).
    WithSeed(12345). // Deterministic randomness
    Repeat(1000).
    Build()
```

### Step 6.4: Scoped Injectors

Organize injectors by system component:

```go
scenario := chaoskit.NewScenario("scoped-chaos").
    WithTarget(system).
    // Database injectors
    Scope("db", func(s *chaoskit.ScopeBuilder) {
        s.Inject("delay", injectors.RandomDelay(50*time.Millisecond, 200*time.Millisecond)).
          Inject("panic", injectors.PanicProbability(0.05))
    }).
    // API injectors
    Scope("api", func(s *chaoskit.ScopeBuilder) {
        s.Inject("delay", injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond))
    }).
    Step("execute", ExecuteSystem).
    Build()
```

### Step 6.5: Metrics and Reporting

Access metrics and generate reports:

```go
executor := chaoskit.NewExecutor()

if err := executor.Run(ctx, scenario); err != nil {
    log.Printf("Execution completed with errors: %v", err)
}

// Get metrics
metrics := executor.Metrics().Stats()
fmt.Printf("Metrics: %+v\n", metrics)

// Generate report
report := executor.Reporter().GenerateReport()
fmt.Println(report)

// Save JSON report
if err := executor.Reporter().SaveJSON("report.json"); err != nil {
    log.Printf("Failed to save report: %v", err)
}
```

### Step 6.6: Error Handling

Configure failure policies:

```go
// FailFast: Stop on first failure (default)
executor := chaoskit.NewExecutor(
    chaoskit.WithFailurePolicy(chaoskit.FailFast),
)

// ContinueOnFailure: Continue after failures, collect all errors
executor := chaoskit.NewExecutor(
    chaoskit.WithFailurePolicy(chaoskit.ContinueOnFailure),
)
```

---

## Part 7: Verdicts and Quality Gates

### Step 7.1: Evaluate a Verdict

```go
report, err := executor.Reporter().GetVerdict(chaoskit.DefaultThresholds())
if err != nil {
    log.Fatalf("failed to calculate verdict: %v", err)
}

fmt.Println(executor.Reporter().GenerateTextReport(report))

switch report.Verdict {
case chaoskit.VerdictPass:
    fmt.Println("System passed chaos verification")
case chaoskit.VerdictUnstable:
    fmt.Println("Scenario completed with warnings")
case chaoskit.VerdictFail:
    fmt.Println("Critical failures detected")
    os.Exit(report.Verdict.ExitCode())
}
```

### Step 7.2: Custom Thresholds

```go
thresholds := &chaoskit.SuccessThresholds{
    MinSuccessRate:              0.98,
    RequireAllValidatorsPassing: true,
    CriticalValidators: []string{
        chaoskit.ValidatorGoroutineLimit,
        chaoskit.ValidatorInfiniteLoop,
    },
    WarningValidators: []string{
        chaoskit.ValidatorExecutionTime,
    },
    MaxFailedIterations: 2,
}

report, err := executor.Reporter().GetVerdict(thresholds)
```

### Step 7.3: Persist Verdict Output

```go
jsonReport, err := executor.Reporter().GenerateJSON()
if err != nil {
    log.Fatalf("failed to build JSON report: %v", err)
}

if err := os.WriteFile("verdict.json", []byte(jsonReport), 0o644); err != nil {
    log.Fatalf("failed to write verdict: %v", err)
}
```

---

## Part 8: Metrics and Exporters

### Step 8.1: Attach Prometheus Exporter

```go
import (
    "net/http"

    "github.com/rom8726/chaoskit/exporters"
)

prom := exporters.NewPrometheusExporter("chaoskit", "experiments")

executor := chaoskit.NewExecutor(
    chaoskit.WithFailurePolicy(chaoskit.ContinueOnFailure),
    chaoskit.WithMetrics(chaoskit.NewMetricsCollector()),
)

http.Handle("/metrics", prom.Handler())
go http.ListenAndServe(":2112", nil) // Expose metrics endpoint
```

### Step 8.2: Forward Execution Data

The exporter is not coupled to the executor—pull metrics after each run:

```go
if err := executor.Run(ctx, scenario); err != nil {
    log.Printf("run failed: %v", err)
}

for _, result := range executor.Reporter().Results() {
    prom.RecordExecution(result)
}

stats := executor.Metrics().Stats()
if injectors, ok := stats["injector_metrics"].(map[string]map[string]interface{}); ok {
    for name, m := range injectors {
        prom.RecordInjectorMetrics(name, m)
    }
}
```

> Tip: For long-running executions the exporter can be updated periodically from a goroutine.

### Step 8.3: Inspect Metrics Snapshot

```text
$ curl http://localhost:2112/metrics
# HELP chaoskit_executions_total Total number of scenario executions
# TYPE chaoskit_executions_total counter
chaoskit_executions_total{scenario="workflow",result="success"} 42
chaoskit_executions_total{scenario="workflow",result="failure"} 3

# HELP chaoskit_success_rate Success rate of scenario executions (0-1)
chaoskit_success_rate{scenario="workflow"} 0.9333
```

---

## Part 9: Go Testing Integration

### Step 9.1: RunChaos Helper

```go
import (
    "testing"

    chaostesting "github.com/rom8726/chaoskit/testing"
)

func TestWorkflowChaos(t *testing.T) {
    target := &WorkflowEngine{}

    chaostesting.RunChaos(
        t,
        "workflow-chaos",
        target,
        func(b *chaoskit.ScenarioBuilder) *chaoskit.ScenarioBuilder {
            return b.
                Step("execute", ExecuteWorkflow).
                Inject("delay", injectors.RandomDelay(5*time.Millisecond, 25*time.Millisecond)).
                Assert("goroutines", validators.GoroutineLimit(200))
        },
        chaostesting.WithRepeat(25),
        chaostesting.WithDefaultThresholds(),
    )
}
```

### Step 9.2: Simple Variant

```go
chaostesting.RunChaosSimple(
    t,
    "simple-chaos",
    target,
    []func(context.Context, chaoskit.Target) error{ExecuteWorkflow},
    []chaoskit.Injector{injectors.PanicProbability(0.02)},
    []chaoskit.Validator{validators.NoPanics(0)},
    chaostesting.WithRelaxedThresholds(),
)
```

### Step 9.3: Diagnostic Options

```go
chaostesting.RunChaos(
    t,
    "diagnostics",
    target,
    builderFn,
    chaostesting.WithRepeat(50),
    chaostesting.WithFailurePolicy(chaoskit.ContinueOnFailure),
    chaostesting.WithThresholds(&chaoskit.SuccessThresholds{
        MinSuccessRate: 0.90,
        WarningValidators: []string{
            chaoskit.ValidatorExecutionTime,
        },
    }),
)
```

---

## Complete Example

Here's a complete example combining all concepts:

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "log/slog"
    "os"
    "time"
    
    "github.com/rom8726/chaoskit"
    "github.com/rom8726/chaoskit/injectors"
    "github.com/rom8726/chaoskit/validators"
)

type WorkflowEngine struct {
    name string
}

func (w *WorkflowEngine) Name() string { return w.name }
func (w *WorkflowEngine) Setup(ctx context.Context) error {
    fmt.Println("Setting up workflow engine...")
    return nil
}
func (w *WorkflowEngine) Teardown(ctx context.Context) error {
    fmt.Println("Tearing down workflow engine...")
    return nil
}

// ForceRetryProvider registers a chaos provider triggered via ApplyChaos.
type ForceRetryProvider struct {
    probability float64
}

func NewForceRetryProvider(probability float64) *ForceRetryProvider {
    return &ForceRetryProvider{probability: probability}
}

func (p *ForceRetryProvider) Name() string                 { return "force-retry" }
func (p *ForceRetryProvider) Inject(context.Context) error { return nil }
func (p *ForceRetryProvider) Stop(context.Context) error   { return nil }
func (p *ForceRetryProvider) Apply(ctx context.Context) bool {
    return chaoskit.GetRand(ctx).Float64() < p.probability
}

func ExecuteWorkflow(ctx context.Context, target chaoskit.Target) error {
    // Instrument your code with chaos points
    chaoskit.MaybePanic(ctx)
    chaoskit.MaybeDelay(ctx)
    
    if err := chaoskit.MaybeError(ctx); err != nil {
        chaoskit.RecordError(ctx)
        return err
    }
    
    // Context cancellation chaos and provider-triggered behavior
    childCtx, cancel := chaoskit.MaybeCancelContext(ctx)
    defer cancel()
    
    if chaoskit.ApplyChaos(childCtx, "force-retry") {
        log.Println("retry path activated by chaos provider")
    }
    
    // Record events for validators
    chaoskit.RecordRecursionDepth(childCtx, 0)
    
    // Your workflow logic
    time.Sleep(10 * time.Millisecond)
    
    return nil
}

func main() {
    // Setup structured logging
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    
    engine := &WorkflowEngine{name: "production-engine"}
    
    // Build comprehensive scenario
    scenario := chaoskit.NewScenario("production-test").
        WithTarget(engine).
        Step("execute", ExecuteWorkflow).
        Inject("delay", injectors.RandomDelay(5*time.Millisecond, 25*time.Millisecond)).
        Inject("panic", injectors.PanicProbability(0.01)).
        Inject("errors", injectors.ErrorWithProbability(io.ErrUnexpectedEOF.Error(), 0.02)).
        Inject("cancellation", injectors.NewContextCancellationInjector(0.15)).
        Assert("goroutines", validators.GoroutineLimit(200)).
        Assert("recursion", validators.RecursionDepthLimit(100)).
        Assert("no-infinite-loop", validators.NoSlowIteration(5*time.Second)).
        Assert("max-errors", validators.MaxErrors(50)).
        Repeat(500).
        Build()
    
    // Create executor with options
    executor := chaoskit.NewExecutor(
        chaoskit.WithSlogLogger(logger),
        chaoskit.WithFailurePolicy(chaoskit.ContinueOnFailure),
    )
    
    // Run scenario
    ctx := context.Background()
    if err := executor.Run(ctx, scenario); err != nil {
        log.Printf("Scenario completed with errors: %v", err)
    }
    
    // Summaries
    fmt.Println(executor.Reporter().GenerateReport())
    fmt.Printf("Metrics: %+v\n", executor.Metrics().Stats())
    
    // Verdict with strict thresholds
    if report, err := executor.Reporter().GetVerdict(chaoskit.StrictThresholds()); err == nil {
        fmt.Println(executor.Reporter().GenerateTextReport(report))
        os.Exit(report.Verdict.ExitCode())
    }
}
```

---

## Next Steps

- Explore the [examples/](examples/) directory for more patterns
- Read [ARCHITECTURE.md](ARCHITECTURE.md) for design details
- Check [README.md](README.md) for API reference

---

**Happy Chaos Testing!**
