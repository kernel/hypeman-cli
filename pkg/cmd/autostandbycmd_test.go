package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoStandbyCommandStructure(t *testing.T) {
	assert.Equal(t, "auto-standby", autoStandbyCmd.Name)

	subcommandNames := make([]string, len(autoStandbyCmd.Commands))
	for i, cmd := range autoStandbyCmd.Commands {
		subcommandNames[i] = cmd.Name
	}

	assert.Contains(t, subcommandNames, "status")
	assert.Contains(t, subcommandNames, "hold")
}

func TestAutoStandbyHoldCmdStructure(t *testing.T) {
	assert.Equal(t, "hold", autoStandbyHoldCmd.Name)
	assert.Equal(t, "<instance>", autoStandbyHoldCmd.ArgsUsage)
	require.NotNil(t, autoStandbyHoldCmd.Action)
}
