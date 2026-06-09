package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	hypeman "github.com/kernel/hypeman-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func apiErrorWithStatus(code int) *hypeman.Error {
	return &hypeman.Error{Response: &http.Response{StatusCode: code}}
}

func TestIsNotFoundError(t *testing.T) {
	t.Run("404 API error is not found", func(t *testing.T) {
		assert.True(t, isNotFoundError(apiErrorWithStatus(http.StatusNotFound)))
	})

	t.Run("wrapped 404 API error is not found", func(t *testing.T) {
		// errors.As traverses the wrap chain, so an in-flight pull's 404 is still
		// detected even when callers add context with fmt.Errorf.
		err := fmt.Errorf("failed to check image status: %w", apiErrorWithStatus(http.StatusNotFound))
		assert.True(t, isNotFoundError(err))
	})

	t.Run("non-404 API error is not not-found", func(t *testing.T) {
		assert.False(t, isNotFoundError(apiErrorWithStatus(http.StatusInternalServerError)))
	})

	t.Run("API error without response is not not-found", func(t *testing.T) {
		assert.False(t, isNotFoundError(&hypeman.Error{}))
	})

	t.Run("plain error is not not-found", func(t *testing.T) {
		assert.False(t, isNotFoundError(errors.New("boom")))
	})

	t.Run("nil error is not not-found", func(t *testing.T) {
		assert.False(t, isNotFoundError(nil))
	})
}

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
