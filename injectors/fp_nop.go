//go:build !failpoint

package injectors

// This file provides no-op stubs when building without -tags failpoint.
// It allows the project to compile and run even if the gofail runtime is not included.

func enableFailpoint(name, action string) error { return ErrFailpointDisabled }
func disableFailpoint(name string) error        { return ErrFailpointDisabled }
