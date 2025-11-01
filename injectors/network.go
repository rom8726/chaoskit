package injectors

import (
	"context"
	"fmt"
	"sync"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
)

// ToxiProxyClient wraps the ToxiProxy client for easier usage
type ToxiProxyClient struct {
	client *toxiproxy.Client
}

// NewToxiProxyClient creates a new ToxiProxy client
func NewToxiProxyClient(host string) *ToxiProxyClient {
	return &ToxiProxyClient{
		client: toxiproxy.NewClient(host),
	}
}

// ToxiProxyLatencyInjector adds network latency via ToxiProxy
type ToxiProxyLatencyInjector struct {
	name      string
	client    *ToxiProxyClient
	proxyName string
	latency   int // milliseconds
	jitter    int // milliseconds
	toxicName string
	proxy     *toxiproxy.Proxy
	mu        sync.Mutex
	stopped   bool
}

// ToxiProxyLatency creates a latency injector
func ToxiProxyLatency(
	client *ToxiProxyClient,
	proxyName string,
	latency, jitter time.Duration,
) *ToxiProxyLatencyInjector {
	return &ToxiProxyLatencyInjector{
		name:      fmt.Sprintf("toxiproxy_latency_%s_%dms", proxyName, latency.Milliseconds()),
		client:    client,
		proxyName: proxyName,
		latency:   int(latency.Milliseconds()),
		jitter:    int(jitter.Milliseconds()),
		toxicName: fmt.Sprintf("latency_%d", time.Now().Unix()),
	}
}

func (t *ToxiProxyLatencyInjector) Name() string {
	return t.name
}

func (t *ToxiProxyLatencyInjector) Inject(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return fmt.Errorf("injector already stopped")
	}

	// Get the proxy
	proxy, err := t.client.client.Proxy(t.proxyName)
	if err != nil {
		return fmt.Errorf("failed to get proxy %s: %w", t.proxyName, err)
	}
	t.proxy = proxy

	// Add latency toxic
	_, err = proxy.AddToxic(t.toxicName, "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": t.latency,
		"jitter":  t.jitter,
	})
	if err != nil {
		return fmt.Errorf("failed to add latency toxic: %w", err)
	}

	fmt.Printf("[TOXIPROXY] Latency injected on %s: %dms ±%dms\n", t.proxyName, t.latency, t.jitter)

	return nil
}

func (t *ToxiProxyLatencyInjector) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped || t.proxy == nil {
		return nil
	}

	if err := t.proxy.RemoveToxic(t.toxicName); err != nil {
		return fmt.Errorf("failed to remove toxic: %w", err)
	}

	t.stopped = true
	fmt.Printf("[TOXIPROXY] Latency removed from %s\n", t.proxyName)

	return nil
}

// ToxiProxyBandwidthInjector limits bandwidth via ToxiProxy
type ToxiProxyBandwidthInjector struct {
	name      string
	client    *ToxiProxyClient
	proxyName string
	rate      int64 // KB/s
	toxicName string
	proxy     *toxiproxy.Proxy
	mu        sync.Mutex
	stopped   bool
}

// ToxiProxyBandwidth creates a bandwidth limiter injector
func ToxiProxyBandwidth(client *ToxiProxyClient, proxyName string, rateKBps int64) *ToxiProxyBandwidthInjector {
	return &ToxiProxyBandwidthInjector{
		name:      fmt.Sprintf("toxiproxy_bandwidth_%s_%dkbps", proxyName, rateKBps),
		client:    client,
		proxyName: proxyName,
		rate:      rateKBps,
		toxicName: fmt.Sprintf("bandwidth_%d", time.Now().Unix()),
	}
}

func (t *ToxiProxyBandwidthInjector) Name() string {
	return t.name
}

