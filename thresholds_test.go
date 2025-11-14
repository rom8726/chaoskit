package chaoskit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuccessThresholds_Validate(t *testing.T) {
	tests := []struct {
		name       string
		thresholds *SuccessThresholds
		wantErr    bool
	}{
		{
			name:       "valid default",
			thresholds: DefaultThresholds(),
			wantErr:    false,
		},
		{
			name: "invalid success rate too high",
			thresholds: &SuccessThresholds{
				MinSuccessRate: 1.5,
			},
			wantErr: true,
		},
		{
			name: "invalid success rate too low",
			thresholds: &SuccessThresholds{
				MinSuccessRate: -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid max failed iterations",
			thresholds: &SuccessThresholds{
				MinSuccessRate:      0.95,
				MaxFailedIterations: -1,
			},
			wantErr: true,
		},
		{
			name:       "valid strict thresholds",
			thresholds: StrictThresholds(),
			wantErr:    false,
		},
		{
			name:       "valid relaxed thresholds",
			thresholds: RelaxedThresholds(),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.thresholds.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultThresholds(t *testing.T) {
	thresholds := DefaultThresholds()
	assert.Equal(t, 0.95, thresholds.MinSuccessRate)
	assert.Contains(t, thresholds.CriticalValidators, "goroutine-limit")
	assert.Contains(t, thresholds.CriticalValidators, "recursion-depth")
	assert.Contains(t, thresholds.CriticalValidators, "slow-iteration")
	assert.Contains(t, thresholds.CriticalValidators, "memory-limit")
}

func TestStrictThresholds(t *testing.T) {
	thresholds := StrictThresholds()
	assert.Equal(t, 1.0, thresholds.MinSuccessRate)
	assert.True(t, thresholds.RequireAllValidatorsPassing)
	assert.Contains(t, thresholds.CriticalValidators, "panic-recovery")
}

func TestRelaxedThresholds(t *testing.T) {
	thresholds := RelaxedThresholds()
	assert.Equal(t, 0.80, thresholds.MinSuccessRate)
	assert.Equal(t, 200, thresholds.MaxFailedIterations)
	assert.Contains(t, thresholds.CriticalValidators, "goroutine-limit")
}
