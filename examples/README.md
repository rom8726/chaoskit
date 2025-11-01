# ChaosKit Examples

This directory contains example applications demonstrating how to use ChaosKit for chaos testing.

## Examples

### 0. Monkey Patch Example (`monkey_patch/`)

Demonstrates monkey patching injection for intercepting function calls and injecting panics.

**WARNING**: Requires `-gcflags=all=-l` flag to disable compiler inlining:
```bash
go run -gcflags=all=-l examples/monkey_patch/main.go
```

**What it demonstrates:**
- Runtime patching of functions without modifying source code
- Injecting panics with configurable probability
- Handling multiple functions with different signatures
- Automatic patch restoration

**Note**: Monkey patching is unsafe and should only be used in testing environments.

### 1. ToxiProxy Network Chaos (`toxiproxy/`)

Demonstrates network chaos testing using ToxiProxy injectors.

**Prerequisites:**
- Install and start ToxiProxy server: `toxiproxy-server`
- Target HTTP server (or use httpbin.org)

**Run:**
```bash
cd examples/toxiproxy
go run main.go
```

**What it demonstrates:**
- Latency injection (network delays)
- Bandwidth limiting
- Connection timeouts
- Packet slicing (intermittent drops)
- Integration with ToxiProxy proxy management

**See:** `examples/toxiproxy/README.md` for detailed setup instructions.

### 2. Simple Example (`simple/`)

A basic example that demonstrates:
- Setting up a workflow engine as a test target
- Configuring chaos injectors (delays, panics)
- Adding validators (goroutine leak, recursion depth, infinite loop detection)
- Running a fixed number of iterations

**Run:**
```bash
make run-simple
# or
./bin/simple
```

**What it tests:**
- Workflow execution with saga-style rollback
- Recursion depth tracking during rollback
- Random failures to trigger rollback scenarios
- Chaos injection (delays and panics)
- Validation of system invariants

### 2. Continuous Testing Example (`continuous/`)

An advanced example that demonstrates:
- Continuous chaos testing (runs until stopped with Ctrl+C)
- Multiple workflow patterns (simple, complex, nested, parallel)
- Random scenario generation
- Real-time statistics reporting
- Concurrent scenario execution

**Run:**
```bash
make run-continuous
# or
./bin/continuous
```

**What it tests:**
- Different workflow execution patterns
- Nested recursion with depth tracking
- Parallel workflow execution
- Continuous stress testing
- Long-running reliability validation

**Features:**
- Runs scenarios every 500ms
- Reports statistics every 5 seconds
- Tests various chaos scenarios randomly
- Validates recursion depth, goroutine leaks, and infinite loops

## Building

Build all examples:
```bash
make build
```

This creates binaries in the `bin/` directory.

## Understanding the Output

### Simple Example Output

```
=== ChaosKit Simple Example ===
[Engine] Setting up workflow engine...
[CHAOS] Delay injector started (range: 5ms-25ms)
[Engine] Tearing down workflow engine...
[Engine] Final stats: map[current_depth:0 executions:50 rollbacks:5]

ChaosKit Execution Report
========================
Total Executions: 50
Success: 45
Failed: 5
Success Rate: 90.00%
Average Duration: 25ms
```

### Continuous Example Output

```
=== ChaosKit Continuous Testing ===
[Stats] Total=150 Success=135 Failed=15 Rollbacks=20 Depth=0
[Stats] Total=320 Success=288 Failed=32 Rollbacks=45 Depth=0
...
[Main] Shutting down...

=== Final Statistics ===
Total scenarios: 500
Total workflow runs: 1500
Successful runs: 1350
Failed runs: 150
Rollbacks: 200
```

## Key Concepts Demonstrated

### 1. Workflow Engine with Rollback

Both examples implement a workflow engine that:
- Executes steps sequentially
- Tracks recursion depth
- Performs rollback on failures
- Prevents infinite recursion

### 2. Chaos Injection

Examples use various injectors:
- **DelayInjector**: Adds random delays to simulate latency
- **PanicInjector**: Randomly triggers panics to test recovery
- **CPUInjector**: Simulates CPU stress (in other examples)

### 3. Validation

Validators check system invariants:
- **RecursionDepthLimit**: Ensures rollback doesn't exceed depth limits
- **GoroutineLimit**: Detects goroutine leaks
- **NoInfiniteLoop**: Detects stuck executions
- **ExecutionTime**: Validates performance bounds

### 4. Metrics and Reporting

The framework automatically:
- Tracks execution statistics
- Records successes and failures
- Measures execution duration
- Generates reports (text and JSON)

## Customization

You can customize the examples by:

1. **Adjusting failure rates**: Change the probability in `rand.Float64() < 0.1`
2. **Modifying chaos intensity**: Adjust injector parameters
3. **Changing validation thresholds**: Update validator limits
4. **Adding new workflow patterns**: Implement additional execution patterns
5. **Extending metrics**: Add custom metrics collection

## Next Steps

After running these examples, you can:

1. Integrate ChaosKit with your own workflow engine
2. Create custom injectors for your specific failure modes
3. Implement domain-specific validators
4. Build continuous testing pipelines
5. Generate detailed reports for analysis
