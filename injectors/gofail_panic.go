package injectors

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// GofailPanicInjector toggles named failpoints with panic actions.
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
type GofailPanicInjector struct {
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

// GofailPanic creates a new gofail-based panic injector.
// names: list of failpoint names placed in the target code via failpoint.Inject.
// probability: per-tick probability to enable each failpoint.
// window: how long an enabled failpoint remains active before auto-disable. Also used as the tick interval.
func GofailPanic(names []string, probability float64, window time.Duration) *GofailPanicInjector {
	return &GofailPanicInjector{
		name:        fmt.Sprintf("gofail_panic_%d_pts_p%.2f", len(names), probability),
		failpoints:  append([]string(nil), names...),
		probability: probability,
		interval:    window,
		window:      window,
		stopCh:      make(chan struct{}),
		active:      make(map[string]bool),
	}
}

func (g *GofailPanicInjector) Name() string { return g.name }

func (g *GofailPanicInjector) Inject(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.stopped {
		return fmt.Errorf("injector already stopped")
	}

	// Probe runtime availability (distinguish missing build tag).
	if err := enableFailpoint("chaoskit_runtime_probe", `panic("probe")`); errors.Is(err, ErrFailpointDisabled) {
		return ErrFailpointDisabled
	} else if err == nil {
		_ = disableFailpoint("chaoskit_runtime_probe")
	}

	// Start background toggler
	interval := g.interval
	if interval <= 0 {
		interval = 250 * time.Millisecond
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-g.stopCh:
				return
			case <-ticker.C:
				g.tickOnce()
			}
		}
	}()

	return nil
}

func (g *GofailPanicInjector) tickOnce() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, fp := range g.failpoints {
		if rand.Float64() < g.probability && !g.active[fp] {
			// Enable panic action for this failpoint for a window
			action := fmt.Sprintf(`panic("chaoskit gofail: %s")`, fp)
			if err := enableFailpoint(fp, action); err == nil {
				g.active[fp] = true
				win := g.window
				if win <= 0 {
					win = 250 * time.Millisecond
				}
				time.AfterFunc(win, func() {
					g.mu.Lock()
					defer g.mu.Unlock()
					_ = disableFailpoint(fp)
					g.active[fp] = false
				})
			}
		}
	}
}

func (g *GofailPanicInjector) Stop(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.stopped {
		close(g.stopCh)
		g.stopped = true
		for fp, en := range g.active {
			if en {
				_ = disableFailpoint(fp)
				g.active[fp] = false
			}
		}
	}

	return nil
}
