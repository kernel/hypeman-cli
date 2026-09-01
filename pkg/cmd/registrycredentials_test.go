package cmd

import (
	"context"
	"testing"

	"github.com/kernel/hypeman-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestBuildRegistryCredentials(t *testing.T) {
	t.Run("reports nothing supplied when all values are empty", func(t *testing.T) {
		credentials, supplied := buildRegistryCredentials("", "", "")

		assert.False(t, supplied)
		assert.False(t, credentials.Username.Valid())
		assert.False(t, credentials.Password.Valid())
		assert.False(t, credentials.RegistryToken.Valid())
	})

	t.Run("sets only the values that were supplied", func(t *testing.T) {
		credentials, supplied := buildRegistryCredentials("", "", "token")

		require.True(t, supplied)
		assert.False(t, credentials.Username.Valid())
		assert.False(t, credentials.Password.Valid())
		require.True(t, credentials.RegistryToken.Valid())
		assert.Equal(t, "token", credentials.RegistryToken.Value)
	})

	t.Run("sets every value", func(t *testing.T) {
		credentials, supplied := buildRegistryCredentials("alice", "s3cret", "token")

		require.True(t, supplied)
		require.True(t, credentials.Username.Valid())
		assert.Equal(t, "alice", credentials.Username.Value)
		require.True(t, credentials.Password.Valid())
		assert.Equal(t, "s3cret", credentials.Password.Value)
		require.True(t, credentials.RegistryToken.Valid())
		assert.Equal(t, "token", credentials.RegistryToken.Value)
	})
}

// TestImageCreateFlagsCarryRegistryCredentials covers the wiring between the
// shared image-create flags and ImageNewParams.Credentials, which lets
// `hypeman pull` and `hypeman image create` pull from a private registry.
func TestImageCreateFlagsCarryRegistryCredentials(t *testing.T) {
	var params hypeman.ImageNewParams

	command := &cli.Command{
		Name:  "create",
		Flags: imageCreateFlags(),
		Action: func(_ context.Context, cmd *cli.Command) error {
			params, _ = buildImageNewParams(cmd.Args().First(), nil, "")
			if credentials, ok := registryCredentialsFromCommand(cmd); ok {
				params.Credentials = credentials
			}
			return nil
		},
	}

	require.NoError(t, command.Run(context.Background(), []string{
		"create", "--username", "alice", "--password", "s3cret", "alpine:latest",
	}))

	require.Equal(t, "alpine:latest", params.Name)
	require.True(t, params.Credentials.Username.Valid())
	assert.Equal(t, "alice", params.Credentials.Username.Value)
	require.True(t, params.Credentials.Password.Valid())
	assert.Equal(t, "s3cret", params.Credentials.Password.Value)
	assert.False(t, params.Credentials.RegistryToken.Valid())
}
