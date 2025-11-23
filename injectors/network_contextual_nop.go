//go:build !chaos

package injectors

import (
	"context"
	"fmt"
	"time"

	"github.com/rom8726/chaoskit"
)

// ContextualNetworkInjector is a no-op placeholder compiled without the chaos tag.
type ContextualNetworkInjector struct {
	name         string
	proxyConfig  ProxyConfig
	applyRate    float64
	hostPatterns map[string]NetworkRule
	stopped      bool
}

// NetworkRule mirrors the real configuration structure.
type NetworkRule struct {
	Latency         time.Duration
	Jitter          time.Duration
	DropProbability float64
	ApplyRate       float64
}

// NewContextualNetworkInjector returns a stub injector with the same naming semantics.
func NewContextualNetworkInjector(
	client *ToxiProxyClient,
	proxyConfig ProxyConfig,
	applyRate float64,
) *ContextualNetworkInjector {
	_ = client

	if applyRate < 0 {
		applyRate = 0
	}
	if applyRate > 1 {
		applyRate = 1
	}

	return &ContextualNetworkInjector{
		name:         fmt.Sprintf("contextual_network_%s", proxyConfig.Name),
		proxyConfig:  proxyConfig,
		applyRate:    applyRate,
		hostPatterns: make(map[string]NetworkRule),
	}
}

// AddHostRule stores a host-specific rule.
func (c *ContextualNetworkInjector) AddHostRule(hostPattern string, rule NetworkRule) {
	c.hostPatterns[hostPattern] = rule
}

// Name returns the configured name.
func (c *ContextualNetworkInjector) Name() string {
	return c.name
}

// SetupNetwork is a no-op.
func (*ContextualNetworkInjector) SetupNetwork(context.Context) error {
	return nil
}

// TeardownNetwork is a no-op.
func (*ContextualNetworkInjector) TeardownNetwork(context.Context) error {
	return nil
}

// Inject is a no-op.
func (*ContextualNetworkInjector) Inject(context.Context) error {
	return nil
}

// Stop marks the stub as stopped.
func (c *ContextualNetworkInjector) Stop(context.Context) error {
	c.stopped = true

	return nil
}

// ShouldApplyNetworkChaos indicates that no chaos should be applied in stub builds.
func (*ContextualNetworkInjector) ShouldApplyNetworkChaos(string, int) bool {
	return false
}

// GetNetworkLatency always reports no latency injection.
func (*ContextualNetworkInjector) GetNetworkLatency(string, int) (time.Duration, bool) {
	return 0, false
}

// ShouldDropConnection always returns false.
func (*ContextualNetworkInjector) ShouldDropConnection(string, int) bool {
	return false
}

// Type reports the injector classification.
func (*ContextualNetworkInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid
}

// GetMetrics returns stub metrics matching the real structure.
func (c *ContextualNetworkInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"apply_rate":    c.applyRate,
		"host_patterns": len(c.hostPatterns),
		"stopped":       c.stopped,
	}
}
