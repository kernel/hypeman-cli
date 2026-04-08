package cmd

import (
	"testing"

	"github.com/kernel/hypeman-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInstanceWaitState(t *testing.T) {
	t.Run("accepts mixed-case state names", func(t *testing.T) {
		state, err := parseInstanceWaitState("rUnNiNg")
		require.NoError(t, err)
		assert.Equal(t, hypeman.InstanceWaitParamsStateRunning, state)
	})

	t.Run("rejects unsupported state names", func(t *testing.T) {
		_, err := parseInstanceWaitState("booting")
		require.EqualError(t, err, "invalid state: booting (must be Created, Initializing, Running, Paused, Shutdown, Stopped, Standby, or Unknown)")
	})
}

func TestParseAutoStandbyPorts(t *testing.T) {
	t.Run("parses valid port values", func(t *testing.T) {
		ports, err := parseAutoStandbyPorts([]string{"80", " 443 "}, "ignore-destination-port")
		require.NoError(t, err)
		assert.Equal(t, []int64{80, 443}, ports)
	})

	t.Run("rejects out-of-range ports", func(t *testing.T) {
		_, err := parseAutoStandbyPorts([]string{"70000"}, "ignore-destination-port")
		require.EqualError(t, err, `ignore-destination-port must be between 1 and 65535: "70000"`)
	})
}
