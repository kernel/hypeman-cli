package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildImageNewParams(t *testing.T) {
	params, malformed := buildImageNewParams("docker.io/library/alpine:latest", []string{
		"env=staging",
		"team=cli",
		"missing-delimiter",
	})

	require.Equal(t, "docker.io/library/alpine:latest", params.Name)
	assert.Equal(t, map[string]string{
		"env":  "staging",
		"team": "cli",
	}, params.Tags)
	assert.Equal(t, []string{"missing-delimiter"}, malformed)
}

// TestPlatformFromRaw guards the N4 image-list PLATFORM column: the value is read
// from the raw server payload (the SDK model predates the field) and falls back
// to "-" when absent.
func TestPlatformFromRaw(t *testing.T) {
	assert.Equal(t, "linux/amd64", platformFromRaw(`{"name":"alpine","platform":"linux/amd64"}`))
	assert.Equal(t, "-", platformFromRaw(`{"name":"alpine"}`))
	assert.Equal(t, "-", platformFromRaw(`{"name":"alpine","platform":""}`))
	assert.Equal(t, "-", platformFromRaw(""))
}
