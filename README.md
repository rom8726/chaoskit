# ChaosKit

A modular Go framework for chaos engineering, fault injection, and reliability testing of distributed systems, libraries, and services.

## Overview

ChaosKit enables systematic testing of system reliability and resilience through controlled fault injection and invariant validation. The framework is designed to detect issues that traditional unit and integration tests often miss, such as infinite rollback loops, goroutine leaks, and unbounded recursion in saga orchestrators and workflow engines.

## Key Capabilities

- **Controlled Chaos Injection**: Introduce delays, panics, resource pressure, and network faults
- **Multiple Injection Methods**: Context-based, monkey patching, failpoints, and network proxies
- **Invariant Validation**: Verify system properties like bounded recursion, absence of infinite loops, and resource leak prevention
- **Continuous Testing**: Run long-duration stress tests to discover edge cases
- **Comprehensive Metrics**: Automatic collection of execution statistics and performance data
- **Flexible Configuration**: Customize behavior through functional options pattern

## Latest Features

**Monkey Patching**: Runtime function patching without code modification
- Inject panics and delays into arbitrary functions
- Requires `-gcflags=all=-l` build flag

**ToxiProxy Integration**: Network-level chaos for database and service connections
- Latency, bandwidth, timeout, and packet slicing
- Optional proxy setup for database connections

**Gofail Support**: Failpoint-based injection for production-safe testing
- Compile-time injection points via `failpoint.Inject`
- Requires `-tags failpoint` build flag

**Context-Based Chaos**: Inject chaos directly in your code
- Use `chaoskit.MaybePanic(ctx)`, `chaoskit.MaybeDelay(ctx)`, `chaoskit.MaybeNetworkChaos(ctx, host, port)`
- Enables fine-grained control over chaos timing

## Installation

```bash
go get github.com/rom8726/chaoskit
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/rom8726/chaoskit"
    "github.com/rom8726/chaoskit/injectors"
    "github.com/rom8726/chaoskit/validators"
)

// Define your system under test
type WorkflowEngine struct{}

func (w *WorkflowEngine) Name() string { return "workflow-engine" }
func (w *WorkflowEngine) Setup(ctx context.Context) error { return nil }
func (w *WorkflowEngine) Teardown(ctx context.Context) error { return nil }

// Define execution step with context-based chaos
func ExecuteWorkflow(ctx context.Context, target chaoskit.Target) error {
    // Inject chaos directly in your code
    if chaoskit.MaybePanic(ctx) {
        panic("chaos: intentional panic")
    }
    
    chaoskit.MaybeDelay(ctx) // May add delay here
    
    // Your workflow execution logic
    time.Sleep(10 * time.Millisecond)
    return nil
}

func main() {
    engine := &WorkflowEngine{}

    scenario := chaoskit.NewScenario("reliability-test").
        WithTarget(engine).
        Step("execute-workflow", ExecuteWorkflow).
        Inject("delay", injectors.RandomDelay(5*time.Millisecond, 25*time.Millisecond)).
        Inject("panic", injectors.PanicProbability(0.01)).
        Assert("goroutine-limit", validators.GoroutineLimit(200)).
        Assert("recursion-depth", validators.RecursionDepthLimit(100)).
        Assert("no-infinite-loop", validators.NoInfiniteLoop(5*time.Second)).
        Repeat(100).
        Build()

    if err := chaoskit.Run(context.Background(), scenario); err != nil {
        log.Fatalf("Scenario execution failed: %v", err)
    }
}
```

## Core Components

### Chaos Injectors

**Basic Injectors**:
- **DelayInjector**: Random latency (probability-based or interval-based modes)
- **PanicInjector**: Random panics to test recovery mechanisms
- **CPUInjector**: CPU stress under load
- **MemoryInjector**: Memory pressure simulation

**Network Injectors**:
- **ToxiProxy Injectors**: Network-level chaos (latency, bandwidth, timeout, packet slicing)
  - `ToxiProxyLatency`: Add network delays with jitter
  - `ToxiProxyBandwidth`: Limit transfer speeds
  - `ToxiProxyTimeout`: Connection timeouts
  - `ToxiProxySlicer`: Packet loss simulation
- **ContextualNetworkInjector**: Per-request network chaos via context

**Advanced Injectors**:
- **MonkeyPatchPanicInjector**: Runtime function patching for panic injection
- **MonkeyPatchDelayInjector**: Runtime function patching for delay injection
- **GofailPanicInjector**: Failpoint-based panic injection (requires `-tags failpoint`)

**CompositeInjector**: Combines multiple injectors for complex failure scenarios

### Validators

**PanicRecoveryValidator**: Ensures proper panic recovery and error handling

**RecursionDepthValidator**: Validates bounded recursion depth (critical for rollback testing)

**GoroutineLeakValidator**: Detects goroutine leaks and resource exhaustion

**InfinityLoopValidator**: Identifies infinite loops and stuck executions

**ExecutionTimeValidator**: Validates performance within specified bounds

**MemoryLimitValidator**: Monitors memory usage against defined thresholds

**StateConsistencyValidator**: Enables custom state validation logic

**CompositeValidator**: Combines multiple validators for comprehensive checks

