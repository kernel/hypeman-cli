package cmd

import (
	"testing"

	"github.com/kernel/hypeman-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatGPU(t *testing.T) {
	tests := []struct {
		name     string
		gpu      hypeman.InstanceGPU
		expected string
	}{
		{
			name:     "no GPU",
			gpu:      hypeman.InstanceGPU{},
			expected: "-",
		},
		{
			name: "vGPU with profile",
			gpu: hypeman.InstanceGPU{
				Profile:  "L40S-1Q",
				MdevUuid: "abc-123",
			},
			expected: "L40S-1Q",
		},
		{
			name: "vGPU without profile but with mdev",
			gpu: hypeman.InstanceGPU{
				MdevUuid: "abc-123",
			},
			expected: "vgpu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatGPU(tt.gpu)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatHypervisor(t *testing.T) {
	tests := []struct {
		name       string
		hypervisor hypeman.InstanceHypervisor
		expected   string
	}{
		{
			name:       "cloud-hypervisor",
			hypervisor: hypeman.InstanceHypervisorCloudHypervisor,
			expected:   "ch",
		},
		{
			name:       "qemu",
			hypervisor: hypeman.InstanceHypervisorQemu,
			expected:   "qemu",
		},
		{
			name:       "firecracker",
			hypervisor: hypeman.InstanceHypervisorFirecracker,
			expected:   "fc",
		},
		{
			name:       "empty defaults to ch",
			hypervisor: "",
			expected:   "ch",
		},
		{
			name:       "unknown value",
			hypervisor: hypeman.InstanceHypervisor("unknown"),
			expected:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatHypervisor(tt.hypervisor)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseMetadataFilters(t *testing.T) {
	t.Run("parses valid entries", func(t *testing.T) {
		metadata, malformed := parseMetadataFilters([]string{
			"team=backend",
			"env=staging",
		})

		require.Empty(t, malformed)
		assert.Equal(t, map[string]string{
			"team": "backend",
			"env":  "staging",
		}, metadata)
	})

	t.Run("returns malformed entries and only valid metadata", func(t *testing.T) {
		metadata, malformed := parseMetadataFilters([]string{
			"team=backend",
			"missing-delimiter",
			"=empty-key",
			"region=us-east-1",
		})

		assert.Equal(t, map[string]string{
			"team":   "backend",
			"region": "us-east-1",
		}, metadata)
		assert.Equal(t, []string{
			"missing-delimiter",
			"=empty-key",
		}, malformed)
	})
}
