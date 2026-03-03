package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactEnvValues(t *testing.T) {
	t.Run("returns empty map as-is", func(t *testing.T) {
		var env map[string]string
		assert.Nil(t, redactEnvValues(env))

		empty := map[string]string{}
		assert.Equal(t, empty, redactEnvValues(empty))
	})

	t.Run("redacts all values and preserves keys", func(t *testing.T) {
		env := map[string]string{
			"FOO": "bar",
			"BAZ": "qux",
		}

		redacted := redactEnvValues(env)

		assert.Equal(t, map[string]string{
			"FOO": "[hidden]",
			"BAZ": "[hidden]",
		}, redacted)
		require.Equal(t, "bar", env["FOO"])
		require.Equal(t, "qux", env["BAZ"])
	})
}