func (t *ToxiProxyBandwidthInjector) Inject(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return fmt.Errorf("injector already stopped")
	}

	proxy, err := t.client.client.Proxy(t.proxyName)
	if err != nil {
		return fmt.Errorf("failed to get proxy %s: %w", t.proxyName, err)
	}
	t.proxy = proxy

	_, err = proxy.AddToxic(t.toxicName, "bandwidth", "downstream", 1.0, toxiproxy.Attributes{
		"rate": t.rate,
	})
	if err != nil {
		return fmt.Errorf("failed to add bandwidth toxic: %w", err)
	}

	fmt.Printf("[TOXIPROXY] Bandwidth limited on %s: %d KB/s\n", t.proxyName, t.rate)

	return nil
}

func (t *ToxiProxyBandwidthInjector) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped || t.proxy == nil {
		return nil
	}

	if err := t.proxy.RemoveToxic(t.toxicName); err != nil {
		return fmt.Errorf("failed to remove toxic: %w", err)
	}

	t.stopped = true
	fmt.Printf("[TOXIPROXY] Bandwidth limit removed from %s\n", t.proxyName)

	return nil
}

// ToxiProxyTimeoutInjector injects connection timeouts
type ToxiProxyTimeoutInjector struct {
	name      string
	client    *ToxiProxyClient
	proxyName string
	timeout   int // milliseconds
	toxicName string
	proxy     *toxiproxy.Proxy
	mu        sync.Mutex
	stopped   bool
}

// ToxiProxyTimeout creates a timeout injector
func ToxiProxyTimeout(client *ToxiProxyClient, proxyName string, timeout time.Duration) *ToxiProxyTimeoutInjector {
	return &ToxiProxyTimeoutInjector{
		name:      fmt.Sprintf("toxiproxy_timeout_%s_%dms", proxyName, timeout.Milliseconds()),
		client:    client,
		proxyName: proxyName,
		timeout:   int(timeout.Milliseconds()),
		toxicName: fmt.Sprintf("timeout_%d", time.Now().Unix()),
	}
}

func (t *ToxiProxyTimeoutInjector) Name() string {
	return t.name
}

func (t *ToxiProxyTimeoutInjector) Inject(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return fmt.Errorf("injector already stopped")
	}

	proxy, err := t.client.client.Proxy(t.proxyName)
	if err != nil {
		return fmt.Errorf("failed to get proxy %s: %w", t.proxyName, err)
	}
	t.proxy = proxy

	_, err = proxy.AddToxic(t.toxicName, "timeout", "downstream", 1.0, toxiproxy.Attributes{
		"timeout": t.timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to add timeout toxic: %w", err)
	}

	fmt.Printf("[TOXIPROXY] Timeout injected on %s: %dms\n", t.proxyName, t.timeout)

	return nil
}

func (t *ToxiProxyTimeoutInjector) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped || t.proxy == nil {
		return nil
	}

	if err := t.proxy.RemoveToxic(t.toxicName); err != nil {
		return fmt.Errorf("failed to remove toxic: %w", err)
	}

	t.stopped = true
	fmt.Printf("[TOXIPROXY] Timeout removed from %s\n", t.proxyName)

	return nil
}

// ToxiProxySlicerInjector creates intermittent connection drops
type ToxiProxySlicerInjector struct {
	name          string
	client        *ToxiProxyClient
	proxyName     string
	averageSize   int // bytes
	sizeVariation int // bytes
	delay         int // microseconds
	toxicName     string
	proxy         *toxiproxy.Proxy
	mu            sync.Mutex
	stopped       bool
}

// ToxiProxySlicer creates a slicer injector (intermittent drops)
func ToxiProxySlicer(
	client *ToxiProxyClient,
	proxyName string,
	avgSize, sizeVar int,
	delay time.Duration,
) *ToxiProxySlicerInjector {
	return &ToxiProxySlicerInjector{
		name:          fmt.Sprintf("toxiproxy_slicer_%s", proxyName),
		client:        client,
		proxyName:     proxyName,
		averageSize:   avgSize,
		sizeVariation: sizeVar,
		delay:         int(delay.Microseconds()),
		toxicName:     fmt.Sprintf("slicer_%d", time.Now().Unix()),
	}
}

func (t *ToxiProxySlicerInjector) Name() string {
	return t.name
}

