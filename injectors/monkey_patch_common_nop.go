//go:build !chaos

package injectors

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// PatchHandle mirrors the real structure but performs no modifications.
type PatchHandle struct {
	Func        interface{}
	Original    reflect.Value
	RestoreFunc func()
	Patched     bool
	Data        interface{}
}

// PatchManager manages patch handles in stub builds.
type PatchManager struct {
	mu      sync.Mutex
	patches []PatchHandle
}

// NewPatchManager returns an empty manager.
func NewPatchManager() *PatchManager {
	return &PatchManager{
		patches: make([]PatchHandle, 0),
	}
}

// ValidateFunction performs basic validation equivalent to the real implementation.
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

// CreatePatch prepares a patch handle without applying any runtime patching.
func CreatePatch(funcPtr interface{}) (PatchHandle, error) {
	if err := ValidateFunction(funcPtr); err != nil {
		return PatchHandle{}, err
	}

	funcVal := reflect.ValueOf(funcPtr)
	elem := funcVal.Elem()

	original := elem.Interface()
	originalVal := reflect.ValueOf(original)

	return PatchHandle{
		Func:     funcPtr,
		Original: originalVal,
		Patched:  false,
	}, nil
}

// ApplyPatch marks the handle as patched but does not alter the target function.
func ApplyPatch(handle *PatchHandle, _ func(args []reflect.Value) []reflect.Value) error {
	if handle == nil {
		return nil
	}

	handle.RestoreFunc = func() {}
	handle.Patched = true

	return nil
}

// RestorePatch clears the patched flag.
func RestorePatch(handle *PatchHandle) error {
	if handle == nil || !handle.Patched {
		return nil
	}

	if handle.RestoreFunc != nil {
		handle.RestoreFunc()
	}

	handle.Patched = false

	return nil
}

// GetFuncName returns the name of a function for logging.
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

// ValidateProbability matches the logic of the real implementation.
func ValidateProbability(probability float64) error {
	if probability < 0 || probability > 1 {
		return fmt.Errorf("probability must be between 0.0 and 1.0, got %.2f", probability)
	}

	return nil
}

// AddPatch records a patch handle.
func (pm *PatchManager) AddPatch(handle PatchHandle) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.patches = append(pm.patches, handle)
}

// RestoreAllPatches iterates over known patches and resets their state.
func (pm *PatchManager) RestoreAllPatches(ctx context.Context, onRestore func(handle PatchHandle) string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	_ = ctx

	for i := range pm.patches {
		if pm.patches[i].Patched {
			_ = RestorePatch(&pm.patches[i])

			if onRestore != nil {
				_ = onRestore(pm.patches[i])
			}
		}
	}
}

// RollbackPatches clears the patched flag for the first upTo patches.
func (pm *PatchManager) RollbackPatches(upTo int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for i := 0; i < upTo && i < len(pm.patches); i++ {
		_ = RestorePatch(&pm.patches[i])
	}
}

// GetActivePatchCount returns the number of handles marked as patched.
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

// GetPatches returns a copy of the recorded handles.
func (pm *PatchManager) GetPatches() []PatchHandle {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	result := make([]PatchHandle, len(pm.patches))
	copy(result, pm.patches)

	return result
}
