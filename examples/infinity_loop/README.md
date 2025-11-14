# Infinite Loop Detection Example

This example demonstrates the **InfiniteLoopValidator** which detects infinite loops in step execution by applying timeouts to each step.

## What It Demonstrates

1. **Infinite Loop Detection**: The validator wraps each step with a timeout to detect hung steps
2. **Step Wrapping**: Shows how validators can intercept and wrap step execution
3. **Timeout Handling**: Demonstrates graceful handling of steps that exceed timeout
4. **Chaos Integration**: Works alongside other validators and injectors

## Key Features

- **Timeout-based Detection**: Any step taking longer than the configured timeout is detected as an infinite loop
- **Automatic Wrapping**: Steps are automatically wrapped by validators implementing `StepWrapper` interface
- **Graceful Error Handling**: Detected infinite loops are logged and reported without crashing the executor
- **Context Awareness**: Steps can still respect context cancellation to break loops

## Running the Example

```bash
cd examples/infinity_loop
go run main.go
```

## Expected Behavior

The example runs 10 iterations with:
- **Normal tasks**: Complete successfully within timeout
- **Random tasks**: 90% normal, 10% infinite loop (for testing)
- **Infinite loop detection**: Steps exceeding 200ms timeout are detected and logged

## Example Output

```
=== ChaosKit Infinite Loop Detection Example ===
[Service] Setting up processing service...
Running scenario...
[Service] Task 123: Processing...
[Service] Task 123: Completed in 87ms
[Service] Task 456: Starting (will loop infinitely)
ERROR infinite loop detected validator=no_infinite_loop_200ms step=process-random timeout=200ms total_detections=1
WARN step goroutine still running after timeout validator=no_infinite_loop_200ms step=process-random
...
=== Chaos Test Report ===
...
```

## How It Works

1. **Validator Setup**:
   ```go
   Assert("no_infinite_loop", validators.NoInfiniteLoop(200*time.Millisecond))
   ```
   - Creates a validator with 200ms timeout
   - Any step exceeding this timeout is considered an infinite loop

2. **Step Wrapping**:
   - The executor automatically detects validators implementing `StepWrapper` interface
   - Each step is wrapped with timeout logic before execution
   - Wrappers are applied in reverse order (first wrapper becomes outermost)

3. **Detection Logic**:
   - Step is executed in a separate goroutine with timeout context
   - If step completes within timeout → success
   - If timeout expires → infinite loop detected
   - Goroutine is given 100ms grace period to complete after timeout

4. **Error Reporting**:
   - Detections are logged with step name, timeout, and total count
   - Errors are returned to executor and included in execution results
   - Report includes failed iterations due to infinite loops

## Validator Interface

The `InfiniteLoopValidator` implements:
- `Validator` interface: Standard validation interface
- `StepWrapper` interface: Allows wrapping step execution
- `GetDetectionsCount()`: Returns total number of detections

## Use Cases

- **Detecting hung operations**: Find steps that never complete
- **Performance monitoring**: Identify steps that take too long
- **Resource leak detection**: Find steps that consume resources indefinitely
- **Integration testing**: Ensure steps complete within expected timeframes

## Configuration

Adjust the timeout based on your needs:
- **Short timeout (50-100ms)**: For fast operations, strict timing
- **Medium timeout (200-500ms)**: For typical operations
- **Long timeout (1-5s)**: For slow operations, network calls

## Notes

- The validator uses goroutines, so some goroutines may remain if steps don't respect context cancellation
- Steps should check `ctx.Done()` to break loops when context is cancelled
- Multiple validators can wrap steps (wrappers are applied in reverse order)
- Detections are counted per validator instance

