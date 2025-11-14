# ChaosKit

[![Go Report Card](https://goreportcard.com/badge/github.com/rom8726/chaoskit)](https://goreportcard.com/report/github.com/rom8726/chaoskit)
[![Go Reference](https://pkg.go.dev/badge/github.com/rom8726/chaoskit.svg)](https://pkg.go.dev/github.com/rom8726/chaoskit)
[![Coverage Status](https://coveralls.io/repos/github/rom8726/chaoskit/badge.svg?branch=main)](https://coveralls.io/github/rom8726/chaoskit?branch=main)

A modular Go framework for chaos engineering, fault injection, and reliability testing of distributed systems, libraries, and services.

<img src="docs/chaoskit_logo.png" alt="ChaosKit Logo" width="500" />

## Table of Contents

- [Overview](#overview)
- [Key Capabilities](#key-capabilities)
- [Latest Features](#latest-features)
- [Installation](#installation)
- [When to Use ChaosKit](#when-to-use-chaoskit)
- [Quick Start](#quick-start)
- [Injection Methods: Capabilities and Limitations](#injection-methods-capabilities-and-limitations)
  - [1. Context-Based Injection (Recommended for New Code)](#1-context-based-injection-recommended-for-new-code)
  - [2. Failpoint Injection (Recommended for Production)](#2-failpoint-injection-recommended-for-production)
  - [3. ToxiProxy (Recommended for Network Chaos)](#3-toxiproxy-recommended-for-network-chaos)
  - [4. Monkey Patching (⚠️ Limited Use Cases)](#4-monkey-patching--limited-use-cases)
  - [Summary Table](#summary-table)
  - [Choosing the Right Method](#choosing-the-right-method)
- [Core Components](#core-components)
  - [Chaos Injectors](#chaos-injectors)
  - [Validators](#validators)
  - [Metrics and Reporting](#metrics-and-reporting)
- [Usage Patterns](#usage-patterns)
  - [Basic Reliability Testing](#basic-reliability-testing)
  - [Continuous Testing](#continuous-testing)
  - [Load Testing with Validation](#load-testing-with-validation)
- [Configuration Options](#configuration-options)
  - [Failure Policies](#failure-policies)
  - [Logging Configuration](#logging-configuration)
- [Event Recording](#event-recording)
- [Examples](#examples)
- [Architecture](#architecture)
- [Building](#building)
- [Roadmap](#roadmap)
- [Best Practices](#best-practices)
- [Documentation](#documentation)
- [Comparison with Other Tools](#comparison-with-other-tools)
- [FAQ](#faq)
- [Performance Considerations](#performance-considerations)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)

## Overview

ChaosKit enables systematic testing of system reliability and resilience through controlled fault injection and invariant validation.
The framework is designed to detect issues that traditional unit and integration tests often miss, such as infinite rollback loops,
goroutine leaks, and unbounded recursion in saga orchestrators and workflow engines.

**⚠️ Important**: ChaosKit is designed for **proactive chaos engineering** - it works best when integrated into your code from the start.
Most injection methods require adding chaos hooks to your code (except ToxiProxy for network chaos).
This makes ChaosKit ideal for new projects or when you can modify the code being tested.

**Key Philosophy**: Build resilient systems by designing them to be testable with chaos from day one, rather than retrofitting chaos testing into existing code.

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

**Failpoint Support**: Failpoint-based injection for production-safe testing
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

- Go 1.25 or later
- **Important**: Most chaos injection requires **modifying your code** (see "Injection Methods" section)
- For monkey patching: Build with `-gcflags=all=-l` (not recommended, see limitations)
- For failpoint: Build with `-tags failpoint` and instrument code with `failpoint.Inject()`
- For ToxiProxy: ToxiProxy server running (this is the only method that works without code changes)

## When to Use ChaosKit

### Ideal Use Cases

ChaosKit is **best suited for**:

- **New Libraries and Frameworks**: Projects designed from scratch with chaos testing in mind
- **Workflow Engine Testing**: Test saga orchestrators, state machines, and workflow engines
- **Rollback Testing**: Verify bounded recursion depth in compensation handlers
- **Resource Leak Detection**: Find goroutine leaks, memory leaks, and resource exhaustion
- **Network Resilience**: Test system behavior under network failures (via ToxiProxy)
- **Error Recovery**: Validate error handling and recovery mechanisms
- **Continuous Reliability**: Long-duration testing to discover edge cases

### ⚠️ Important Limitations

**Code Instrumentation Required**: Most injection methods (context-based, monkey patching, failpoints) require **modifying your code** to add chaos hooks. This means:

- ✅ **Works great**: New projects designed with ChaosKit integration
- ✅ **Works great**: Network operations (ToxiProxy works without code changes)
- ⚠️ **Limited support**: Existing libraries require code modifications or wrappers
- ⚠️ **Limited support**: Monkey patching only works with specific code patterns
- ❌ **Won't work**: Production-optimized code with inlining enabled
- ❌ **Won't work**: Third-party libraries you cannot modify

### ❌ Not Suitable For

- **Testing existing libraries "as-is"**: Without adding chaos hooks or using ToxiProxy
- **Unit testing**: Use standard Go testing instead
- **Infrastructure-level chaos**: Use Chaos Mesh, Litmus, or Gremlin instead
- **Production monitoring**: Use observability tools like Prometheus/Grafana
- **Black-box testing**: When you cannot modify the tested code

### 💡 Recommended Approach

**For New Projects**: Design your code with chaos testing from the start by:
- Adding `chaoskit.MaybeX(ctx)` calls at critical points
- Using `failpoint.Inject()` for production-safe chaos points
- Passing context through your entire call chain

**For Existing Projects**: Consider:
- ToxiProxy for network-level chaos (no code changes needed)
- Creating chaos-aware wrappers around existing components
- Gradually adding failpoints to critical paths
- Forking libraries to add chaos instrumentation

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

## Injection Methods: Capabilities and Limitations

ChaosKit provides multiple injection methods, each with different trade-offs:

### 1. Context-Based Injection (Recommended for New Code)

**How it works**: Explicitly call chaos functions in your code.

```go
func ProcessOrder(ctx context.Context, order Order) error {
    chaoskit.MaybePanic(ctx)   // Inject panic with configured probability
    chaoskit.MaybeDelay(ctx)   // Inject delay with configured duration
    
    // Your business logic
}
```

**Capabilities**:
- ✅ Fine-grained control over injection points
- ✅ Works in production (controlled by context)
- ✅ Type-safe and explicit
- ✅ Low overhead

**Limitations**:
- ❌ Requires code modification
- ❌ Cannot test existing code without changes
- ❌ Need to identify and instrument all critical paths

**Best for**: New projects, microservices, workflow engines

---

### 2. Failpoint Injection (Recommended for Production)

**How it works**: Add compile-time injection points.

```go
import "github.com/pingcap/failpoint"

func SaveData(data Data) error {
    failpoint.Inject("save-error", func() {
        return errors.New("injected error")
    })
    
    // Your business logic
}
```

**Capabilities**:
- ✅ Production-safe (compiles to no-op without `-tags failpoint`)
- ✅ Used by production systems (etcd, TiDB)
- ✅ No runtime overhead in production builds
- ✅ Explicit injection points

**Limitations**:
- ❌ Requires code modification
- ❌ Need separate build for chaos testing (`-tags failpoint`)
- ❌ Cannot test existing code without adding failpoints

**Best for**: Production systems, critical infrastructure, databases

---

### 3. ToxiProxy (Recommended for Network Chaos)

**How it works**: Proxy network connections through ToxiProxy.

```go
// No code changes needed!
// Just change connection string from:
db, _ := sql.Open("postgres", "localhost:5432")

// To proxied connection:
db, _ := sql.Open("postgres", "localhost:25432")  // ToxiProxy listens here

// Configure chaos via ChaosKit:
toxiProxy := injectors.ToxiProxyLatency("db-proxy", "localhost:5432", 100*time.Millisecond)
```

**Capabilities**:
- ✅ **No code changes required**
- ✅ Works with any language/library
- ✅ Real network conditions (latency, bandwidth, packet loss)
- ✅ Production-ready tool

**Limitations**:
- ⚠️ Only works for network operations
- ⚠️ Requires ToxiProxy server infrastructure
- ⚠️ Cannot inject application-level failures (panics, logic errors)

**Best for**: Database chaos, HTTP services, gRPC, any network I/O

---

### 4. Monkey Patching (⚠️ Limited Use Cases)

**How it works**: Runtime function replacement via reflection.

```go
// Your code must be structured like this:
var ProcessFunc = func() error {  // Must be package-level var
    return nil
}

// ChaosKit can patch it:
injector := injectors.MonkeyPatchPanic([]PatchTarget{
    {Func: &ProcessFunc, Probability: 0.1},
})
```

**Capabilities**:
- ⚠️ No code changes *if* functions are already package-level vars
- ⚠️ Can inject chaos into specific functions

**Limitations**:
- ❌ Only works with **package-level function variables**
- ❌ Does NOT work with: struct methods, private functions, closures, local vars
- ❌ Requires `-gcflags=all=-l` (disables inlining optimization)
- ❌ High performance overhead (reflection)
- ❌ **NEVER use in production**
- ❌ Most real Go code cannot be monkey-patched

**Example of what CANNOT be patched**:
```go
// ❌ Struct methods - cannot patch
type Service struct{}
func (s *Service) Process() error { return nil }

// ❌ Private functions - cannot patch
func processInternal() error { return nil }

// ❌ Local functions - cannot patch
func main() {
    localFunc := func() error { return nil }
}
```

**Best for**: Very specific testing scenarios, code already using function variables

---

### Summary Table

| Method | Code Changes | Works with Existing Code | Production-Safe | Performance | Recommendation |
|--------|-------------|--------------------------|-----------------|-------------|----------------|
| **Context-Based** | ✅ Required | ❌ No | ✅ Yes | Excellent | ⭐⭐⭐⭐⭐ New projects |
| **Failpoints** | ✅ Required | ❌ No | ✅ Yes (no-op) | Excellent | ⭐⭐⭐⭐⭐ Production |
| **ToxiProxy** | ❌ None | ✅ Yes | ✅ Yes | Good | ⭐⭐⭐⭐⭐ Network |
| **Monkey Patch** | ⚠️ Specific | ❌ Rarely | ❌ Never | Poor | ⭐ Avoid |

### Choosing the Right Method

**For new Go projects**: Use **Context-Based** + **ToxiProxy**
- Add `MaybeX(ctx)` calls at critical points
- Use ToxiProxy for database/network chaos

**For production systems**: Use **Failpoints** + **ToxiProxy**
- Add `failpoint.Inject()` at critical points
- Build with `-tags failpoint` for chaos testing
- Production builds have zero overhead

**For existing code (cannot modify)**: Use **ToxiProxy only**
- Limited to network-level chaos
- Consider creating wrappers for application-level chaos

**Avoid monkey patching** unless you have very specific needs and understand the limitations.

## Core Components

### Chaos Injectors

**Basic Injectors**:
- **DelayInjector**: Random latency (probability-based or interval-based modes)
- **PanicInjector**: Random panics via `MaybePanic(ctx)` to test recovery mechanisms
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
- **FailpointPanicInjector**: Failpoint-based panic injection (requires `-tags failpoint`)

**CompositeInjector**: Combines multiple injectors for complex failure scenarios

### Validators

**PanicRecoveryValidator**: Ensures proper panic recovery and error handling

**RecursionDepthValidator**: Validates bounded recursion depth (critical for rollback testing)

**GoroutineLeakValidator**: Detects goroutine leaks and resource exhaustion

**SlowIterationValidator**: Identifies long-running executions

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

**floxy_stress_test**: Comprehensive stress test with monkey patching, failpoint, and ToxiProxy

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

## Documentation

- **[TUTORIAL.md](TUTORIAL.md)** - Progressive tutorial from basics to advanced
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Detailed architecture with diagrams
- [GoDoc](https://pkg.go.dev/github.com/rom8726/chaoskit) - API documentation

## Comparison with Other Tools

| Feature | ChaosKit | ToxiProxy | Chaos Mesh | Litmus | Gremlin |
|---------|----------|-----------|------------|--------|---------|
| **Target** | Go libraries | Network layer | Kubernetes | Kubernetes | Infrastructure |
| **Scope** | Code-level | Network-level | Pod-level | Cluster-level | System-level |
| **Code Changes Required** | ✅ Yes (most)* | ❌ No | ❌ No | ❌ No | ❌ No |
| **Language** | Go | Any | Any | Any | Any |
| **Network Chaos** | ✅ (ToxiProxy) | ✅ Native | ✅ | ✅ | ✅ |
| **Custom Injectors** | ✅ Easy | ⚠️ Limited | ⚠️ Limited | ⚠️ Limited | ❌ No |
| **Validators/Assertions** | ✅ Built-in | ❌ No | ⚠️ Limited | ⚠️ Limited | ❌ No |
| **Fine-grained Control** | ✅ Yes | ⚠️ Connection | ⚠️ Pod | ⚠️ Pod | ⚠️ Node |
| **Production Ready** | ⚠️ Failpoints | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes |
| **Learning Curve** | Medium | Low | High | High | Low |
| **Deployment** | Embedded | Proxy | Operator | Operator | Agent |

\* **Exception**: ToxiProxy integration works without code changes

### Key Differences

**ChaosKit Strengths**:
- 🎯 **Fine-grained control**: Inject chaos at specific code points
- 🔍 **Built-in validation**: Verify invariants like recursion depth, goroutine leaks
- 🛠️ **Extensible**: Easy to create custom injectors for Go code
- 📊 **Go-native**: Native Go integration, no external dependencies (except ToxiProxy)

**ChaosKit Limitations**:
- ⚠️ **Requires code instrumentation**: Most features need code modifications
- ⚠️ **Go-only**: Cannot test services in other languages
- ⚠️ **Not infrastructure-level**: Cannot simulate node failures, network partitions
- ⚠️ **Limited to application layer**: Cannot test kernel, storage, or hardware failures

**When to Choose Each Tool**:

| Use Case | Best Tool | Why |
|----------|-----------|-----|
| New Go library/framework | **ChaosKit** | Code-level control, built-in validators |
| Network chaos (any language) | **ToxiProxy** | No code changes, works everywhere |
| Kubernetes chaos | **Chaos Mesh / Litmus** | Pod/container-level chaos |
| Infrastructure chaos | **Gremlin** | System-level failures |
| Existing code (no changes) | **ToxiProxy** | Only network, but works out-of-box |

**ChaosKit is best for:**
- Testing **new** Go libraries and services designed with chaos testing
- Workflow engine reliability with custom validators
- Code-level fault injection with fine-grained control
- Development and CI/CD pipelines
- Learning chaos engineering principles in Go

**Consider alternatives if:**
- You need infrastructure-level chaos (use Chaos Mesh/Litmus/Gremlin)
- You cannot modify code and need more than network chaos
- You're testing non-Go services
- You need production-ready chaos without code changes

## FAQ

### Q: Can I use ChaosKit to test existing libraries without modifying them?

**A**: Only partially. **ToxiProxy integration** works without code changes for network chaos (database connections, HTTP calls, etc.). However, for panic injection, delays, or custom chaos, you'll need to either:
- Add chaos hooks to the code (`MaybePanic`, `MaybeDelay`)
- Create a wrapper that adds chaos between your code and the library
- Fork the library and add failpoints

**Realistic expectation**: ChaosKit works best with code designed for chaos testing from the start.

### Q: Will monkey patching work with my existing code?

**A**: Probably not. Monkey patching has severe limitations:
- ❌ Only works with **package-level function variables** (`var MyFunc = func() {}`)
- ❌ Does **not work** with struct methods, private functions, or local closures
- ❌ Requires building with `-gcflags=all=-l` (disables optimizations)
- ❌ **Never use in production** - only for isolated testing

Most real-world Go code uses methods and private functions, which cannot be monkey-patched. Consider using failpoints or context-based chaos instead.

### Q: Can I use ChaosKit in production?

**A**: With significant caveats:
- ✅ **Failpoints** are production-safe (compile to no-op without `-tags failpoint`)
- ✅ **ToxiProxy** can be used in staging/pre-prod environments
- ✅ **Context-based chaos** is production-safe
- ❌ **Monkey patching** should NEVER be used in production

### Q: How do I test a third-party library like `github.com/someone/library`?

**A**: You have three options:

1. **ToxiProxy** (if the library uses network): Proxy database/HTTP connections
   ```go
   // No library changes needed
   toxiProxy := injectors.ToxiProxyLatency(...)
   ```

2. **Wrapper Pattern**: Create your own wrapper with chaos hooks
   ```go
   type ChaosLibWrapper struct {
       lib *library.Client
   }
   func (w *ChaosLibWrapper) DoWork(ctx context.Context) error {
       chaoskit.MaybeDelay(ctx)
       return w.lib.DoWork()
   }
   ```

3. **Fork and instrument**: Fork the library and add failpoints (maintenance burden)

**Realistic expectation**: Without network operations, testing third-party libraries requires wrapper code or forking.

### Q: What's the difference between ChaosKit and Chaos Mesh/Litmus?

**A**: Different layers:

- **ChaosKit**: Application code level (function calls, goroutines)
    - Requires code changes
    - Fine-grained control at code level
    - Go-specific

- **Chaos Mesh/Litmus**: Infrastructure level (pods, network, nodes)
    - No code changes needed
    - Kubernetes-native
    - Language-agnostic

**Use both**: ChaosKit for application logic, Chaos Mesh for infrastructure failures.

### Q: Can I create custom injectors?

**A**: Yes! Implement the `Injector` interface:

```go
type MyInjector struct{}

func (m *MyInjector) Name() string { return "my-injector" }
func (m *MyInjector) Inject(ctx context.Context) error { /* start chaos */ }
func (m *MyInjector) Stop(ctx context.Context) error { /* stop chaos */ }
```

See [TUTORIAL.md](TUTORIAL.md) Part 5 for detailed examples of custom injectors and validators.

### Q: How do I debug failing scenarios?

**A**: Follow this debugging process:

1. Enable debug logging: `WithLogLevel(slog.LevelDebug)`
2. Use `WithSeed()` for reproducible failures
3. Reduce chaos probabilities to isolate the issue
4. Check validator thresholds (may be too strict)
5. Review execution reports: `reporter.GenerateReport()`
6. Add instrumentation: `RecordRecursionDepth()`, `RecordPanic()`

**Common issues**:
- Validators too strict: Adjust thresholds based on baseline metrics
- Chaos probability too high: Start with 0.01 (1%) and increase gradually
- Missing instrumentation: Ensure you call `MaybeX()` functions

### Q: What's the difference between injectors and validators?

**A**: Two different phases:

- **Injectors** (Inject phase): Introduce chaos during execution
    - Examples: delays, panics, resource pressure
    - Active: Modify system behavior

- **Validators** (Validate phase): Check invariants after execution
    - Examples: goroutine count, recursion depth, memory usage
    - Passive: Verify system state

**Pattern**: Injector creates chaos → System responds → Validator checks if system handled it correctly

### Q: Can I test distributed systems with ChaosKit?

**A**: Limited support:

- ✅ **Network chaos**: Use ToxiProxy to simulate network failures between services
- ✅ **Per-service chaos**: Each service runs its own ChaosKit scenarios
- ❌ **Coordinated chaos**: No built-in support for multi-service orchestration
- ❌ **Network partitions**: Use Chaos Mesh or Litmus instead

**Approach for distributed systems**:
1. Use ToxiProxy for inter-service network chaos
2. Run ChaosKit independently in each service
3. Use Chaos Mesh for infrastructure-level failures (network partitions, node failures)

### Q: How do I test rollback mechanisms?

**A**: Use recursion depth tracking:

```go
func CompensateOrder(ctx context.Context, depth int) error {
    chaoskit.RecordRecursionDepth(ctx, depth)
    chaoskit.MaybePanic(ctx)  // Inject chaos during rollback
    
    // Your compensation logic
    if depth < maxDepth {
        return CompensateOrder(ctx, depth+1)
    }
}

// Validate bounded recursion
scenario := chaoskit.NewScenario("rollback-test").
    Assert("recursion-depth", validators.RecursionDepthLimit(10))
```

See `examples/workflow_engine/` for complete examples.

### Q: Can I use ChaosKit with standard Go tests?

**A**: Yes! ChaosKit scenarios can run from Go test functions:

```go
func TestServiceWithChaos(t *testing.T) {
    scenario := chaoskit.NewScenario("test").
        WithTarget(myService).
        Inject("delay", injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond)).
        Assert("no-leaks", validators.GoroutineLimit(100)).
        Repeat(100).
        Build()
    
    if err := chaoskit.Run(context.Background(), scenario); err != nil {
        t.Fatalf("Chaos test failed: %v", err)
    }
}
```

Run with: `go test -v ./...`

### Q: Why are my chaos injections not working?

**A**: Check these common issues:

1. **Context-based chaos**: Did you call `MaybePanic(ctx)` / `MaybeDelay(ctx)` in your code?
    - These functions do nothing if not called explicitly

2. **Monkey patching**: Did you build with `-gcflags=all=-l`?
    - Without this flag, functions get inlined and can't be patched

3. **Probability too low**: `0.01` = 1% chance
    - Run 100+ iterations or increase probability for testing

4. **Injector not started**: Check logs for "injector started" messages

5. **Wrong injection method**: Some methods only work with specific code patterns

**Debugging**: Enable debug logging and review injector metrics: `injector.GetMetrics()`

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

### "I added injectors but no chaos is happening"

**Root cause**: Most chaos injection requires explicit instrumentation in your code.

**Solutions**:
- **Context-based chaos**: Add `chaoskit.MaybePanic(ctx)`, `chaoskit.MaybeDelay(ctx)` in your code
- **Failpoints**: Add `failpoint.Inject("name", func() {...})` and build with `-tags failpoint`
- **Monkey patching**: Check if your functions are package-level vars (most aren't)
- **ToxiProxy**: Verify proxy is running and connections are routed through it

**Check**: Enable debug logging to see if injectors are starting:
```go
executor := chaoskit.NewExecutor(
    chaoskit.WithLogLevel(slog.LevelDebug),
)
```

### "Can I test library X without modifying it?"

**Short answer**: Only if X uses network I/O (use ToxiProxy). Otherwise, you need code changes.

**Options**:
1. **ToxiProxy**: If library uses database/HTTP/network → proxy connections (no code changes)
2. **Wrapper**: Create your own wrapper with chaos hooks
3. **Fork**: Fork library and add failpoints/context chaos
4. **Different tool**: Consider infrastructure-level tools (Chaos Mesh, Litmus)

**Reality check**: ChaosKit is designed for proactive chaos engineering, not retrofitting existing code.

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

### Monkey patching not working

**Problem**: Functions aren't being patched.

**Root causes**:
- ❌ Function is a struct method (not supported)
- ❌ Function is private (not supported)
- ❌ Function is a local variable (not supported)
- ❌ Not building with `-gcflags=all=-l`
- ❌ Function got inlined despite the flag

**Solution**: Use context-based chaos or failpoints instead. Monkey patching works in very limited scenarios.

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
