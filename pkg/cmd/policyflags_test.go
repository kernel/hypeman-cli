package cmd

import (
	"context"
	"testing"

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
