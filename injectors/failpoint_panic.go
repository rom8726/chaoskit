package injectors

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

// FailpointPanicInjector toggles named failpoints with panic actions.
// To use it you must:
//  1. Instrument your target code with failpoint.Inject("name", func(){ panic("...") }) blocks
//  2. Build and run with `-tags failpoint`
//  3. Optionally `go get github.com/pingcap/failpoint` in your project
//
// The injector will enable selected failpoints for a short window with given probability on each tick.
// If the failpoint runtime is not enabled, Inject will return ErrFailpointDisabled.
//
// Note: enabling a failpoint means that any code path hitting it during the enable-window will panic.
// Choose probability and window cautiously.
type FailpointPanicInjector struct {
	name        string
	failpoints  []string
	probability float64
	interval    time.Duration
	window      time.Duration

	mu      sync.Mutex
	stopCh  chan struct{}
	stopped bool
	active  map[string]bool
}

// FailpointPanic creates a new failpoint-based panic injector.
// names: list of failpoint names placed in the target code via failpoint.Inject.
// probability: per-tick probability to enable each failpoint.
// window: how long an enabled failpoint remains active before auto-disable. Also used as the tick interval.
func FailpointPanic(names []string, probability float64, window time.Duration) *FailpointPanicInjector {
	return &FailpointPanicInjector{
		name:        fmt.Sprintf("failpoint_panic_%d_pts_p%.2f", len(names), probability),
		failpoints:  append([]string(nil), names...),
		probability: probability,
		interval:    window,
		window:      window,
		stopCh:      make(chan struct{}),
		active:      make(map[string]bool),
	}
}

func (f *FailpointPanicInjector) Name() string { return f.name }

func (f *FailpointPanicInjector) Inject(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.stopped {
		return fmt.Errorf("injector already stopped")
	}

	// Probe runtime availability (distinguish missing build tag).
	if err := enableFailpoint("chaoskit_runtime_probe", `panic("probe")`); errors.Is(err, ErrFailpointDisabled) {
		return ErrFailpointDisabled
	} else if err == nil {
		_ = disableFailpoint("chaoskit_runtime_probe")
	}

	// Start background toggler
	interval := f.interval
	if interval <= 0 {
		interval = 250 * time.Millisecond
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-f.stopCh:
				return
			case <-ticker.C:
				f.tickOnce()
			}
		}
	}()

	return nil
}

func (f *FailpointPanicInjector) tickOnce() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, fp := range f.failpoints {
		if rand.Float64() < f.probability && !f.active[fp] {
			// Enable panic action for this failpoint for a window
			action := fmt.Sprintf(`panic("chaoskit failpoint: %s")`, fp)
			if err := enableFailpoint(fp, action); err == nil {
				f.active[fp] = true
				win := f.window
				if win <= 0 {
					win = 250 * time.Millisecond
				}
				time.AfterFunc(win, func() {
					f.mu.Lock()
					defer f.mu.Unlock()
					_ = disableFailpoint(fp)
					f.active[fp] = false
				})
			}
		}
	}
}

func (f *FailpointPanicInjector) Stop(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.stopped {
		close(f.stopCh)
		f.stopped = true
		for fp, en := range f.active {
			if en {
				_ = disableFailpoint(fp)
				f.active[fp] = false
			}
		}
	}

	return nil
}

// Type implements CategorizedInjector
func (f *FailpointPanicInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeGlobal // Works globally via failpoint runtime
}

// GetMetrics implements MetricsProvider
func (f *FailpointPanicInjector) GetMetrics() map[string]interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	activeCount := 0
	for _, active := range f.active {
		if active {
			activeCount++
		}
	}

	return map[string]interface{}{
		"failpoints":       f.failpoints,
		"probability":      f.probability,
		"interval":         f.interval.String(),
		"window":           f.window.String(),
		"active_count":     activeCount,
		"total_failpoints": len(f.failpoints),
		"stopped":          f.stopped,
	}
}
