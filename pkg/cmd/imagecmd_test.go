package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildImageNewParams(t *testing.T) {
	params, malformed := buildImageNewParams("docker.io/library/alpine:latest", []string{
		"env=staging",
		"team=cli",
		"missing-delimiter",
	}, "")

	require.Equal(t, "docker.io/library/alpine:latest", params.Name)
	assert.Equal(t, map[string]string{
		"env":  "staging",
		"team": "cli",
	}, params.Tags)
	assert.Equal(t, []string{"missing-delimiter"}, malformed)
	assert.False(t, params.Platform.Valid())
}

func TestBuildImageNewParamsPlatform(t *testing.T) {
	params, malformed := buildImageNewParams("docker.io/library/alpine:latest", nil, "linux/amd64")

	require.Empty(t, malformed)
	require.True(t, params.Platform.Valid())
	assert.Equal(t, "linux/amd64", params.Platform.Value)
}

// TestPlatformOrDash guards the image-list PLATFORM column, which falls back to
// "-" when the server reports no resolved platform.
func TestPlatformOrDash(t *testing.T) {
	assert.Equal(t, "linux/amd64", platformOrDash("linux/amd64"))
	assert.Equal(t, "-", platformOrDash(""))
}

func TestValidateTaggedImageReference(t *testing.T) {
	assert.NoError(t, validateTaggedImageReference("registry.example.com/app:v1"))
	assert.NoError(t, validateTaggedImageReference("nginx:stable"))
	assert.ErrorContains(t, validateTaggedImageReference("registry.example.com/app"), "explicit tag")
	assert.ErrorContains(t, validateTaggedImageReference(
		"registry.example.com/app@sha256:"+strings.Repeat("a", 64)), "explicit tag")
	assert.ErrorContains(t, validateTaggedImageReference("not valid"), "invalid target")
}
