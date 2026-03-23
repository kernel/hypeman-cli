package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildNetworkEgress(t *testing.T) {
	t.Run("defaults enabled to true when mode is set", func(t *testing.T) {
		egress, err := buildNetworkEgress(false, false, "all")
		require.NoError(t, err)
		require.True(t, egress.Enabled.Valid())
		assert.True(t, egress.Enabled.Value)
		assert.Equal(t, "all", egress.Enforcement.Mode)
	})

	t.Run("honors explicit disabled flag when mode is set", func(t *testing.T) {
		egress, err := buildNetworkEgress(false, true, "http_https_only")
		require.NoError(t, err)
		require.True(t, egress.Enabled.Valid())
		assert.False(t, egress.Enabled.Value)
		assert.Equal(t, "http_https_only", egress.Enforcement.Mode)
	})

	t.Run("rejects unsupported modes", func(t *testing.T) {
		_, err := buildNetworkEgress(false, false, "smtp_only")
		require.EqualError(t, err, "invalid network-egress-mode: smtp_only (must be 'all' or 'http_https_only')")
	})
}
