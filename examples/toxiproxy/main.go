package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/injectors"
	"github.com/rom8726/chaoskit/validators"
)

// NetworkTarget demonstrates network chaos testing with ToxiProxy
type NetworkTarget struct {
	httpClient *http.Client
	baseURL    string
	proxyURL   string
	callCount  int
}

func NewNetworkTarget(baseURL, proxyHost string) *NetworkTarget {
	// Parse proxy URL - proxyHost should be like "localhost:18080"
	proxyAddr := proxyHost
	if proxyAddr == "" {
		proxyAddr = "localhost:18080"
	}

	// Create HTTP client that uses proxy
	// ToxiProxy is a TCP-level proxy, so we connect to proxy and it forwards to upstream
	transport := &http.Transport{
		Proxy: nil, // Direct connection (we'll connect to proxy address)
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Always connect to proxy - it will forward based on upstream config
			dialer := &net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			log.Printf("[Client] Dialing proxy at %s (requested target: %s)", proxyAddr, addr)

			return dialer.DialContext(ctx, network, proxyAddr)
		},
	}

	return &NetworkTarget{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second, // Increased timeout for network chaos
		},
		baseURL:  baseURL,
		proxyURL: proxyAddr,
	}
}

func (t *NetworkTarget) Name() string {
	return "network-target"
}

func (t *NetworkTarget) Setup(ctx context.Context) error {
	log.Println("[Target] Setting up network target...")
	log.Printf("[Target] Base URL: %s", t.baseURL)
	log.Printf("[Target] Proxy URL: %s", t.proxyURL)
	log.Println("[Target] Will make requests through ToxiProxy")

	return nil
}

func (t *NetworkTarget) Teardown(ctx context.Context) error {
	log.Printf("[Target] Tearing down (total requests: %d)", t.callCount)

	return nil
}

// MakeRequest makes an HTTP request through the proxy
// Note: The request URL is still the original target, but connection goes through proxy
func (t *NetworkTarget) MakeRequest(ctx context.Context, path string) error {
	t.callCount++

	// Construct full URL - proxy will forward based on Host header
	url := fmt.Sprintf("%s%s", t.baseURL, path)
	log.Printf("[Target] Making request #%d to %s (via proxy %s)", t.callCount, url, t.proxyURL)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	start := time.Now()
	resp, err := t.httpClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("[Target] Request #%d failed after %v: %v", t.callCount, duration, err)

		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("[Target] Request #%d succeeded: status=%d, duration=%v, body_len=%d",
		t.callCount, resp.StatusCode, duration, len(body))

	return nil
}

// Execute runs multiple requests to demonstrate network chaos effects
func (t *NetworkTarget) Execute(ctx context.Context) error {
	paths := []string{"/", "/health", "/api/v1/test"}

	for _, path := range paths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := t.MakeRequest(ctx, path); err != nil {
				log.Printf("[Target] Request to %s failed: %v", path, err)
				// Continue with other requests even if one fails
			}
			time.Sleep(100 * time.Millisecond) // Small delay between requests
		}
	}

	return nil
}

