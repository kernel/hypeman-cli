package cmd

import (
	"testing"

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
