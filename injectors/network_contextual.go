package injectors

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

// ContextualNetworkInjector wraps a ToxiProxy injector to provide context-based network chaos
// This allows user code to control when network chaos is applied via MaybeNetworkChaos()
type ContextualNetworkInjector struct {
	name string
	// baseInjector *ToxiProxyLatencyInjector
	manager      *ToxiProxyManager
	proxyConfig  ProxyConfig
	applyRate    float64                // 0.0-1.0 probability of applying chaos
	hostPatterns map[string]NetworkRule // host patterns with specific rules
	mu           sync.RWMutex
	stopped      bool
	rng          *rand.Rand // Deterministic random generator from context
}

// NetworkRule defines chaos rules for specific hosts/ports
type NetworkRule struct {
	Latency         time.Duration
	Jitter          time.Duration
	DropProbability float64
	ApplyRate       float64
}

// NewContextualNetworkInjector creates a network injector that works through context
func NewContextualNetworkInjector(
	client *ToxiProxyClient,
	proxyConfig ProxyConfig,
	applyRate float64,
) *ContextualNetworkInjector {
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
		manager:      NewToxiProxyManager(client),
	}
}

// AddHostRule adds a rule for a specific host pattern
func (c *ContextualNetworkInjector) AddHostRule(hostPattern string, rule NetworkRule) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.hostPatterns[hostPattern] = rule
}

func (c *ContextualNetworkInjector) Name() string {
	return c.name
}

// SetupNetwork implements NetworkInjectorLifecycle
func (c *ContextualNetworkInjector) SetupNetwork(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return fmt.Errorf("injector already stopped")
	}

	// Create proxy if it doesn't exist
	if err := c.manager.CreateProxy(c.proxyConfig); err != nil {
		// Proxy might already exist, try to get it
		if _, getErr := c.manager.GetProxy(c.proxyConfig.Name); getErr != nil {
			return fmt.Errorf("failed to setup network proxy: %w", err)
		}
		// Proxy exists, continue
	}

	fmt.Printf("[CHAOS] Network injector %s setup completed\n", c.name)

	return nil
}

// TeardownNetwork implements NetworkInjectorLifecycle
func (c *ContextualNetworkInjector) TeardownNetwork(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return nil
	}

	if err := c.manager.DeleteProxy(c.proxyConfig.Name); err != nil {
		return fmt.Errorf("failed to teardown network proxy: %w", err)
	}

	fmt.Printf("[CHAOS] Network injector %s teardown completed\n", c.name)

	return nil
}

func (c *ContextualNetworkInjector) Inject(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return fmt.Errorf("injector already stopped")
	}

	// Store deterministic random generator from context
	c.rng = chaoskit.GetRand(ctx)

	fmt.Printf("[CHAOS] Contextual network injector started (apply rate: %.2f)\n", c.applyRate)

	return nil
}

func (c *ContextualNetworkInjector) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.stopped {
		c.stopped = true
		fmt.Printf("[CHAOS] Contextual network injector stopped\n")
	}

	return nil
}

// ShouldApplyNetworkChaos implements ChaosNetworkProvider
func (c *ContextualNetworkInjector) ShouldApplyNetworkChaos(host string, port int) bool {
	c.mu.RLock()
	stopped := c.stopped
	applyRate := c.applyRate
	rng := c.rng
	hostPatterns := make(map[string]NetworkRule)
	for k, v := range c.hostPatterns {
		hostPatterns[k] = v
	}
	c.mu.RUnlock()

	if stopped {
		return false
	}

	// Use stored generator
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	// Check apply rate
	if rng.Float64() >= applyRate {
		return false
	}

	// Check if host matches any pattern
	for pattern, rule := range hostPatterns {
		if matchesHost(pattern, host) {
			return rng.Float64() < rule.ApplyRate
		}
	}

	return true
}

// GetNetworkLatency implements ChaosNetworkProvider
func (c *ContextualNetworkInjector) GetNetworkLatency(host string, port int) (time.Duration, bool) {
	c.mu.RLock()
	stopped := c.stopped
	rng := c.rng
	hostPatterns := make(map[string]NetworkRule)
	for k, v := range c.hostPatterns {
		hostPatterns[k] = v
	}
	c.mu.RUnlock()

	if stopped {
		return 0, false
	}

	// Check host patterns first
	for pattern, rule := range hostPatterns {
		if matchesHost(pattern, host) && rule.Latency > 0 {
			latency := rule.Latency
			if rule.Jitter > 0 {
				jitterMs := int(rule.Jitter.Milliseconds())
				if rng == nil {
					rng = rand.New(rand.NewSource(rand.Int63()))
				}
				latency += time.Duration(rng.Intn(jitterMs*2)-jitterMs) * time.Millisecond
			}

			return latency, true
		}
	}

	// Default latency - use default rule if no base injector
	// This would need to be configured separately
	return 0, false
}

// ShouldDropConnection implements ChaosNetworkProvider
func (c *ContextualNetworkInjector) ShouldDropConnection(host string, port int) bool {
	c.mu.RLock()
	stopped := c.stopped
	rng := c.rng
	hostPatterns := make(map[string]NetworkRule)
	for k, v := range c.hostPatterns {
		hostPatterns[k] = v
	}
	c.mu.RUnlock()

	if stopped {
		return false
	}

	// Check host patterns
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	for pattern, rule := range hostPatterns {
		if matchesHost(pattern, host) {
			return rng.Float64() < rule.DropProbability
		}
	}

	return false
}

// matchesHost checks if host matches pattern (simple wildcard support)
func matchesHost(pattern, host string) bool {
	if pattern == "*" || pattern == host {
		return true
	}

	// Simple suffix matching (e.g., "*.example.com" matches "api.example.com")
	if len(pattern) > 2 && pattern[0] == '*' && pattern[1] == '.' {
		suffix := pattern[2:]

		return len(host) >= len(suffix) && host[len(host)-len(suffix):] == suffix
	}

	// Simple prefix matching (e.g., "api.*" matches "api.example.com")
	if len(pattern) > 2 && pattern[len(pattern)-2:] == ".*" {
		prefix := pattern[:len(pattern)-2]

		return len(host) >= len(prefix) && host[:len(prefix)] == prefix
	}

	return false
}

// Type implements CategorizedInjector
func (c *ContextualNetworkInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid // Works both globally (proxy) and contextually
}

// GetMetrics implements MetricsProvider
func (c *ContextualNetworkInjector) GetMetrics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"apply_rate":    c.applyRate,
		"host_patterns": len(c.hostPatterns),
		"stopped":       c.stopped,
	}
}
