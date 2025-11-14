package chaoskit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerdict_String(t *testing.T) {
	tests := []struct {
		name    string
		verdict Verdict
		want    string
	}{
		{"Pass", VerdictPass, "PASS"},
		{"Unstable", VerdictUnstable, "UNSTABLE"},
		{"Fail", VerdictFail, "FAIL"},
		{"Unknown", Verdict(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.verdict.String())
		})
	}
}

func TestVerdict_ExitCode(t *testing.T) {
	assert.Equal(t, 0, VerdictPass.ExitCode())
	assert.Equal(t, 0, VerdictUnstable.ExitCode())
	assert.Equal(t, 1, VerdictFail.ExitCode())
	assert.Equal(t, 1, Verdict(999).ExitCode())
}

func TestValidationSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity ValidationSeverity
		want     string
	}{
		{"Critical", SeverityCritical, "CRITICAL"},
		{"Warning", SeverityWarning, "WARNING"},
		{"Info", SeverityInfo, "INFO"},
		{"Unknown", ValidationSeverity(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.severity.String())
		})
	}
}
