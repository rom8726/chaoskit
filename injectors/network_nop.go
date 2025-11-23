//go:build !chaos

package injectors

import (
	"context"
	"fmt"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"

	"github.com/rom8726/chaoskit"
)

// ToxiProxyClient is a minimal placeholder that preserves the public API.
type ToxiProxyClient struct {
	host string
}

// NewToxiProxyClient returns a stub client.
func NewToxiProxyClient(host string) *ToxiProxyClient {
	return &ToxiProxyClient{host: host}
}

// ToxiProxyLatencyInjector is a no-op placeholder compiled without the chaos tag.
type ToxiProxyLatencyInjector struct {
	name      string
	proxyName string
	latency   time.Duration
	jitter    time.Duration
}

// ToxiProxyLatency returns a stub latency injector.
func ToxiProxyLatency(
	client *ToxiProxyClient,
	proxyName string,
	latency, jitter time.Duration,
) *ToxiProxyLatencyInjector {
	_ = client

	return &ToxiProxyLatencyInjector{
		name:      fmt.Sprintf("toxiproxy_latency_%s_%dms", proxyName, latency.Milliseconds()),
		proxyName: proxyName,
		latency:   latency,
		jitter:    jitter,
	}
}

// Name returns the configured name.
func (t *ToxiProxyLatencyInjector) Name() string {
	return t.name
}

// Inject is a no-op.
func (*ToxiProxyLatencyInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*ToxiProxyLatencyInjector) Stop(context.Context) error {
	return nil
}

// ToxiProxyBandwidthInjector is a no-op placeholder.
type ToxiProxyBandwidthInjector struct {
	name      string
	proxyName string
	rate      int64
}

// ToxiProxyBandwidth returns a stub bandwidth injector.
func ToxiProxyBandwidth(client *ToxiProxyClient, proxyName string, rateKBps int64) *ToxiProxyBandwidthInjector {
	_ = client

	return &ToxiProxyBandwidthInjector{
		name:      fmt.Sprintf("toxiproxy_bandwidth_%s_%dkbps", proxyName, rateKBps),
		proxyName: proxyName,
		rate:      rateKBps,
	}
}

// Name returns the configured name.
func (t *ToxiProxyBandwidthInjector) Name() string {
	return t.name
}

// Inject is a no-op.
func (*ToxiProxyBandwidthInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*ToxiProxyBandwidthInjector) Stop(context.Context) error {
	return nil
}

// ToxiProxyTimeoutInjector is a no-op placeholder.
type ToxiProxyTimeoutInjector struct {
	name      string
	proxyName string
	timeout   time.Duration
}

// ToxiProxyTimeout returns a stub timeout injector.
func ToxiProxyTimeout(client *ToxiProxyClient, proxyName string, timeout time.Duration) *ToxiProxyTimeoutInjector {
	_ = client

	return &ToxiProxyTimeoutInjector{
		name:      fmt.Sprintf("toxiproxy_timeout_%s_%dms", proxyName, timeout.Milliseconds()),
		proxyName: proxyName,
		timeout:   timeout,
	}
}

// Name returns the configured name.
func (t *ToxiProxyTimeoutInjector) Name() string {
	return t.name
}

// Inject is a no-op.
func (*ToxiProxyTimeoutInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*ToxiProxyTimeoutInjector) Stop(context.Context) error {
	return nil
}

// ToxiProxySlicerInjector is a no-op placeholder.
type ToxiProxySlicerInjector struct {
	name          string
	proxyName     string
	averageSize   int
	sizeVariation int
	delay         time.Duration
}

// ToxiProxySlicer returns a stub slicer injector.
func ToxiProxySlicer(
	client *ToxiProxyClient,
	proxyName string,
	avgSize, sizeVar int,
	delay time.Duration,
) *ToxiProxySlicerInjector {
	_ = client

	return &ToxiProxySlicerInjector{
		name:          fmt.Sprintf("toxiproxy_slicer_%s", proxyName),
		proxyName:     proxyName,
		averageSize:   avgSize,
		sizeVariation: sizeVar,
		delay:         delay,
	}
}

// Name returns the configured name.
func (t *ToxiProxySlicerInjector) Name() string {
	return t.name
}

// Inject is a no-op.
func (*ToxiProxySlicerInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*ToxiProxySlicerInjector) Stop(context.Context) error {
	return nil
}

// ProxyConfig mirrors the real configuration structure.
type ProxyConfig struct {
	Name     string
	Listen   string
	Upstream string
	Enabled  bool
}

// ToxiProxyManager manages proxy metadata for stub builds.
type ToxiProxyManager struct {
	client  *ToxiProxyClient
	proxies map[string]*toxiproxy.Proxy
}

// NewToxiProxyManager returns a stub manager.
func NewToxiProxyManager(client *ToxiProxyClient) *ToxiProxyManager {
	return &ToxiProxyManager{
		client:  client,
		proxies: make(map[string]*toxiproxy.Proxy),
	}
}

// CreateProxy records proxy metadata without touching ToxiProxy.
func (m *ToxiProxyManager) CreateProxy(cfg ProxyConfig) error {
	m.proxies[cfg.Name] = nil

	return nil
}

// DeleteProxy removes recorded metadata.
func (m *ToxiProxyManager) DeleteProxy(name string) error {
	delete(m.proxies, name)

	return nil
}

// GetProxy returns recorded metadata (always nil in stub builds).
func (m *ToxiProxyManager) GetProxy(name string) (*toxiproxy.Proxy, error) {
	proxy, ok := m.proxies[name]
	if !ok {
		return nil, fmt.Errorf("proxy %s not found", name)
	}

	return proxy, nil
}

// CleanupAll clears recorded metadata.
func (m *ToxiProxyManager) CleanupAll() error {
	m.proxies = make(map[string]*toxiproxy.Proxy)

	return nil
}

// ListProxies lists recorded proxy names.
func (m *ToxiProxyManager) ListProxies() []string {
	names := make([]string, 0, len(m.proxies))
	for name := range m.proxies {
		names = append(names, name)
	}

	return names
}

// Type implements CategorizedInjector for network injectors where needed.
func (*ToxiProxyLatencyInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeGlobal
}

// Type implements CategorizedInjector for network injectors where needed.
func (*ToxiProxyBandwidthInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeGlobal
}

// Type implements CategorizedInjector for network injectors where needed.
func (*ToxiProxyTimeoutInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeGlobal
}

// Type implements CategorizedInjector for network injectors where needed.
func (*ToxiProxySlicerInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeGlobal
}
