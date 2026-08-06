package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilderCommandStructure(t *testing.T) {
	assert.Equal(t, "builder", builderCmd.Name)
	assert.Contains(t, builderCmd.Aliases, "builders")

	subcommandNames := make([]string, len(builderCmd.Commands))
	for i, cmd := range builderCmd.Commands {
		subcommandNames[i] = cmd.Name
	}

	assert.Contains(t, subcommandNames, "create")
	assert.Contains(t, subcommandNames, "list")
	assert.Contains(t, subcommandNames, "get")
	assert.Contains(t, subcommandNames, "delete")
	assert.Contains(t, subcommandNames, "prune")
}

func TestBuilderCreateCmdFlags(t *testing.T) {
	flagNames := make([]string, 0)
	for _, flag := range builderCreateCmd.Flags {
		flagNames = append(flagNames, flag.Names()...)
	}

	assert.Contains(t, flagNames, "id")
	assert.Contains(t, flagNames, "name")
	assert.Contains(t, flagNames, "disk-size")
	assert.Contains(t, flagNames, "tag")
}

func TestBuildCmdHasBuilderFlag(t *testing.T) {
	flagNames := make([]string, 0)
	for _, flag := range buildCmd.Flags {
		flagNames = append(flagNames, flag.Names()...)
	}

	assert.Contains(t, flagNames, "builder")
	assert.Contains(t, flagNames, "builder-id")
}
