package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/kernel/hypeman-cli/lib/compose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func runHealthParse(t *testing.T, args ...string) (compose.HealthCheckInput, bool, error) {
	t.Helper()
	var (
		gotIn  compose.HealthCheckInput
		gotSet bool
		gotErr error
	)
	cmd := &cli.Command{
		Name:  "x",
		Flags: healthCheckFlags(""),
		Action: func(_ context.Context, c *cli.Command) error {
			gotIn, gotSet, gotErr = parseHealthCheckInput(c, "")
			return nil
		},
	}
	require.NoError(t, cmd.Run(context.Background(), append([]string{"x"}, args...)))
	return gotIn, gotSet, gotErr
}

func TestParseHealthCheckInput(t *testing.T) {
	t.Run("http flags without http-port are rejected", func(t *testing.T) {
		_, _, err := runHealthParse(t, "--http-path", "/healthz")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "http-port is required")
	})

	t.Run("http-port engages the http probe", func(t *testing.T) {
		in, set, err := runHealthParse(t, "--http-port", "8080", "--http-path", "/healthz")
		require.NoError(t, err)
		require.True(t, set)
		require.NotNil(t, in.HTTP)
		assert.Equal(t, int64(8080), in.HTTP.Port)
		assert.Equal(t, "/healthz", in.HTTP.Path)
	})

	t.Run("exec-working-dir without exec is rejected", func(t *testing.T) {
		_, _, err := runHealthParse(t, "--exec-working-dir", "/srv")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exec is required")
	})

	t.Run("no flags reports unset without error", func(t *testing.T) {
		_, set, err := runHealthParse(t)
		require.NoError(t, err)
		assert.False(t, set)
	})
}

func runExpirationParse(t *testing.T, args ...string) (expirationInput, bool, error) {
	t.Helper()
	var (
		gotIn  expirationInput
		gotSet bool
		gotErr error
	)
	cmd := &cli.Command{
		Name:  "x",
		Flags: expirationFlags(""),
		Action: func(_ context.Context, c *cli.Command) error {
			gotIn, gotSet, gotErr = parseExpirationInput(c, "")
			return nil
		},
	}
	require.NoError(t, cmd.Run(context.Background(), append([]string{"x"}, args...)))
	return gotIn, gotSet, gotErr
}

func TestParseExpirationInput(t *testing.T) {
	t.Run("ttl is passed through verbatim", func(t *testing.T) {
		in, set, err := runExpirationParse(t, "--ttl", "90m")
		require.NoError(t, err)
		require.True(t, set)
		assert.Equal(t, "90m", in.TTL.Value)
		assert.False(t, in.ExpiresAt.Valid())
	})

	t.Run("zero ttl disables expiration", func(t *testing.T) {
		in, set, err := runExpirationParse(t, "--ttl", "0s")
		require.NoError(t, err)
		require.True(t, set)
		assert.Equal(t, "0s", in.TTL.Value)
	})

	t.Run("expires-at is parsed as RFC3339", func(t *testing.T) {
		in, set, err := runExpirationParse(t, "--expires-at", "2026-01-02T15:04:05Z")
		require.NoError(t, err)
		require.True(t, set)
		require.True(t, in.ExpiresAt.Valid())
		assert.Equal(t, time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC), in.ExpiresAt.Value)
		assert.False(t, in.TTL.Valid())
	})

	t.Run("ttl and expires-at are mutually exclusive", func(t *testing.T) {
		_, _, err := runExpirationParse(t, "--ttl", "1h", "--expires-at", "2026-01-02T15:04:05Z")
		require.EqualError(t, err, "--ttl and --expires-at are mutually exclusive")
	})

	t.Run("malformed ttl is rejected", func(t *testing.T) {
		_, _, err := runExpirationParse(t, "--ttl", "2 hours")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid --ttl")
	})

	t.Run("malformed expires-at is rejected", func(t *testing.T) {
		_, _, err := runExpirationParse(t, "--expires-at", "tomorrow")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid --expires-at")
	})

	t.Run("no flags reports unset without error", func(t *testing.T) {
		_, set, err := runExpirationParse(t)
		require.NoError(t, err)
		assert.False(t, set)
	})
}

func runRestartParse(t *testing.T, args ...string) (*int64, bool) {
	t.Helper()
	var (
		gotMax *int64
		gotSet bool
	)
	cmd := &cli.Command{
		Name:  "x",
		Flags: restartPolicyFlags(""),
		Action: func(_ context.Context, c *cli.Command) error {
			in, set := parseRestartPolicyInput(c, "")
			gotMax, gotSet = in.MaxAttempts, set
			return nil
		},
	}
	require.NoError(t, cmd.Run(context.Background(), append([]string{"x"}, args...)))
	return gotMax, gotSet
}

func TestParseRestartPolicyInputMaxAttempts(t *testing.T) {
	t.Run("explicit 0 is sent (unlimited)", func(t *testing.T) {
		max, set := runRestartParse(t, "--max-attempts", "0")
		require.True(t, set)
		require.NotNil(t, max)
		assert.Equal(t, int64(0), *max)
	})

	t.Run("omitted when not provided", func(t *testing.T) {
		max, set := runRestartParse(t, "--policy", "always")
		require.True(t, set)
		assert.Nil(t, max)
	})
}
