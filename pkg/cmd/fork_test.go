package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeForkTargetState(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		shouldErr bool
	}{
		{
			name:     "empty state",
			input:    "",
			expected: "",
		},
		{
			name:     "stopped lowercase",
			input:    "stopped",
			expected: "Stopped",
		},
		{
			name:     "standby mixed case",
			input:    "StAnDbY",
			expected: "Standby",
		},
		{
			name:     "running title case",
			input:    "Running",
			expected: "Running",
		},
		{
			name:      "invalid state",
			input:     "paused",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeForkTargetState(tt.input)
			if tt.shouldErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
