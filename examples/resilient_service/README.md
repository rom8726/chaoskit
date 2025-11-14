# Resilient Service Example

This example demonstrates a **resilient service** that gracefully handles chaos (panics, errors, delays) and always results in `VerdictPass`.

## What It Demonstrates

1. **Panic Recovery**: The service uses `defer recover()` to catch and handle all panics injected by chaos
2. **Error Handling**: Errors are handled gracefully with retry logic
3. **Chaos Injection**: Uses `MaybePanic()` and `MaybeDelay()` to inject chaos
4. **Resilience**: Despite chaos, the service maintains high success rate through proper error handling

## Key Features

- **Panic Recovery**: All panics are caught and recovered, preventing crashes
- **Retry Logic**: Failed requests are retried up to 3 times
- **Statistics Tracking**: Tracks requests, successes, panics, errors, and recoveries
- **Proper Validators**: Uses validators that allow expected panics (since they're recovered)

## Running the Example

```bash
cd examples/resilient_service
go run main.go
```

## Expected Output

The example should always result in `VerdictPass` because:

1. All panics are recovered (counted but don't crash the service)
2. Errors are handled with retry logic
3. Validators allow up to 20 panics (which is more than expected)
4. Success rate threshold is set to 50% (allowing for chaos)
5. All iterations complete successfully (even if individual requests fail)

## Example Output

```
=== Verdict: PASS ===
✅ All tests passed! Service demonstrated resilience to chaos.

=== Service Statistics ===
Total requests: 125
Successful: 125
Panics (recovered): 25
Errors (handled): 34
Recovered panics: 25
Success rate: 100.00%
```

## How It Works

1. **Chaos Injection**: 
   - 15% probability of panic injection via `PanicProbability(0.15)`
   - Random delays between 5-20ms

2. **Panic Handling**:
   - Each request has `defer recover()` to catch panics
   - Panics are logged and counted, but don't crash the service

3. **Error Handling**:
   - Uses retry logic (up to 3 attempts)
   - Errors are logged but don't fail the iteration

4. **Validators**:
   - `NoPanics(20)`: Allows up to 20 panics (expected ~19 with 25 iterations)
   - `GoroutineLimit(500)`: High limit to allow for retries
   - `RecursionDepthLimit(50)`: Prevents infinite recursion
   - `NoInfiniteLoop(2s)`: Prevents infinite loops

5. **Thresholds**:
   - `MinSuccessRate: 0.5`: Allows 50% success rate
   - `MaxFailedIterations: 15`: Allows up to 15 failed iterations out of 25

## Why It Always Passes

The service is designed to be resilient:
- Panics are **recovered**, not propagated
- Errors are **handled** with retries, not fatal
- Validators are **configured** to allow expected chaos
- Thresholds are **set** to realistic values for chaos testing

This demonstrates that with proper error handling and recovery mechanisms, a service can maintain stability even under chaos conditions.