### Metrics and Reporting

- Automatic collection of execution statistics
- JSON and text report generation
- Success rate and duration tracking
- Extensible metrics collection interface

## Usage Patterns

### Basic Reliability Testing

```go
scenario := chaoskit.NewScenario("basic-test").
    WithTarget(workflowEngine).
    Step("execute", ExecuteWorkflow).
    Assert("no-leaks", validators.GoroutineLimit(100)).
    Repeat(1000).
    Build()

chaoskit.Run(ctx, scenario)
```

### Continuous Testing

```go
executor := chaoskit.NewExecutor(
    chaoskit.WithFailurePolicy(chaoskit.ContinueOnFailure),
    chaoskit.WithLogger(customLogger),
)

scenario := chaoskit.NewScenario("continuous-test").
    WithTarget(engine).
    Step("execute", ExecuteWorkflow).
    Inject("chaos", injectors.CompositeInjector(
        injectors.RandomDelay(10*time.Millisecond, 100*time.Millisecond),
        injectors.PanicProbability(0.05),
    )).
    Assert("stability", validators.GoroutineLimit(500)).
    RunFor(24*time.Hour).
    Build()

executor.Run(ctx, scenario)
```

### Load Testing with Validation

```go
scenario := chaoskit.NewScenario("load-test").
    WithTarget(engine).
    Step("execute", ExecuteWorkflow).
    Inject("cpu-stress", injectors.NewCPUInjector(4, 500*time.Millisecond)).
    Assert("performance", validators.NewExecutionTimeValidator(100*time.Millisecond)).
    Assert("memory", validators.NewMemoryLimitValidator(512*1024*1024)).
    Repeat(10000).
    Build()

chaoskit.Run(ctx, scenario)
```

## Configuration Options

The framework supports flexible configuration through functional options:

```go
executor := chaoskit.NewExecutor(
    chaoskit.WithLogger(logger),              // Custom logger
    chaoskit.WithFailurePolicy(policy),       // Error handling strategy
    chaoskit.WithMetrics(metricsCollector),   // Custom metrics
    chaoskit.WithReporter(reporter),          // Custom reporter
)
```

### Failure Policies

**FailFast**: Stops execution on first failure (default behavior)

**ContinueOnFailure**: Continues execution after failures, collecting all errors

## Event Recording

ChaosKit provides context-based event recording for tracking runtime behavior:

```go
func rollbackFunction(ctx context.Context, depth int) {
    // Record recursion depth for validator analysis
    chaoskit.RecordRecursionDepth(ctx, depth)
    
    // Rollback logic
    if depth > 0 {
        rollbackFunction(ctx, depth-1)
    }
}
```

Validators implementing `RecursionRecorder` automatically receive these events for analysis.

## Examples

The `examples/` directory contains complete working demonstrations:

**simple**: Basic chaos testing with fixed iterations

**continuous**: Continuous testing with multiple workflow patterns and real-time statistics

**chaos_context**: Context-based chaos injection using `MaybePanic()` and `MaybeDelay()`

**monkey_patch**: Monkey patching injection (panic and delay) - requires `-gcflags=all=-l`

**toxiproxy**: Network chaos testing with ToxiProxy (latency, bandwidth, timeouts)

**floxy_stress_test**: Comprehensive stress test with monkey patching, gofail, and ToxiProxy

Build and run examples:

```bash
make build          # Build all examples
make run-simple     # Run simple example
make run-continuous # Run continuous testing

# Advanced examples
go run -gcflags=all=-l examples/monkey_patch/main.go
USE_TOXIPROXY=true go run -gcflags=all=-l examples/toxiproxy/main.go
```

## Architecture

ChaosKit follows clean architecture principles with clear separation of concerns:

**Target**: System under test interface

**Injector**: Fault injection mechanism

**Validator**: Invariant verification

**Executor**: Orchestration and lifecycle management

**Reporter**: Results aggregation and reporting

**MetricsCollector**: Performance data collection

For detailed architectural information, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Building

```bash
make build  # Build examples
make test   # Run tests
make fmt    # Format code
make vet    # Run static analysis
make clean  # Remove build artifacts
```

## Best Practices

1. Start with simple scenarios and gradually increase complexity
2. Select validators that match your system's invariants
3. Monitor metrics and success rates during testing
4. Run multiple iterations to catch intermittent issues
5. Use continuous testing for long-duration validation
6. Record events (recursion depth, panics) in your code for validator analysis

## Documentation

- [chaoskit.md](chaoskit.md) - Original design specification
- [examples/README.md](examples/README.md) - Example documentation

## Troubleshooting

**Scenario fails immediately**

Verify that your Target implements all required methods correctly and that Setup() returns nil on success.

**Validators consistently fail**

Check that validator thresholds are appropriate for your system and ensure you're recording events (RecordRecursionDepth, etc.) where needed.

**High failure rate**

This may be expected with high chaos injection rates. Adjust injector probabilities or verify if failures indicate legitimate system issues.

## Contributing

Contributions are welcome. Please submit issues and pull requests through the standard GitHub workflow.

## License

Apache-2.0

## Acknowledgments

Inspired by chaos engineering principles and the need for robust testing of distributed systems and workflow engines.
