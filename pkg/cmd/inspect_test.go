package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestRedactEnvValues(t *testing.T) {
	t.Run("returns payload unchanged when env absent", func(t *testing.T) {
		raw := `{"id":"i1","platform":"linux/amd64"}`
		assert.Equal(t, raw, redactEnvValues(raw))
	})

	t.Run("redacts all values and preserves keys", func(t *testing.T) {
		raw := `{"id":"i1","env":{"FOO":"bar","BAZ":"qux"}}`

		redacted := redactEnvValues(raw)

		assert.Equal(t, "[hidden]", gjson.Get(redacted, "env.FOO").String())
		assert.Equal(t, "[hidden]", gjson.Get(redacted, "env.BAZ").String())
		assert.True(t, gjson.Get(redacted, "env.FOO").Exists())
		assert.True(t, gjson.Get(redacted, "env.BAZ").Exists())
		// Non-env fields are untouched.
		assert.Equal(t, "i1", gjson.Get(redacted, "id").String())
	})

	t.Run("handles env keys containing dots", func(t *testing.T) {
		raw := `{"env":{"my.var":"secret"}}`

		redacted := redactEnvValues(raw)

		assert.Equal(t, "[hidden]", gjson.Get(redacted, `env.my\.var`).String())
	})

	t.Run("handles env keys containing sjson special chars", func(t *testing.T) {
		// '*', '?', '|', '#', '@' are all special to sjson path parsing.
		raw := `{"env":{"A*B":"s1","C?D":"s2","E|F":"s3","G#H":"s4","I@J":"s5"}}`

		redacted := redactEnvValues(raw)

		for _, key := range []string{`env.A\*B`, `env.C\?D`, `env.E\|F`, `env.G\#H`, `env.I\@J`} {
			assert.Equal(t, "[hidden]", gjson.Get(redacted, key).String(), "key %s", key)
		}
	})
}

// TestRedactEnvValuesPreservesPlatform guards the N4 fix: rendering from the raw
// server payload must keep fields (like platform) that the SDK's typed Instance
// model predates and would otherwise drop.
func TestRedactEnvValuesPreservesPlatform(t *testing.T) {
	raw := `{"id":"i1","platform":"linux/amd64","env":{"FOO":"bar"}}`

	redacted := redactEnvValues(raw)

	assert.Equal(t, "linux/amd64", gjson.Get(redacted, "platform").String())
	assert.Equal(t, "[hidden]", gjson.Get(redacted, "env.FOO").String())
}
