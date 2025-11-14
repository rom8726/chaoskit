# ChaosKit Tutorial

This tutorial provides a progressive guide to using ChaosKit, from basic scenarios to advanced patterns.

## Table of Contents

1. [Part 1: Basic Scenario](#part-1-basic-scenario)
2. [Part 2: Adding Validators](#part-2-adding-validators)
3. [Part 3: Multiple Injectors](#part-3-multiple-injectors)
4. [Part 4: Context-Based Chaos](#part-4-context-based-chaos)
5. [Part 5: Custom Injectors](#part-5-custom-injectors)
6. [Part 6: Production Usage](#part-6-production-usage)

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

Validators check that your system maintains invariants during chaos testing.

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

Some validators need you to record events in your code:

```go
func ExecuteSystem(ctx context.Context, target chaoskit.Target) error {
    // Record recursion depth for RecursionDepthValidator
    chaoskit.RecordRecursionDepth(ctx, currentDepth)
    
    // Your logic that might recurse
    if shouldRecurse {
        return ExecuteSystem(ctx, target) // Recursive call
    }
    
    return nil
}
```

### Step 2.3: Multiple Validators

Add multiple validators to check different invariants:

```go
scenario := chaoskit.NewScenario("comprehensive-validation").
    WithTarget(system).
    Step("execute", ExecuteSystem).
    Assert("goroutines", validators.GoroutineLimit(200)).
    Assert("recursion", validators.RecursionDepthLimit(50)).
    Assert("no-infinite-loop", validators.NoSlowIteration(5*time.Second)).
    Assert("memory", validators.MemoryUnderLimit(512*1024*1024)). // 512MB
    Repeat(100).
    Build()
```

**What happens:**
- If any validator fails, the scenario stops (or continues, depending on failure policy)
- Validators log warnings when approaching limits (80% threshold)
- All validation results are included in the final report

---

## Part 3: Multiple Injectors

Injectors introduce faults into your system. Let's add multiple types of chaos.

### Step 3.1: Basic Injectors

```go
import "github.com/rom8726/chaoskit/injectors"

scenario := chaoskit.NewScenario("multiple-injectors").
    WithTarget(system).
    Step("execute", ExecuteSystem).
    // Delay injection: random delays between 10-50ms
    Inject("delay", injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond)).
    // Panic injection: 1% chance of panic
    Inject("panic", injectors.PanicProbability(0.01)).
    Assert("no-panics", validators.NoPanics(5)). // Allow up to 5 panics
    Repeat(100).
    Build()
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
    
    // Network call: might have latency
    chaoskit.MaybeNetworkChaos(ctx, "api.example.com", 443)
    
    // Processing: might be delayed
    chaoskit.MaybeDelay(ctx)
    
    // Another critical point
    chaoskit.MaybePanic(ctx)
    
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
    Repeat(20).
    Build()
```

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

## Part 5: Custom Injectors

Create your own injectors for custom chaos behaviors.

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

---

## Part 6: Production Usage

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

## Complete Example

Here's a complete example combining all concepts:

```go
package main

import (
    "context"
    "fmt"
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

func ExecuteWorkflow(ctx context.Context, target chaoskit.Target) error {
    // Instrument your code with chaos points
    chaoskit.MaybePanic(ctx)
    chaoskit.MaybeDelay(ctx)
    
    // Record events for validators
    chaoskit.RecordRecursionDepth(ctx, 1)
    
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
        Assert("goroutines", validators.GoroutineLimit(200)).
        Assert("recursion", validators.RecursionDepthLimit(100)).
        Assert("no-infinite-loop", validators.NoSlowIteration(5*time.Second)).
        Repeat(1000).
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
    
    // Print results
    fmt.Println(executor.Reporter().GenerateReport())
    fmt.Printf("Metrics: %+v\n", executor.Metrics().Stats())
}
```

---

## Next Steps

- Explore the [examples/](examples/) directory for more patterns
- Read [ARCHITECTURE.md](ARCHITECTURE.md) for design details
- Check [README.md](README.md) for API reference
- Review [TECHNICAL_SPECIFICATION.md](TECHNICAL_SPECIFICATION.md) for implementation details

---

**Happy Chaos Testing!** 🎲

