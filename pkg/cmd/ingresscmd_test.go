package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestRedactIngressAuthValues(t *testing.T) {
	t.Run("redacts get response", func(t *testing.T) {
		raw := `{"id":"ing-1","rules":[{"request_header_auth":{"header":"X-Origin-Verification","value":"secret-value"}},{"target":{"port":8080}}]}`
		redacted := redactIngressAuthValues(raw)

		assert.Equal(t, "[hidden]", gjson.Get(redacted, "rules.0.request_header_auth.value").String())
		assert.Equal(t, "X-Origin-Verification", gjson.Get(redacted, "rules.0.request_header_auth.header").String())
	})

	t.Run("redacts list response", func(t *testing.T) {
		raw := `[{"id":"ing-1","rules":[{"request_header_auth":{"value":"first-secret"}}]},{"id":"ing-2","rules":[{"request_header_auth":{"value":"second-secret"}}]}]`
		redacted := redactIngressAuthValues(raw)

		assert.Equal(t, "[hidden]", gjson.Get(redacted, "0.rules.0.request_header_auth.value").String())
		assert.Equal(t, "[hidden]", gjson.Get(redacted, "1.rules.0.request_header_auth.value").String())
	})

	t.Run("preserves response without authorization values", func(t *testing.T) {
		raw := `{"id":"ing-1","rules":[{"target":{"port":8080}}]}`
		assert.Equal(t, raw, redactIngressAuthValues(raw))
	})
}

func TestParseIngressRuleSpec(t *testing.T) {
	t.Run("full spec with host port, tls, and redirect", func(t *testing.T) {
		rule, err := parseIngressRuleSpec("api.example.com:443=web:8080,tls,redirect-http", "fallback")
		require.NoError(t, err)
		assert.Equal(t, "api.example.com", rule.Match.Hostname)
		assert.Equal(t, int64(443), rule.Match.Port.Value)
		assert.Equal(t, "web", rule.Target.Instance)
		assert.Equal(t, int64(8080), rule.Target.Port)
		assert.True(t, rule.Tls.Value)
		assert.True(t, rule.RedirectHTTP.Value)
	})

	t.Run("defaults host port to 80 when omitted", func(t *testing.T) {
		rule, err := parseIngressRuleSpec("api.example.com=web:8080", "fallback")
		require.NoError(t, err)
		assert.Equal(t, int64(80), rule.Match.Port.Value)
		assert.False(t, rule.Tls.Valid())
		assert.False(t, rule.RedirectHTTP.Valid())
	})

	t.Run("falls back to positional instance when omitted", func(t *testing.T) {
		rule, err := parseIngressRuleSpec("api.example.com:80=:8080", "fallback")
		require.NoError(t, err)
		assert.Equal(t, "fallback", rule.Target.Instance)
		assert.Equal(t, int64(8080), rule.Target.Port)
	})

	t.Run("rejects missing target separator", func(t *testing.T) {
		_, err := parseIngressRuleSpec("api.example.com:80", "fallback")
		require.EqualError(t, err, "expected format hostname[:host-port]=instance:port[,tls][,redirect-http]")
	})

	t.Run("rejects empty hostname", func(t *testing.T) {
		_, err := parseIngressRuleSpec("=web:8080", "fallback")
		require.EqualError(t, err, "hostname cannot be empty")
	})

	t.Run("rejects target without port", func(t *testing.T) {
		_, err := parseIngressRuleSpec("api.example.com=web", "fallback")
		require.EqualError(t, err, "target must be instance:port")
	})

	t.Run("rejects non-numeric target port", func(t *testing.T) {
		_, err := parseIngressRuleSpec("api.example.com=web:http", "fallback")
		require.ErrorContains(t, err, `invalid target port "http"`)
	})

	t.Run("rejects unknown option", func(t *testing.T) {
		_, err := parseIngressRuleSpec("api.example.com=web:8080,gzip", "fallback")
		require.EqualError(t, err, `unknown option "gzip"`)
	})
}
