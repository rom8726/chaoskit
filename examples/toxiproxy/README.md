# ToxiProxy Network Chaos Example

This example demonstrates how to use ToxiProxy injectors for network chaos testing with ChaosKit.

## Prerequisites

### 1. Install ToxiProxy

```bash
# macOS
brew install toxiproxy

# Or download from: https://github.com/Shopify/toxiproxy/releases
```

### 2. Start ToxiProxy Server

```bash
toxiproxy-server

# Server runs on http://localhost:8474 by default
```

### 3. Target Server

You can use:
- **httpbin.org** (default in example) - public HTTP testing service (HTTP, not HTTPS)
- Your own local HTTP server
- Any accessible HTTP endpoint

**Note**: The example uses HTTP (not HTTPS) for simplicity. For HTTPS, you'll need additional TLS configuration.

## Running the Example

```bash
cd examples/toxiproxy
go run main.go
```

The example will:
1. Create a ToxiProxy proxy that routes traffic to the target server
2. Apply network chaos injectors (latency, bandwidth, timeout, slicer)
3. Make HTTP requests through the proxy
4. Demonstrate the effects of network chaos

## Configuration

Edit `main.go` to customize:

```go
// ToxiProxy server address
toxiproxyHost := "http://localhost:8474"

// Target server (HTTP for simplicity)
baseURL := "http://httpbin.org"

// Proxy configuration
proxyName := "httpbin-proxy"
proxyListen := "localhost:18080"    // Where proxy listens
proxyUpstream := "httpbin.org:80"  // Upstream server (HTTP port)
```

## Network Injectors

### 1. Latency Injector

Adds delay and jitter to network requests:

```go
latencyInjector := injectors.ToxiProxyLatency(
    client,
    proxyName,
    200*time.Millisecond, // base latency
    50*time.Millisecond,  // jitter (±50ms)
)
```

**Effect**: All requests will have 200ms ± 50ms delay.

### 2. Bandwidth Injector

Limits network transfer speed:

```go
bandwidthInjector := injectors.ToxiProxyBandwidth(
    client,
    proxyName,
    100, // 100 KB/s limit
)
```

**Effect**: Downloads/uploads will be throttled to 100 KB/s.

### 3. Timeout Injector

Causes connection timeouts:

```go
timeoutInjector := injectors.ToxiProxyTimeout(
    client,
    proxyName,
    1000*time.Millisecond, // 1 second timeout
)
```

**Effect**: Connections will timeout after 1 second.

### 4. Slicer Injector

Creates packet loss and delays (unreliable connection):

```go
slicerInjector := injectors.ToxiProxySlicer(
    client,
    proxyName,
    1024,              // average packet size: 1KB
    512,               // size variation: 512 bytes
    100*time.Microsecond, // delay between packets
)
```

**Effect**: Packets are fragmented and delayed, simulating unreliable network.

## Using Multiple Injectors

You can combine multiple injectors:

```go
scenario := chaoskit.NewScenario("network-chaos").
    WithTarget(target).
    Step("test", RunNetworkTest).
    Inject("latency", latencyInjector).
    Inject("bandwidth", bandwidthInjector).
    Inject("slicer", slicerInjector).
    Build()
```

## Custom Target Server

To test with your own server:

1. Start your HTTP server (e.g., on port 8080)
2. Update configuration:

```go
baseURL := "http://localhost:8080"
proxyUpstream := "localhost:8080"
```

3. Make requests through proxy:

```bash
# Direct request (without chaos)
curl http://localhost:8080/api

# Request through proxy (with chaos)
curl http://localhost:18080/api
```

## Understanding the Output

```
[TOXIPROXY] Proxy created: httpbin-proxy (localhost:18080 -> httpbin.org:443)
[TOXIPROXY] Latency injected on httpbin-proxy: 200ms ±50ms
[Target] Making request #1 to https://httpbin.org/
[Target] Request #1 succeeded: status=200, duration=287ms, body_len=9593
```

- **duration** shows the effect of latency injector
- Higher duration = more network chaos applied
- Failed requests indicate timeouts or connection issues

## Troubleshooting

### ToxiProxy server not running

```
Error: failed to get proxy httpbin-proxy: ...
```

**Solution**: Start ToxiProxy server:
```bash
toxiproxy-server
```

### Connection refused

```
Error: request failed: dial tcp [::1]:18080: connect: connection refused
```

**Solution**: 
- Check that proxy was created successfully
- Verify proxy listen address is correct
- Make sure no firewall is blocking the port

### Proxy already exists

```
Error: failed to create proxy: proxy already exists
```

**Solution**: Delete existing proxy first:
```bash
toxiproxy-cli delete httpbin-proxy
```

Or modify the code to handle this case (example already does this).

## Advanced Usage

### Contextual Network Injector

For context-based network chaos (without ToxiProxy), see:
- `examples/chaos_context/` - demonstrates `MaybeNetworkChaos()`
- Uses `ContextualNetworkInjector` for per-request chaos

### Combining with Other Injectors

```go
scenario := chaoskit.NewScenario("full-chaos").
    WithTarget(target).
    Step("test", RunTest).
    // Network chaos
    Inject("latency", latencyInjector).
    // Application chaos
    Inject("delay", injectors.RandomDelay(10*time.Millisecond, 50*time.Millisecond)).
    Inject("panic", injectors.PanicProbability(0.05)).
    // Resource chaos
    Inject("cpu", injectors.CPUStress(2)).
    Build()
```

## References

- [ToxiProxy Documentation](https://github.com/Shopify/toxiproxy)
- [ChaosKit Network Injectors](../injectors/network.go)
- [Contextual Network Injector](../injectors/network_contextual.go)

