package injectors

import (
	"context"
	"testing"
	"time"
)

func TestContextualNetwork_BasicLogic_NoExternalDeps(t *testing.T) {
	client := NewToxiProxyClient("http://localhost:8474") // won't be contacted in this test
	cfg := ProxyConfig{Name: "test", Listen: "localhost:0", Upstream: "localhost:80"}
	c := NewContextualNetworkInjector(client, cfg, 1.0) // always apply if matches

	if c.Name() == "" {
		t.Fatalf("expected non-empty name")
	}

	// Add rule for host pattern
	c.AddHostRule("api.*", NetworkRule{Latency: 5 * time.Millisecond, Jitter: 0, DropProbability: 1.0, ApplyRate: 1.0})

	// With applyRate=1 and matching rule ApplyRate=1, ShouldApplyNetworkChaos returns true
	if !c.ShouldApplyNetworkChaos("api.example.com", 443) {
		t.Fatalf("expected to apply network chaos for matching host")
	}
	if c.ShouldApplyNetworkChaos("unmatched.host", 80) == false {
		// with no matching pattern, default behavior is true when applyRate=1
		t.Fatalf("expected to apply network chaos for unmatched host with applyRate=1")
	}

	// Latency should be provided for matching host
	if d, ok := c.GetNetworkLatency("api.example.com", 443); !ok || d != 5*time.Millisecond {
		t.Fatalf("expected 5ms latency for matching host, got %v %v", d, ok)
	}

	// Drop should be true for matching host because DropProbability=1
	if !c.ShouldDropConnection("api.example.com", 443) {
		t.Fatalf("expected drop= true for matching host")
	}

	// Stop should disable logic
	_ = c.Stop(context.Background())
	if c.ShouldApplyNetworkChaos("api.example.com", 443) {
		t.Fatalf("expected no apply after stop")
	}
	if d, ok := c.GetNetworkLatency("api.example.com", 443); ok || d != 0 {
		t.Fatalf("expected no latency after stop")
	}
}

func TestMatchesHost(t *testing.T) {
	cases := []struct {
		pattern, host string
		want          bool
	}{
		{"*", "anything", true},
		{"example.com", "example.com", true},
		{"example.com", "api.example.com", false},
		{"*.example.com", "api.example.com", true},
		{"api.*", "api.example.com", true},
		{"api.*", "xapi.example.com", false},
	}
	for _, c := range cases {
		if got := matchesHost(c.pattern, c.host); got != c.want {
			t.Fatalf("pattern %q host %q: want %v got %v", c.pattern, c.host, c.want, got)
		}
	}
}
