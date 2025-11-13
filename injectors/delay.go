package injectors

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

// DelayMode defines how delays are injected
type DelayMode int

const (
	// ProbabilityMode uses random probability to determine if delay should be applied
	ProbabilityMode DelayMode = iota
	// IntervalMode uses a background goroutine to periodically inject delays that block MaybeDelay() calls
	IntervalMode
)

// DelayInjector introduces random delays during execution
type DelayInjector struct {
	name        string
	minDelay    time.Duration
	maxDelay    time.Duration
	interval    time.Duration
	probability float64 // for ProbabilityMode
	mode        DelayMode

	// For IntervalMode synchronization
	delayCond   *sync.Cond    // condition variable for signaling delays
	activeDelay time.Duration // current delay to apply (protected by delayMu)
	delayMu     sync.Mutex    // mutex for delay state
	mu          sync.Mutex
	stopCh      chan struct{}
	stopped     bool
	delayCount  int64
	rng         *rand.Rand // Deterministic random generator from context
}

// RandomDelay creates a delay injector with probability-based delays (default mode)
// Each call to MaybeDelay() has a chance to apply a delay based on probability
func RandomDelay(min, max time.Duration) *DelayInjector {
	return &DelayInjector{
		name:        fmt.Sprintf("delay_injector_prob_%v_%v", min, max),
		minDelay:    min,
		maxDelay:    max,
		probability: 1.0, // 100% chance by default
		mode:        ProbabilityMode,
		stopCh:      make(chan struct{}),
	}
}

// RandomDelayWithProbability creates a delay injector with probability-based delays
// probability: 0.0 to 1.0, determines the chance of applying delay on each MaybeDelay() call
func RandomDelayWithProbability(min, max time.Duration, probability float64) *DelayInjector {
	if probability < 0 {
		probability = 0
	}
	if probability > 1 {
		probability = 1
	}

	return &DelayInjector{
		name:        fmt.Sprintf("delay_injector_prob_%v_%v_%.2f", min, max, probability),
		minDelay:    min,
		maxDelay:    max,
		probability: probability,
		mode:        ProbabilityMode,
		stopCh:      make(chan struct{}),
	}
}

// RandomDelayWithInterval creates a delay injector with interval-based delays
// The background goroutine periodically injects delays that block MaybeDelay() calls
func RandomDelayWithInterval(min, max, interval time.Duration) *DelayInjector {
	di := &DelayInjector{
		name:     fmt.Sprintf("delay_injector_interval_%v_%v_%v", min, max, interval),
		minDelay: min,
		maxDelay: max,
		interval: interval,
		mode:     IntervalMode,
		stopCh:   make(chan struct{}),
	}
	di.delayCond = sync.NewCond(&di.delayMu)

	return di
}

func (d *DelayInjector) Name() string {
	return d.name
}

func (d *DelayInjector) Inject(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return fmt.Errorf("injector already stopped")
	}

	// Store deterministic random generator from context
	d.rng = chaoskit.GetRand(ctx)

	if d.mode == IntervalMode {
		slog.Info("delay injector started",
			slog.String("injector", d.name),
			slog.String("mode", "interval"),
			slog.Duration("min_delay", d.minDelay),
			slog.Duration("max_delay", d.maxDelay),
			slog.Duration("interval", d.interval))
		// Start a background goroutine that periodically injects delays
		go d.delayLoop(ctx)
	} else {
		slog.Info("delay injector started",
			slog.String("injector", d.name),
			slog.String("mode", "probability"),
			slog.Duration("min_delay", d.minDelay),
			slog.Duration("max_delay", d.maxDelay),
			slog.Float64("probability", d.probability))
	}

	return nil
}

func (d *DelayInjector) delayLoop(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Calculate delay and activate it
			// This will make all MaybeDelay() calls block and apply this delay
			delay := d.calculateDelay()
			if delay > 0 {
				slog.Debug("interval-based delay triggered",
					slog.String("injector", d.name),
					slog.Duration("delay", delay))

				// Set active delay and broadcast to all waiting goroutines
				d.delayMu.Lock()
				d.activeDelay = delay
				d.delayCond.Broadcast() // Wake up all waiting MaybeDelay() calls
				d.delayMu.Unlock()

				// Wait for a delay window to close (give time for user code to apply it)
				time.Sleep(delay + 200*time.Millisecond)

				// Reset active delay if not consumed
				d.delayMu.Lock()
				if d.activeDelay > 0 {
					d.activeDelay = 0
					slog.Debug("interval delay window closed",
						slog.String("injector", d.name))
				}
				d.delayMu.Unlock()
			}
		}
	}
}

