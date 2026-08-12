package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushCommandStructure(t *testing.T) {
	subcommandNames := make([]string, 0, len(pushCmd.Commands))
	for _, sub := range pushCmd.Commands {
		subcommandNames = append(subcommandNames, sub.Name)
	}

	assert.Contains(t, subcommandNames, "create")
	assert.Contains(t, subcommandNames, "list")
	assert.Contains(t, subcommandNames, "get")

	// The parent action still pushes a local Docker image into hypeman, so it
	// must stay reachable alongside the outbound push subcommands.
	assert.NotNil(t, pushCmd.Action)
}

func TestBuildPushNewParams(t *testing.T) {
	params := buildPushNewParams("alpine:latest", "registry.example.com/alpine:v1", false, "", "", "")

	require.Equal(t, "alpine:latest", params.CreatePushRequest.Image)
	require.Equal(t, "registry.example.com/alpine:v1", params.CreatePushRequest.Target)
	assert.False(t, params.CreatePushRequest.Insecure.Valid())

	// Omitting credentials lets the server use its own registry credentials.
	assert.False(t, params.CreatePushRequest.Credentials.Username.Valid())
	assert.False(t, params.CreatePushRequest.Credentials.Password.Valid())
	assert.False(t, params.CreatePushRequest.Credentials.RegistryToken.Valid())
}

func TestBuildPushNewParamsWithCredentials(t *testing.T) {
	params := buildPushNewParams("alpine:latest", "registry.example.com/alpine:v1", true, "alice", "s3cret", "token")

	require.True(t, params.CreatePushRequest.Insecure.Valid())
	assert.True(t, params.CreatePushRequest.Insecure.Value)

	credentials := params.CreatePushRequest.Credentials
	require.True(t, credentials.Username.Valid())
	assert.Equal(t, "alice", credentials.Username.Value)
	require.True(t, credentials.Password.Valid())
	assert.Equal(t, "s3cret", credentials.Password.Value)
	require.True(t, credentials.RegistryToken.Valid())
	assert.Equal(t, "token", credentials.RegistryToken.Value)
}

func TestBuildPushNewParamsPartialCredentials(t *testing.T) {
	params := buildPushNewParams("alpine:latest", "registry.example.com/alpine:v1", false, "", "", "token")

	credentials := params.CreatePushRequest.Credentials
	assert.False(t, credentials.Username.Valid())
	assert.False(t, credentials.Password.Valid())
	require.True(t, credentials.RegistryToken.Valid())
	assert.Equal(t, "token", credentials.RegistryToken.Value)
}
