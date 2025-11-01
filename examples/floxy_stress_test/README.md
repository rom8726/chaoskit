# Floxy Stress Test with ChaosKit

Comprehensive stress testing for Floxy workflow engine using ChaosKit framework.

## Overview

This example demonstrates how to use ChaosKit to perform reliability testing on the Floxy saga-based workflow engine.
It tests various workflow patterns under chaos conditions to validate:

- Rollback recursion depth limits
- Goroutine leak prevention
- Infinite loop detection
- Compensation handler reliability
- SavePoint functionality
- Fork/Join parallel execution

## Features

### Workflow Patterns Tested

1. **Simple Order Workflow**
   - Linear execution with compensation handlers
   - Tests basic rollback mechanism

2. **Complex Order Workflow**
   - Multiple SavePoints
   - Rollback to specific checkpoints
   - Multi-level compensation

3. **Parallel Processing Workflow**
   - Fork/Join patterns
   - Concurrent step execution
   - Parallel compensation

4. **Nested Workflow**
   - Nested Fork/Join structures
   - Complex dependency graphs
   - Deep rollback chains

### Chaos Injection Methods

This example demonstrates three advanced chaos injection techniques:

1. **Monkey Patching** - Runtime function patching
   - Panic injection in handlers (5% probability)
   - Delay injection in handlers (15-20% probability)
   - No code modification required (functions must be variables, not methods)
   - Requires `-gcflags=all=-l` build flag
   - Functions must be passed as pointers: `&processPaymentHandler`

2. **Gofail** - Failpoint-based injection
   - Panic injection via failpoints
   - Requires build with `-tags failpoint`
   - Requires instrumenting code with failpoint.Inject
   - 3% probability per failpoint

3. **ToxiProxy** - Network chaos for database
   - Latency injection (100ms ± 20ms)
   - Bandwidth limiting (500 KB/s)
   - Connection timeouts (2 seconds)
   - Optional: Enable with `USE_TOXIPROXY=true`

**Note**: Monkey patching is always active. Gofail and ToxiProxy are optional.

### Validation

- **Recursion Depth**: Ensures rollback depth stays below 50
- **Goroutine Leak**: Monitors goroutine count (limit: 500)
- **Infinite Loop**: Detects stuck workflows (timeout: 15s)

### Metrics Collection

Integrated Floxy plugins:
- **Rollback Depth Plugin**: Tracks maximum rollback depth per instance
- **Metrics Plugin**: Collects workflow and step statistics

## Prerequisites

### Required

- PostgreSQL database running on `localhost:5435`
  - Database: `floxy`
  - User: `floxy`
  - Password: `password`

Start PostgreSQL using Docker:

```bash
docker run -d \
  --name floxy-postgres \
  -e POSTGRES_DB=floxy \
  -e POSTGRES_USER=floxy \
  -e POSTGRES_PASSWORD=password \
  -p 5435:5432 \
  postgres:15
```

### Optional

**ToxiProxy** (for database network chaos):
```bash
# Install ToxiProxy
brew install toxiproxy  # macOS
# Or download from: https://github.com/Shopify/toxiproxy/releases

# Start ToxiProxy server
toxiproxy-server
```

**Gofail** (for failpoint-based injection):
- Already included in dependencies
- Requires build with `-tags failpoint`
- Requires uncommenting failpoint.Inject calls in handlers

## Running the Test

### Basic Run (Monkey Patching Only)

```bash
cd examples/floxy_stress_test

# Run with monkey patching (requires -gcflags=all=-l)
go run -gcflags=all=-l main.go
```

### With ToxiProxy (Database Network Chaos)

```bash
# Start ToxiProxy server (in separate terminal)
toxiproxy-server

# Run with ToxiProxy enabled
USE_TOXIPROXY=true go run -gcflags=all=-l main.go
```

### With Gofail (Failpoint Injection)

1. Uncomment failpoint calls in handlers (`main.go`)
2. Add import: `import "github.com/pingcap/failpoint"`
3. Build and run:

```bash
go run -tags failpoint -gcflags=all=-l main.go
```

### Full Chaos (All Injectors)

```bash
# Start ToxiProxy
toxiproxy-server

# Run with all injectors
USE_TOXIPROXY=true go run -tags failpoint -gcflags=all=-l main.go
```

The test will run for 60 seconds by default. Press `Ctrl+C` to stop earlier.

## Output

### Real-time Statistics

Every 10 seconds, you'll see statistics like:

```
[Stats] map[
  failed_runs:12
  max_rollback_depth:4
  metrics:map[
    avg_step_duration_ms:15
    avg_workflow_duration_ms:120
    steps_completed:450
    steps_failed:35
    steps_started:485
    workflows_completed:88
    workflows_failed:12
    workflows_started:100
  ]
  rollback_count:12
  successful_runs:88
  total_workflows:100
]
```

### Final Report

At the end, you'll see:

```
=== Final Report ===
ChaosKit Execution Report
========================
Total Executions: 150
Success: 135
Failed: 15
Success Rate: 90.00%
Average Duration: 250ms

=== Floxy Statistics ===
Total workflows: 150
Successful runs: 135
Failed runs: 15
Rollback count: 15
Max rollback depth: 6
Metrics: map[...]
```

## What It Tests

### Rollback Mechanism

- Verifies compensation handlers execute in correct order
- Ensures rollback doesn't exceed maximum depth
- Tests SavePoint rollback functionality
- Validates state consistency after rollback

### Concurrency

- 5 concurrent workers processing workflows
- Multiple workflow instances running simultaneously
- Fork/Join parallel execution
- Race condition detection

### Error Handling

- Random failures trigger compensation
- Retry mechanisms with max retry limits
- Graceful degradation under load
- Proper error propagation

### Resource Management

- Goroutine leak detection
- Database connection pooling
- Worker lifecycle management
- Graceful shutdown

## Configuration

Modify the test parameters in `main.go`:

### Monkey Patching

```go
// Panic probability in handlers
PanicProb: 0.05  // 5%

// Delay injection
DelayMin: 50 * time.Millisecond
DelayMax: 200 * time.Millisecond
Probability: 0.2  // 20% chance
```

### Gofail

```go
// Failpoint names (must match failpoint.Inject calls)
failpointNames := []string{
    "payment-handler-panic",
    "inventory-handler-panic",
    "shipping-handler-panic",
}
probability: 0.03  // 3% per failpoint
window: 500 * time.Millisecond
```

### ToxiProxy

```go
// Enable via environment variable
USE_TOXIPROXY=true

// Or modify connection config
proxyConfig := injectors.ProxyConfig{
    Name:     "postgres-proxy",
    Listen:   "localhost:6432",
    Upstream: "localhost:5435",
}
```

### Other Parameters

```go
// Worker count
workerCount := 5

// Test duration
RunFor(60 * time.Second)

// Validation limits
Assert("recursion-depth", validators.RecursionDepthLimit(50))
Assert("goroutine-leak", validators.GoroutineLimit(500))
```

## Integration with ChaosKit

This example demonstrates ChaosKit's advanced capabilities:

1. **Target Interface**: Floxy engine wrapped as ChaosKit target
2. **Monkey Patching Injectors**: Runtime function patching for panic and delay injection
3. **Gofail Injectors**: Failpoint-based chaos injection (compile-time)
4. **ToxiProxy Injectors**: Network-level chaos for database connections
5. **Validators**: Recursion depth, goroutine leak, infinite loop detection
6. **Failure Policy**: ContinueOnFailure for continuous testing
7. **Metrics**: Automatic collection and reporting

### Injection Methods Comparison

| Method | Type | Requires Build Flags | Code Changes | When to Use |
|--------|------|---------------------|--------------|-------------|
| Monkey Patching | Runtime | `-gcflags=all=-l` | No | Runtime patching, testing external code |
| Gofail | Compile-time | `-tags failpoint` | Yes (failpoint.Inject) | Production-safe, explicit injection points |
| ToxiProxy | Network | No | No (proxy setup) | Network-level chaos, database testing |

## Troubleshooting

**Database connection errors**

Ensure PostgreSQL is running and accessible:
```bash
psql -h localhost -p 5435 -U floxy -d floxy
```

**High failure rate**

This is expected with 15% random failure injection. Adjust the probability:
```go
"should_fail": rand.Float64() < 0.05  // 5% instead of 15%
```

**Goroutine limit exceeded**

Increase the limit or reduce worker count:
```go
Assert("goroutine-leak", validators.GoroutineLimit(1000))
// or
workerCount := 3
```

## Expected Results

A healthy Floxy engine should show:

- Success rate: 80-90% (with 15% random failures)
- Max rollback depth: < 10
- No goroutine leaks
- No infinite loops
- Proper compensation execution
- Consistent state after rollback

## Next Steps

- Increase test duration for long-running validation
- Add more complex workflow patterns
- Test DLQ mode workflows
- Add custom validators for business logic
- Integrate with CI/CD pipeline
