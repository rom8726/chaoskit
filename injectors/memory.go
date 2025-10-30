package injectors

import (
	"context"
	"fmt"
	"sync"
)

// MemoryPressureInjector allocates memory to create pressure
type MemoryPressureInjector struct {
	name      string
	sizeMB    int
	mu        sync.Mutex
	stopCh    chan struct{}
	stopped   bool
	allocated [][]byte
}

// MemoryPressure creates a memory pressure injector
func MemoryPressure(sizeMB int) *MemoryPressureInjector {
	return &MemoryPressureInjector{
		name:   fmt.Sprintf("memory_pressure_%dMB", sizeMB),
		sizeMB: sizeMB,
		stopCh: make(chan struct{}),
	}
}

func (m *MemoryPressureInjector) Name() string {
	return m.name
}

func (m *MemoryPressureInjector) Inject(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return fmt.Errorf("injector already stopped")
	}

	// Allocate memory in chunks
	chunkSize := 1024 * 1024 // 1MB chunks
	for i := 0; i < m.sizeMB; i++ {
		chunk := make([]byte, chunkSize)
		// Write to ensure allocation
		for j := range chunk {
			chunk[j] = byte(j % 256)
		}
		m.allocated = append(m.allocated, chunk)
	}

	fmt.Printf("[CHAOS] Memory pressure injected: %d MB allocated\n", m.sizeMB)

	return nil
}

func (m *MemoryPressureInjector) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stopped {
		close(m.stopCh)
		m.stopped = true
		m.allocated = nil // Release memory
	}

	return nil
}
