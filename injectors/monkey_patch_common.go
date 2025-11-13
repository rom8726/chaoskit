package injectors

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"

	"github.com/rom8726/chaoskit"
)

// PatchHandle represents a handle to a patched function
type PatchHandle struct {
	Func        interface{}   // Pointer to function
	Original    reflect.Value // Original function value
	RestoreFunc func()        // Function to restore original
	Patched     bool          // Whether currently patched
	Data        interface{}   // Optional additional data (e.g., counters)
}

// PatchManager manages a collection of patches
type PatchManager struct {
	patches []PatchHandle
	mu      sync.Mutex
}

// NewPatchManager creates a new patch manager
func NewPatchManager() *PatchManager {
	return &PatchManager{
		patches: make([]PatchHandle, 0),
	}
}

// ValidateFunction validates that the target is a valid function pointer
func ValidateFunction(funcPtr interface{}) error {
	if funcPtr == nil {
		return fmt.Errorf("function is nil")
	}

	funcVal := reflect.ValueOf(funcPtr)
	if funcVal.Kind() != reflect.Ptr {
		return fmt.Errorf("function must be a pointer, got %v", funcVal.Kind())
	}

	elem := funcVal.Elem()
	if elem.Kind() != reflect.Func {
		return fmt.Errorf("target must be a function pointer, got %v", elem.Kind())
	}

	return nil
}

// CreatePatch creates a patch handle for a function
func CreatePatch(funcPtr interface{}) (PatchHandle, error) {
	if err := ValidateFunction(funcPtr); err != nil {
		return PatchHandle{}, err
	}

	funcVal := reflect.ValueOf(funcPtr)
	elem := funcVal.Elem()

	// Store original function value
	original := elem.Interface()
	originalVal := reflect.ValueOf(original)

	return PatchHandle{
		Func:        funcPtr,
		Original:    originalVal,
		RestoreFunc: nil, // Will be set when applying patch
		Patched:     false,
	}, nil
}

// ApplyPatch applies a replacement function to a patch handle
// replacementFunc is called with original args and should return modified results
func ApplyPatch(handle *PatchHandle, replacementFunc func(args []reflect.Value) []reflect.Value) error {
	funcVal := reflect.ValueOf(handle.Func)
	elem := funcVal.Elem()

	originalType := reflect.TypeOf(handle.Original.Interface())
	originalNumIn := originalType.NumIn()
	originalNumOut := originalType.NumOut()

	// Create replacement function type
	inTypes := make([]reflect.Type, originalNumIn)
	for i := 0; i < originalNumIn; i++ {
		inTypes[i] = originalType.In(i)
	}

	outTypes := make([]reflect.Type, originalNumOut)
	for i := 0; i < originalNumOut; i++ {
		outTypes[i] = originalType.Out(i)
	}

	replacementType := reflect.FuncOf(inTypes, outTypes, originalType.IsVariadic())

	// Store original for restoration
	originalCopy := handle.Original

	// Create replacement implementation
	replacement := reflect.MakeFunc(replacementType, func(args []reflect.Value) []reflect.Value {
		// Call replacement function which should handle the logic
		return replacementFunc(args)
	})

	// Create restore function
	handle.RestoreFunc = func() {
		funcVal := reflect.ValueOf(handle.Func)
		elem := funcVal.Elem()
		elem.Set(originalCopy)
	}

	// Apply patch: replace function with replacement
	elem.Set(replacement)
	handle.Patched = true

	return nil
}

// RestorePatch restores a patched function to its original state
func RestorePatch(handle *PatchHandle) error {
	if !handle.Patched {
		return nil
	}

	if handle.RestoreFunc != nil {
		handle.RestoreFunc()
		handle.Patched = false
	}

	return nil
}

// GetFuncName returns the name of a function for logging
func GetFuncName(funcPtr interface{}, customName string) string {
	if customName != "" {
		return customName
	}

	funcType := reflect.TypeOf(funcPtr)
	if funcType.Kind() == reflect.Ptr {
		funcType = funcType.Elem()
	}

	return funcType.String()
}

// ValidateProbability validates that probability is in valid range
func ValidateProbability(probability float64) error {
	if probability < 0 || probability > 1 {
		return fmt.Errorf("probability must be between 0.0 and 1.0, got %.2f", probability)
	}

	return nil
}

// AddPatch adds a patch to the manager
func (pm *PatchManager) AddPatch(handle PatchHandle) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.patches = append(pm.patches, handle)
}

// RestoreAllPatches restores all patches in the manager
func (pm *PatchManager) RestoreAllPatches(ctx context.Context, onRestore func(handle PatchHandle) string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for i := range pm.patches {
		if pm.patches[i].Patched {
			name := ""
			if onRestore != nil {
				name = onRestore(pm.patches[i])
			}

			if err := RestorePatch(&pm.patches[i]); err == nil && name != "" {
				chaoskit.GetLogger(ctx).Debug("monkey patch restored",
					slog.String("function", name))
			}
		}
	}
}

// RollbackPatches rolls back patches up to a certain index
func (pm *PatchManager) RollbackPatches(upTo int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for i := 0; i < upTo && i < len(pm.patches); i++ {
		if pm.patches[i].Patched {
			_ = RestorePatch(&pm.patches[i])
		}
	}
}

// GetActivePatchCount returns the number of active patches
func (pm *PatchManager) GetActivePatchCount() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	count := 0
	for _, patch := range pm.patches {
		if patch.Patched {
			count++
		}
	}

	return count
}

// GetPatches returns all patches
func (pm *PatchManager) GetPatches() []PatchHandle {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.patches
}