func (d *DelayInjector) calculateDelay() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return 0
	}

	if d.maxDelay <= d.minDelay {
		return d.minDelay
	}

	delta := d.maxDelay - d.minDelay
	rng := d.rng
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}
	delay := d.minDelay + time.Duration(rng.Int63n(int64(delta)))

	return delay
}

func (d *DelayInjector) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.stopped {
		close(d.stopCh)
		d.stopped = true
		slog.Info("delay injector stopped",
			slog.String("injector", d.name),
			slog.Int64("total_delays", d.delayCount))
	}

	return nil
}

// GetDelayCount returns the number of delays injected
func (d *DelayInjector) GetDelayCount() int64 {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.delayCount
}

// BeforeStep injects a delay before step execution
func (d *DelayInjector) BeforeStep(ctx context.Context) error {
	if d.stopped {
		return nil
	}

	delay := d.calculateDelay()
	if delay > 0 {
		d.mu.Lock()
		d.delayCount++
		count := d.delayCount
		d.mu.Unlock()

		slog.Debug("injecting delay before step",
			slog.String("injector", d.name),
			slog.Int64("step_count", count),
			slog.Duration("delay", delay))
		time.Sleep(delay)
	}

	return nil
}

// AfterStep is called after step execution (no-op for delay injector)
func (d *DelayInjector) AfterStep(ctx context.Context, err error) error {
	return nil
}

// GetChaosDelay returns a delay for chaos context
// In ProbabilityMode: returns delay based on random probability
// In IntervalMode: blocks until background goroutine signals a delay should be applied
func (d *DelayInjector) GetChaosDelay() (time.Duration, bool) {
	d.mu.Lock()
	stopped := d.stopped
	mode := d.mode
	probability := d.probability
	d.mu.Unlock()

	if stopped {
		return 0, false
	}

	if mode == IntervalMode {
		// Wait for delay signal from background goroutine
		d.delayMu.Lock()

		// Check if delay is already active
		if d.activeDelay > 0 {
			delay := d.activeDelay
			d.activeDelay = 0 // Consume delay
			d.delayMu.Unlock()

			// Increment counter when delay is actually applied
			d.mu.Lock()
			d.delayCount++
			d.mu.Unlock()

			slog.Debug("interval delay applied in user code",
				slog.String("injector", d.name),
				slog.Duration("delay", delay))

			return delay, true
		}

		// Wait for a delay to be triggered with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		delayReceived := make(chan time.Duration, 1)

		// Goroutine to wait for delay
		go func() {
			d.delayMu.Lock()
			defer d.delayMu.Unlock()

			// Handle timeout
			go func() {
				<-ctx.Done()
				d.delayCond.Signal() // Wake up to check timeout
			}()

			for d.activeDelay == 0 {
				// Check if context timed out
				if ctx.Err() != nil {
					delayReceived <- 0

					return
				}

				// Wait for broadcast from delayLoop
				d.delayCond.Wait()

				if d.activeDelay > 0 {
					delay := d.activeDelay
					d.activeDelay = 0 // Consume delay

					// Increment counter when delay is actually applied
					d.mu.Lock()
					d.delayCount++
					d.mu.Unlock()

					delayReceived <- delay

					return
				}
			}
		}()

		d.delayMu.Unlock()

		// Wait for result
		select {
		case delay := <-delayReceived:
			if delay > 0 {
				slog.Debug("interval delay applied in user code",
					slog.String("injector", d.name),
					slog.Duration("delay", delay))

				return delay, true
			}

			return 0, false
		case <-ctx.Done():
			return 0, false
		}
	}

	// ProbabilityMode: use random generator
	d.mu.Lock()
	rng := d.rng
	d.mu.Unlock()
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	if rng.Float64() < probability {
		delay := d.calculateDelay()
		if delay > 0 {
			d.mu.Lock()
			d.delayCount++
			d.mu.Unlock()

			return delay, true
		}
	}

	return 0, false
}

// Type implements CategorizedInjector
func (d *DelayInjector) Type() chaoskit.InjectorType {
	if d.mode == IntervalMode {
		return chaoskit.InjectorTypeHybrid // Can work both globally and contextually
	}

	return chaoskit.InjectorTypeContext
}

// GetMetrics implements MetricsProvider
func (d *DelayInjector) GetMetrics() map[string]interface{} {
	d.mu.Lock()
	defer d.mu.Unlock()

	return map[string]interface{}{
		"mode":        d.mode.String(),
		"min_delay":   d.minDelay.String(),
		"max_delay":   d.maxDelay.String(),
		"probability": d.probability,
		"interval":    d.interval.String(),
		"delay_count": d.delayCount,
		"stopped":     d.stopped,
	}
}

// String returns string representation of DelayMode
func (m DelayMode) String() string {
	switch m {
	case ProbabilityMode:
		return "probability"
	case IntervalMode:
		return "interval"
	default:
		return "unknown"
	}
}
