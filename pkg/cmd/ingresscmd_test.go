package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	t.Run("parses request header auth option", func(t *testing.T) {
		rule, err := parseIngressRuleSpec("api.example.com=web:8080,tls,request-header-auth=X-Ingress-Token:s3cret", "fallback")
		require.NoError(t, err)
		assert.Equal(t, "X-Ingress-Token", rule.RequestHeaderAuth.Header)
		assert.Equal(t, "s3cret", rule.RequestHeaderAuth.Value)
	})

	t.Run("keeps colons in the request header auth value", func(t *testing.T) {
		rule, err := parseIngressRuleSpec("api.example.com=web:8080,request-header-auth=X-Token:user:pass", "fallback")
		require.NoError(t, err)
		assert.Equal(t, "X-Token", rule.RequestHeaderAuth.Header)
		assert.Equal(t, "user:pass", rule.RequestHeaderAuth.Value)
	})

	t.Run("rejects request header auth without a value", func(t *testing.T) {
		_, err := parseIngressRuleSpec("api.example.com=web:8080,request-header-auth=X-Token", "fallback")
		require.EqualError(t, err, `request header auth requires a value for header "X-Token"`)
	})

	t.Run("rejects empty request header auth", func(t *testing.T) {
		_, err := parseIngressRuleSpec("api.example.com=web:8080,request-header-auth=", "fallback")
		require.EqualError(t, err, "request-header-auth must be HEADER:VALUE")
	})

	t.Run("rejects missing target separator", func(t *testing.T) {
		_, err := parseIngressRuleSpec("api.example.com:80", "fallback")
		require.EqualError(t, err, "expected format "+ingressRuleSpecFormat)
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

func TestRequestHeaderAuthFromFlags(t *testing.T) {
	t.Run("returns nil when neither half is set", func(t *testing.T) {
		auth, err := requestHeaderAuthFromFlags("", "")
		require.NoError(t, err)
		assert.Nil(t, auth)
	})

	t.Run("builds the param when both halves are set", func(t *testing.T) {
		auth, err := requestHeaderAuthFromFlags("X-Ingress-Token", "s3cret")
		require.NoError(t, err)
		require.NotNil(t, auth)
		assert.Equal(t, "X-Ingress-Token", auth.Header)
		assert.Equal(t, "s3cret", auth.Value)
	})

	t.Run("rejects a value without a header", func(t *testing.T) {
		_, err := requestHeaderAuthFromFlags("", "s3cret")
		require.EqualError(t, err, "request header auth requires a header name")
	})

	t.Run("rejects a header without a value", func(t *testing.T) {
		_, err := requestHeaderAuthFromFlags("X-Ingress-Token", "")
		require.EqualError(t, err, `request header auth requires a value for header "X-Ingress-Token"`)
	})
}