// RunNetworkTest is the step function
func RunNetworkTest(ctx context.Context, target chaoskit.Target) error {
	networkTarget, ok := target.(*NetworkTarget)
	if !ok {
		return fmt.Errorf("target is not a NetworkTarget")
	}

	return networkTarget.Execute(ctx)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=== ChaosKit ToxiProxy Network Chaos Example ===")
	log.Println()
	log.Println("PREREQUISITES:")
	log.Println("1. Start ToxiProxy server: toxiproxy-server")
	log.Println("   Default: http://localhost:8474")
	log.Println("2. Have a target HTTP server running (or use httpbin.org)")
	log.Println()
	log.Println("This example demonstrates:")
	log.Println("- Latency injection (delays)")
	log.Println("- Bandwidth limiting")
	log.Println("- Connection timeouts")
	log.Println("- Packet slicing (intermittent drops)")
	log.Println()

	// ToxiProxy configuration
	toxiproxyHost := "http://localhost:8474"
	// Using HTTP (not HTTPS) for simplicity - ToxiProxy works better with HTTP
	baseURL := "http://httpbin.org" // Example target server (HTTP)
	proxyName := "httpbin-proxy"
	proxyListen := "localhost:18080"  // Proxy listens here
	proxyUpstream := "httpbin.org:80" // Upstream server (HTTP)

	// Check if user wants to continue
	log.Println("Press Ctrl+C to stop, or wait 3 seconds to continue...")
	time.Sleep(3 * time.Second)

	// Create ToxiProxy client
	client := injectors.NewToxiProxyClient(toxiproxyHost)
	manager := injectors.NewToxiProxyManager(client)

	// Setup proxy (create it if it doesn't exist)
	proxyConfig := injectors.ProxyConfig{
		Name:     proxyName,
		Listen:   proxyListen,
		Upstream: proxyUpstream,
		Enabled:  true,
	}

	log.Println("[Setup] Creating ToxiProxy proxy...")
	if err := manager.CreateProxy(proxyConfig); err != nil {
		log.Printf("[Setup] Note: Proxy might already exist: %v", err)
	}

	// Use proxy listen address for client connection
	// The proxy will forward to upstream (baseURL)
	target := NewNetworkTarget(baseURL, proxyListen)

	// Create various network injectors
	log.Println("[Setup] Creating network chaos injectors...")

	// 1. Latency injector - adds delay to network calls
	latencyInjector := injectors.ToxiProxyLatency(
		client,
		proxyName,
		200*time.Millisecond, // base latency
		50*time.Millisecond,  // jitter
	)

	// 2. Bandwidth injector - limits transfer speed
	bandwidthInjector := injectors.ToxiProxyBandwidth(
		client,
		proxyName,
		100, // 100 KB/s limit
	)

	// 3. Timeout injector - causes connection timeouts
	timeoutInjector := injectors.ToxiProxyTimeout(
		client,
		proxyName,
		1000*time.Millisecond, // 1 second timeout
	)

	// 4. Slicer injector - creates packet loss and delays
	slicerInjector := injectors.ToxiProxySlicer(
		client,
		proxyName,
		1024,                 // average packet size: 1KB
		512,                  // size variation: 512 bytes
		100*time.Microsecond, // delay between packets
	)

	// Build scenario with different injector combinations
	// Uncomment the injectors you want to test

	// Create scenario - start with latency, can combine with others
	scenario := chaoskit.NewScenario("toxiproxy-demo").
		WithTarget(target).
		Step("network-test", RunNetworkTest).
		// Test latency injection (always active)
		Inject("latency", latencyInjector).
		// Uncomment one or more to test additional chaos:
		//
		// Bandwidth limiting - slows down transfers
		Inject("bandwidth", bandwidthInjector).
		//
		// Timeouts - causes connection failures
		Inject("timeout", timeoutInjector).
		//
		// Packet slicing - creates unreliable connection
		Inject("slicer", slicerInjector).
		//
		// Combine multiple for realistic network conditions:
		// Inject("bandwidth", bandwidthInjector).
		// Inject("slicer", slicerInjector).
		Assert("execution-time", validators.ExecutionTime(time.Millisecond, 30*time.Second)).
		Repeat(5).
		Build()

	// Run scenario
	ctx := context.Background()
	if err := chaoskit.Run(ctx, scenario); err != nil {
		log.Printf("Scenario completed with errors: %v", err)
	}

	// Cleanup
	log.Println("\n[Cleanup] Removing proxy...")
	if err := manager.DeleteProxy(proxyName); err != nil {
		log.Printf("[Cleanup] Error removing proxy: %v", err)
	}

	log.Println("\n=== Test Complete ===")
	log.Println("\nKey Points:")
	log.Println("1. ToxiProxy injectors modify network behavior at proxy level")
	log.Println("2. All traffic through proxy is affected")
	log.Println("3. Latency: adds delays with jitter")
	log.Println("4. Bandwidth: limits transfer speed")
	log.Println("5. Timeout: causes connection timeouts")
	log.Println("6. Slicer: creates packet loss and delays (unreliable connection)")
	log.Println("\nTry uncommenting different injectors to see their effects!")
	log.Println("\nTo test with your own server:")
	log.Println("  - Change baseURL to your server address")
	log.Println("  - Update proxyUpstream to point to your server")
	log.Println("  - Modify proxyListen if port 18080 is occupied")
}
