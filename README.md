# ChaosKit

[![Go Report Card](https://goreportcard.com/badge/github.com/rom8726/chaoskit)](https://goreportcard.com/report/github.com/rom8726/chaoskit)
[![Go Reference](https://pkg.go.dev/badge/github.com/rom8726/chaoskit.svg)](https://pkg.go.dev/github.com/rom8726/chaoskit)

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

### Using go get

```bash
go get github.com/rom8726/chaoskit
```

### Using go.mod

```go
require github.com/rom8726/chaoskit v1.0.0
```

### Prerequisites

- Go 1.21 or later (for structured logging with `slog`)
- For monkey patching: Build with `-gcflags=all=-l`
- For gofail: Build with `-tags failpoint`
- For ToxiProxy: ToxiProxy server running (optional)

## When to Use ChaosKit

ChaosKit is ideal for:

- **Workflow Engine Testing**: Test saga orchestrators, state machines, and workflow engines
- **Rollback Testing**: Verify bounded recursion depth in compensation handlers
- **Resource Leak Detection**: Find goroutine leaks, memory leaks, and resource exhaustion
- **Network Resilience**: Test system behavior under network failures
- **Error Recovery**: Validate error handling and recovery mechanisms
- **Performance Testing**: Ensure system maintains performance under chaos
- **Continuous Reliability**: Long-duration testing to discover edge cases

**Not suitable for:**
- Unit testing (use standard Go testing)
- Integration testing without chaos requirements
- Production monitoring (use observability tools)

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
import "log/slog"

// Structured logging
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

executor := chaoskit.NewExecutor(
    chaoskit.WithSlogLogger(logger),          // Structured logger (recommended)
    chaoskit.WithLogLevel(slog.LevelDebug),   // Set log level
    chaoskit.WithJSONLogging(),               // JSON output format
    chaoskit.WithFailurePolicy(policy),       // Error handling strategy
    chaoskit.WithMetrics(metricsCollector),   // Custom metrics
    chaoskit.WithReporter(reporter),          // Custom reporter
)
```

### Failure Policies

**FailFast**: Stops execution on first failure (default behavior)

**ContinueOnFailure**: Continues execution after failures, collecting all errors

### Logging Configuration

ChaosKit uses structured logging with `slog`:

```go
// Text format (default)
logger := slog.Default()

// JSON format
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// Custom handler
logger := slog.New(customHandler)

executor := chaoskit.NewExecutor(
    chaoskit.WithSlogLogger(logger),
)
```

Log levels:
- **Debug**: Verbose operation details (disabled in production by default)
- **Info**: Normal operations (default)
- **Warn**: Recoverable issues
- **Error**: Failures that need attention

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
make test   # Run tests (with race detector)
make race   # Run race detector tests
make fmt    # Format code
make vet    # Run static analysis
make clean  # Remove build artifacts
make help   # Show all available targets
```

## Roadmap

Future enhancements (not in current scope):

- Prometheus metrics integration
- HTML report generation
- Configuration file support (YAML/JSON)
- Web UI dashboard
- OpenTelemetry integration
- Distributed chaos coordination

See [TECHNICAL_SPECIFICATION.md](TECHNICAL_SPECIFICATION.md) for detailed roadmap.

## Best Practices

1. **Start Simple**: Begin with basic scenarios and gradually increase complexity
2. **Match Validators**: Select validators that match your system's invariants
3. **Monitor Metrics**: Track success rates and performance during testing
4. **Multiple Iterations**: Run many iterations to catch intermittent issues
5. **Continuous Testing**: Use long-duration tests for edge case discovery
6. **Record Events**: Call `RecordRecursionDepth()` and `RecordPanic()` in your code
7. **Structured Logging**: Use JSON logging in production for better observability
8. **Deterministic Seeds**: Use `WithSeed()` for reproducible tests
9. **Scoped Injectors**: Organize injectors by system component using scopes
10. **Resource Limits**: Set appropriate limits in validators based on your system

## Production Checklist

Before running chaos tests in production-like environments:

- [ ] Set appropriate validator thresholds based on baseline metrics
- [ ] Use structured logging (JSON format) for log aggregation
- [ ] Configure failure policy (FailFast vs ContinueOnFailure)
- [ ] Set deterministic seeds for reproducible tests
- [ ] Monitor resource usage (CPU, memory, goroutines)
- [ ] Review log levels (Debug in dev, Info in production)
- [ ] Test with low chaos probabilities first, then increase
- [ ] Have rollback plan if tests affect production systems
- [ ] Use scoped injectors to isolate chaos to specific components
- [ ] Review metrics and reports after test completion

## Documentation

- **[TUTORIAL.md](TUTORIAL.md)** - Progressive tutorial from basics to advanced
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Detailed architecture with diagrams
- **[TECHNICAL_SPECIFICATION.md](TECHNICAL_SPECIFICATION.md)** - Technical specification
- [examples/README.md](examples/README.md) - Example documentation
- [GoDoc](https://pkg.go.dev/github.com/rom8726/chaoskit) - API documentation

## Comparison with Other Tools

| Feature | ChaosKit | Chaos Monkey | Litmus | Gremlin |
|---------|----------|-------------|--------|---------|
| Language | Go | Java | Kubernetes | Multi |
| Focus | Libraries/Services | AWS | Kubernetes | Infrastructure |
| Code Instrumentation | ✅ | ❌ | ❌ | ❌ |
| Custom Injectors | ✅ | Limited | Limited | Limited |
| Validators | ✅ | ❌ | ❌ | ❌ |
| Structured Logging | ✅ | ❌ | ❌ | ❌ |
| Context-Based Chaos | ✅ | ❌ | ❌ | ❌ |

**ChaosKit is best for:**
- Testing Go libraries and services
- Workflow engine reliability
- Code-level fault injection
- Custom chaos scenarios
- Development and CI/CD pipelines

## FAQ

### Q: Can I use ChaosKit in production?

A: Yes, but with caution. Use low chaos probabilities and monitor closely. Never use monkey patching in production.

### Q: Does ChaosKit affect production systems?

A: Only if you configure it to. By default, chaos is isolated to your test scenarios. Network injectors require explicit proxy setup.

### Q: How do I test distributed systems?

A: Use network injectors (ToxiProxy) or context-based network chaos. Each service can run its own chaos scenarios.

### Q: Can I create custom injectors?

A: Yes! Implement the `Injector` interface. See [TUTORIAL.md](TUTORIAL.md) Part 5 for examples.

### Q: How do I debug failing scenarios?

A: Enable debug logging: `WithLogLevel(slog.LevelDebug)`. Review structured logs for detailed execution flow.

### Q: What's the difference between injectors and validators?

A: **Injectors** introduce faults (delays, panics, etc.). **Validators** verify invariants (no leaks, bounded recursion, etc.).

### Q: How do I test rollback mechanisms?

A: Use `RecordRecursionDepth()` in your rollback code and add `RecursionDepthLimit` validator. See examples for workflow engines.

### Q: Can I use ChaosKit with existing test frameworks?

A: Yes! ChaosKit scenarios can be run from Go tests. See `examples/testing_example/` for integration examples.

## Performance Considerations

- **Lock Granularity**: Fine-grained locks minimize contention
- **Atomic Operations**: Counters use atomic operations for performance
- **Context Immutability**: Context values are safe for concurrent access
- **Copy-on-Read**: Maps copied when needed for thread safety
- **Lazy Initialization**: Validators initialize baselines on first use

Expected overhead:
- Lock-protected operations: < 5% performance impact
- Context-based chaos: Negligible overhead
- Structured logging: Minimal impact (can be disabled in production)

## Security Considerations

- **Monkey Patching**: Only for testing, requires explicit build flags (`-gcflags=all=-l`)
- **Network Proxies**: Requires explicit setup, not enabled by default
- **Context Isolation**: Each scenario has isolated context
- **Resource Limits**: Validators enforce resource limits to prevent DoS

**Never use monkey patching in production code.**

## Troubleshooting

### Scenario fails immediately

**Problem**: Scenario fails right after starting.

**Solutions**:
- Verify that your `Target` implements all required methods correctly
- Ensure `Setup()` returns `nil` on success
- Check that all injectors are properly configured
- Review logs for specific error messages

### Validators consistently fail

**Problem**: Validators fail on every iteration.

**Solutions**:
- Check that validator thresholds are appropriate for your system
- Ensure you're recording events (`RecordRecursionDepth`, `RecordPanic`) where needed
- Review baseline metrics to set correct limits
- Use `ContinueOnFailure` policy to see all failures

### High failure rate

**Problem**: Many iterations fail.

**Solutions**:
- This may be expected with high chaos injection rates
- Adjust injector probabilities (start with low values like 0.01)
- Verify if failures indicate legitimate system issues
- Review validator thresholds - they may be too strict

### No chaos being injected

**Problem**: Chaos injectors don't seem to be working.

**Solutions**:
- For context-based chaos: Ensure you call `MaybeDelay()`, `MaybePanic()` in your code
- Check injector probabilities (0.01 = 1% chance)
- Verify injectors are started (check logs for "injector started" messages)
- For monkey patching: Ensure you build with `-gcflags=all=-l`

### Race conditions detected

**Problem**: `go test -race` reports race conditions.

**Solutions**:
- All known race conditions have been fixed in the latest version
- Run `make race` to verify
- Report any new race conditions as issues

### Logging not working

**Problem**: No logs appearing.

**Solutions**:
- Check log level (Debug logs are disabled by default in production)
- Verify logger is configured: `WithSlogLogger(logger)`
- For JSON logging: Use `WithJSONLogging()`
- Check that log handler is writing to correct output

## Contributing

Contributions are welcome! Please see our contributing guidelines:

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Make your changes**: Follow Go code style and add tests
4. **Run tests**: `make test` and `make race`
5. **Commit changes**: Use clear commit messages
6. **Push to branch**: `git push origin feature/amazing-feature`
7. **Open a Pull Request**: Describe your changes clearly

### Development Setup

```bash
# Clone repository
git clone https://github.com/rom8726/chaoskit.git
cd chaoskit

# Run tests
make test

# Run race detector
make race

# Format code
make fmt

# Run static analysis
make vet
```

### Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` for formatting
- Add godoc comments for exported types and functions
- Write tests for new features
- Use structured logging with `slog`

## License

Apache-2.0

## Acknowledgments

Inspired by chaos engineering principles and the need for robust testing of distributed systems and workflow engines.
