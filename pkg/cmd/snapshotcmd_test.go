package cmd

import (
	"testing"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSnapshotCompressionAlgorithm(t *testing.T) {
	t.Run("accepts mixed-case zstd", func(t *testing.T) {
		algorithm, err := parseSnapshotCompressionAlgorithm("ZsTd")
		require.NoError(t, err)
		assert.Equal(t, shared.SnapshotCompressionConfigAlgorithmZstd, algorithm)
	})

	t.Run("accepts mixed-case lz4", func(t *testing.T) {
		algorithm, err := parseSnapshotCompressionAlgorithm("LZ4")
		require.NoError(t, err)
		assert.Equal(t, shared.SnapshotCompressionConfigAlgorithmLz4, algorithm)
	})

	t.Run("rejects unsupported algorithms", func(t *testing.T) {
		_, err := parseSnapshotCompressionAlgorithm("gzip")
		require.EqualError(t, err, "invalid compression algorithm: gzip (must be 'zstd' or 'lz4')")
	})
}

func TestParseSnapshotTargetHypervisor(t *testing.T) {
	tests := []struct {
		raw      string
		expected hypeman.InstanceSnapshotRestoreParamsTargetHypervisor
	}{
		{"cloud-hypervisor", hypeman.InstanceSnapshotRestoreParamsTargetHypervisorCloudHypervisor},
		{"ch", hypeman.InstanceSnapshotRestoreParamsTargetHypervisorCloudHypervisor},
		{"firecracker", hypeman.InstanceSnapshotRestoreParamsTargetHypervisorFirecracker},
		{"fc", hypeman.InstanceSnapshotRestoreParamsTargetHypervisorFirecracker},
		{"qemu", hypeman.InstanceSnapshotRestoreParamsTargetHypervisorQemu},
		{"qemu-microvm", hypeman.InstanceSnapshotRestoreParamsTargetHypervisorQemuMicrovm},
		{"MicroVM", hypeman.InstanceSnapshotRestoreParamsTargetHypervisorQemuMicrovm},
		{"vz", hypeman.InstanceSnapshotRestoreParamsTargetHypervisorVz},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			hypervisor, err := parseSnapshotTargetHypervisor(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, hypervisor)
		})
	}

	t.Run("rejects unsupported hypervisors", func(t *testing.T) {
		_, err := parseSnapshotTargetHypervisor("xen")
		require.EqualError(t, err, "invalid target hypervisor: xen (must be cloud-hypervisor, firecracker, qemu, qemu-microvm, or vz)")
	})
}

func TestParseSnapshotForkTargetHypervisor(t *testing.T) {
	tests := []struct {
		raw      string
		expected hypeman.SnapshotForkParamsTargetHypervisor
	}{
		{"cloud-hypervisor", hypeman.SnapshotForkParamsTargetHypervisorCloudHypervisor},
		{"ch", hypeman.SnapshotForkParamsTargetHypervisorCloudHypervisor},
		{"firecracker", hypeman.SnapshotForkParamsTargetHypervisorFirecracker},
		{"fc", hypeman.SnapshotForkParamsTargetHypervisorFirecracker},
		{"qemu", hypeman.SnapshotForkParamsTargetHypervisorQemu},
		{"qemu-microvm", hypeman.SnapshotForkParamsTargetHypervisorQemuMicrovm},
		{"MicroVM", hypeman.SnapshotForkParamsTargetHypervisorQemuMicrovm},
		{"vz", hypeman.SnapshotForkParamsTargetHypervisorVz},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			hypervisor, err := parseSnapshotForkTargetHypervisor(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, hypervisor)
		})
	}

	t.Run("rejects unsupported hypervisors", func(t *testing.T) {
		_, err := parseSnapshotForkTargetHypervisor("xen")
		require.EqualError(t, err, "invalid target hypervisor: xen (must be cloud-hypervisor, firecracker, qemu, qemu-microvm, or vz)")
	})
}
