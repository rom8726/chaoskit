package injectors

import (
	"reflect"
	"testing"
)

// Test functions for testing
var (
	testFuncSimple = func() int {
		return 42
	}

	testFuncWithArgs = func(a, b int) int {
		return a + b
	}

	testFuncWithError = func() (int, error) {
		return 10, nil
	}
)

func TestValidateFunction(t *testing.T) {
	tests := []struct {
		name      string
		funcPtr   interface{}
		wantError bool
	}{
		{
			name:      "valid function pointer",
			funcPtr:   &testFuncSimple,
			wantError: false,
		},
		{
			name:      "nil function",
			funcPtr:   nil,
			wantError: true,
		},
		{
			name:      "not a pointer",
			funcPtr:   testFuncSimple,
			wantError: true,
		},
		{
			name:      "pointer to non-function",
			funcPtr:   &[]int{1, 2, 3},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFunction(tt.funcPtr)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateFunction() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateProbability(t *testing.T) {
	tests := []struct {
		name      string
		prob      float64
		wantError bool
	}{
		{"valid 0.0", 0.0, false},
		{"valid 0.5", 0.5, false},
		{"valid 1.0", 1.0, false},
		{"invalid negative", -0.1, true},
		{"invalid > 1", 1.1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProbability(tt.prob)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateProbability() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCreatePatch(t *testing.T) {
	// Valid case
	handle, err := CreatePatch(&testFuncSimple)
	if err != nil {
		t.Fatalf("CreatePatch() error = %v", err)
	}

	if handle.Func != &testFuncSimple {
		t.Error("CreatePatch() Func mismatch")
	}

	if !handle.Original.IsValid() {
		t.Error("CreatePatch() Original is not valid")
	}

	if handle.Patched {
		t.Error("CreatePatch() Patched should be false initially")
	}

	// Invalid case
	_, err = CreatePatch(nil)
	if err == nil {
		t.Error("CreatePatch() should fail for nil")
	}
}

func TestApplyPatch(t *testing.T) {
	handle, err := CreatePatch(&testFuncSimple)
	if err != nil {
		t.Fatalf("CreatePatch() error = %v", err)
	}

	called := false
	err = ApplyPatch(&handle, func(args []reflect.Value) []reflect.Value {
		called = true
		return handle.Original.Call(args)
	})
	if err != nil {
		t.Fatalf("ApplyPatch() error = %v", err)
	}

	if !handle.Patched {
		t.Error("ApplyPatch() Patched should be true after applying")
	}

	if handle.RestoreFunc == nil {
		t.Error("ApplyPatch() RestoreFunc should be set")
	}

	// Test that patch works
	result := testFuncSimple()
	if result != 42 {
		t.Errorf("Patched function returned %d, want 42", result)
	}
	if !called {
		t.Error("Replacement function was not called")
	}
}

func TestRestorePatch(t *testing.T) {
	handle, err := CreatePatch(&testFuncSimple)
	if err != nil {
		t.Fatalf("CreatePatch() error = %v", err)
	}

	originalValue := testFuncSimple()

	err = ApplyPatch(&handle, func(args []reflect.Value) []reflect.Value {
		// Modify behavior
		return []reflect.Value{reflect.ValueOf(99)}
	})
	if err != nil {
		t.Fatalf("ApplyPatch() error = %v", err)
	}

	// Verify patch is applied
	if testFuncSimple() != 99 {
		t.Error("Patch not applied correctly")
	}

	// Restore
	err = RestorePatch(&handle)
	if err != nil {
		t.Fatalf("RestorePatch() error = %v", err)
	}

	if handle.Patched {
		t.Error("RestorePatch() Patched should be false after restore")
	}

	// Verify original behavior restored
	if testFuncSimple() != originalValue {
		t.Errorf("Restored function returned %d, want %d", testFuncSimple(), originalValue)
	}
}

func TestGetFuncName(t *testing.T) {
	name := GetFuncName(&testFuncSimple, "")
	if name == "" {
		t.Error("GetFuncName() should return non-empty string")
	}

	customName := "myFunction"
	name = GetFuncName(&testFuncSimple, customName)
	if name != customName {
		t.Errorf("GetFuncName() = %v, want %v", name, customName)
	}
}

func TestPatchManager(t *testing.T) {
	pm := NewPatchManager()

	// Add patches
	handle1, _ := CreatePatch(&testFuncSimple)
	handle2, _ := CreatePatch(&testFuncWithArgs)

	pm.AddPatch(handle1)
	pm.AddPatch(handle2)

	if pm.GetActivePatchCount() != 0 {
		t.Errorf("GetActivePatchCount() = %d, want 0", pm.GetActivePatchCount())
	}

	// Apply patches
	ApplyPatch(&handle1, func(args []reflect.Value) []reflect.Value {
		return handle1.Original.Call(args)
	})
	ApplyPatch(&handle2, func(args []reflect.Value) []reflect.Value {
		return handle2.Original.Call(args)
	})

	pm.AddPatch(handle1)
	pm.AddPatch(handle2)

	if pm.GetActivePatchCount() != 2 {
		t.Errorf("GetActivePatchCount() = %d, want 2", pm.GetActivePatchCount())
	}

	// Restore all
	pm.RestoreAllPatches(func(handle PatchHandle) string {
		return "test"
	})

	if pm.GetActivePatchCount() != 0 {
		t.Errorf("GetActivePatchCount() after restore = %d, want 0", pm.GetActivePatchCount())
	}
}

func TestPatchManagerRollback(t *testing.T) {
	pm := NewPatchManager()

	handles := make([]PatchHandle, 3)
	for i := 0; i < 3; i++ {
		var testFunc = func() int { return i }
		handle, _ := CreatePatch(&testFunc)
		ApplyPatch(&handle, func(args []reflect.Value) []reflect.Value {
			return handle.Original.Call(args)
		})
		handles[i] = handle
		pm.AddPatch(handle)
	}

	if pm.GetActivePatchCount() != 3 {
		t.Errorf("GetActivePatchCount() = %d, want 3", pm.GetActivePatchCount())
	}

	pm.RollbackPatches(2)

	if pm.GetActivePatchCount() != 1 {
		t.Errorf("GetActivePatchCount() after rollback = %d, want 1", pm.GetActivePatchCount())
	}
}
