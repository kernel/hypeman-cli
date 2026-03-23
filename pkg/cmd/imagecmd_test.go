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