func (t *ToxiProxySlicerInjector) Inject(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return fmt.Errorf("injector already stopped")
	}

	proxy, err := t.client.client.Proxy(t.proxyName)
	if err != nil {
		return fmt.Errorf("failed to get proxy %s: %w", t.proxyName, err)
	}
	t.proxy = proxy

	_, err = proxy.AddToxic(t.toxicName, "slicer", "downstream", 1.0, toxiproxy.Attributes{
		"average_size":   t.averageSize,
		"size_variation": t.sizeVariation,
		"delay":          t.delay,
	})
	if err != nil {
		return fmt.Errorf("failed to add slicer toxic: %w", err)
	}

	fmt.Printf("[TOXIPROXY] Slicer injected on %s: avg=%d bytes, var=%d bytes, delay=%dµs\n",
		t.proxyName, t.averageSize, t.sizeVariation, t.delay)

	return nil
}

func (t *ToxiProxySlicerInjector) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped || t.proxy == nil {
		return nil
	}

	if err := t.proxy.RemoveToxic(t.toxicName); err != nil {
		return fmt.Errorf("failed to remove toxic: %w", err)
	}

	t.stopped = true
	fmt.Printf("[TOXIPROXY] Slicer removed from %s\n", t.proxyName)

	return nil
}

// ProxyConfig configures a ToxiProxy proxy
type ProxyConfig struct {
	Name     string
	Listen   string
	Upstream string
	Enabled  bool
}

// ToxiProxyManager manages ToxiProxy proxies
type ToxiProxyManager struct {
	client  *ToxiProxyClient
	proxies map[string]*toxiproxy.Proxy
	mu      sync.Mutex
}

// NewToxiProxyManager creates a new proxy manager
func NewToxiProxyManager(client *ToxiProxyClient) *ToxiProxyManager {
	return &ToxiProxyManager{
		client:  client,
		proxies: make(map[string]*toxiproxy.Proxy),
	}
}

// CreateProxy creates a new proxy
func (m *ToxiProxyManager) CreateProxy(cfg ProxyConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxy, err := m.client.client.CreateProxy(cfg.Name, cfg.Listen, cfg.Upstream)
	if err != nil {
		return fmt.Errorf("failed to create proxy %s: %w", cfg.Name, err)
	}

	if cfg.Enabled {
		if err := proxy.Enable(); err != nil {
			return fmt.Errorf("failed to enable proxy %s: %w", cfg.Name, err)
		}
	}

	m.proxies[cfg.Name] = proxy
	fmt.Printf("[TOXIPROXY] Proxy created: %s (%s -> %s)\n", cfg.Name, cfg.Listen, cfg.Upstream)

	return nil
}

// DeleteProxy deletes a proxy
func (m *ToxiProxyManager) DeleteProxy(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxy, exists := m.proxies[name]
	if !exists {
		return fmt.Errorf("proxy %s not found", name)
	}

	if err := proxy.Delete(); err != nil {
		return fmt.Errorf("failed to delete proxy %s: %w", name, err)
	}

	delete(m.proxies, name)
	fmt.Printf("[TOXIPROXY] Proxy deleted: %s\n", name)

	return nil
}

// GetProxy returns a proxy by name
func (m *ToxiProxyManager) GetProxy(name string) (*toxiproxy.Proxy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxy, exists := m.proxies[name]
	if !exists {
		return nil, fmt.Errorf("proxy %s not found", name)
	}

	return proxy, nil
}

// CleanupAll removes all managed proxies
func (m *ToxiProxyManager) CleanupAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, proxy := range m.proxies {
		if err := proxy.Delete(); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete proxy %s: %w", name, err))
		} else {
			fmt.Printf("[TOXIPROXY] Proxy cleaned up: %s\n", name)
		}
	}

	m.proxies = make(map[string]*toxiproxy.Proxy)

	if len(errs) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errs)
	}

	return nil
}

// ListProxies returns all managed proxies
func (m *ToxiProxyManager) ListProxies() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	names := make([]string, 0, len(m.proxies))
	for name := range m.proxies {
		names = append(names, name)
	}

	return names
}
