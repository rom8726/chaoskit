package injectors

import "testing"

func TestFailpointStubs_ReturnDisabledError(t *testing.T) {
	if err := enableFailpoint("test/fp", "return(true)"); err == nil || err != ErrFailpointDisabled {
		t.Fatalf("expected ErrFailpointDisabled from enable, got %v", err)
	}
	if err := disableFailpoint("test/fp"); err == nil || err != ErrFailpointDisabled {
		t.Fatalf("expected ErrFailpointDisabled from disable, got %v", err)
	}
}
