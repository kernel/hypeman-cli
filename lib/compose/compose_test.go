package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadComposeSpecInterpolatesFilesAndEnv(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "otelcol.yaml"), []byte("receivers: {}\n"), 0644))
	t.Setenv("SIGNOZ_ACCESS_TOKEN", "secret-token")

	composePath := filepath.Join(dir, "hypeman.compose.yaml")
	require.NoError(t, os.WriteFile(composePath, []byte(`
version: 1
name: hypeship-otel
services:
  otelcol:
    image: otel/opentelemetry-collector-contrib:0.108.0
    cmd: ["--config=env:OTELCOL_CONFIG"]
    env:
      OTELCOL_CONFIG: ${file:otelcol.yaml}
      SIGNOZ_ACCESS_TOKEN: ${env:SIGNOZ_ACCESS_TOKEN}
`), 0644))

	spec, err := loadComposeSpec(composePath)
	require.NoError(t, err)

	service := spec.Services["otelcol"]
	assert.Equal(t, "receivers: {}\n", service.Env["OTELCOL_CONFIG"])
	assert.Equal(t, "secret-token", service.Env["SIGNOZ_ACCESS_TOKEN"])
}

func TestBuildComposeInstanceInputIncludesPolicyFields(t *testing.T) {
	service := composeServiceSpec{
		Image: "otel/opentelemetry-collector-contrib:0.108.0",
		Cmd:   []string{"--config=env:OTELCOL_CONFIG"},
		Env: map[string]string{
			"OTELCOL_CONFIG": "receivers: {}\n",
		},
		Resources: composeResourcesSpec{
			Vcpus:             8,
			Memory:            "4GB",
			BandwidthUpload:   "300Mbps",
			BandwidthDownload: "300Mbps",
		},
		Restart: &composeRestartSpec{
			Policy:      "on-failure",
			Backoff:     "5s",
			MaxAttempts: 10,
			StableAfter: "10m",
		},
		Health: &composeCheckSpec{
			HTTP:             &composeHTTPCheckSpec{Port: 13133, Path: "/", ExpectedStatus: 200},
			Interval:         "10s",
			Timeout:          "2s",
			FailureThreshold: 3,
		},
	}

	input := buildComposeInstanceInput("hypeship-otel-otelcol", service)
	inputJSON := map[string]any{}
	inputData, err := json.Marshal(input)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(inputData, &inputJSON))

	assert.Equal(t, "hypeship-otel-otelcol", inputJSON["name"])
	assert.Equal(t, service.Image, inputJSON["image"])
	assert.Equal(t, []any{"--config=env:OTELCOL_CONFIG"}, inputJSON["cmd"])
	assert.Equal(t, "4GB", inputJSON["size"])
	assert.Equal(t, float64(8), inputJSON["vcpus"])
	assert.Equal(t, map[string]any{
		"backoff":      "5s",
		"max_attempts": float64(10),
		"policy":       "on_failure",
		"stable_after": "10m",
	}, inputJSON["restart_policy"])
	assert.Equal(t, map[string]any{
		"failure_threshold": float64(3),
		"http": map[string]any{
			"expected_status": float64(200),
			"path":            "/",
			"port":            float64(13133),
		},
		"interval": "10s",
		"timeout":  "2s",
		"type":     "http",
	}, inputJSON["health_check"])
	assert.Equal(t, map[string]any{
		"bandwidth_download": "300Mbps",
		"bandwidth_upload":   "300Mbps",
	}, inputJSON["network"])
}

func TestDesiredResourcesUseDeterministicNamesAndTags(t *testing.T) {
	runner := Runner{
		spec: composeSpec{
			Version: 1,
			Name:    "hypeship-otel",
			Services: map[string]composeServiceSpec{
				"otelcol": {
					Image: "otel/opentelemetry-collector-contrib:0.108.0",
					Ingress: []composeIngressRuleSpec{
						{Hostname: "otel.example.com", HostPort: 443, TargetPort: 4318, TLS: true},
					},
				},
			},
		},
	}

	instances, ingresses, images, err := runner.desiredResources()
	require.NoError(t, err)

	require.Equal(t, []string{"otel/opentelemetry-collector-contrib:0.108.0"}, images)
	require.Len(t, instances, 1)
	assert.Equal(t, "hypeship-otel-otelcol", instances[0].Name)
	assert.Equal(t, composeResourceInstance, instances[0].Input.Tags[composeTagResource])
	assert.NotEmpty(t, instances[0].Input.Tags[composeTagHash])

	require.Len(t, ingresses, 1)
	assert.Equal(t, "hypeship-otel-otelcol-0", ingresses[0].Name)
	assert.Equal(t, composeResourceIngress, ingresses[0].Input.Tags[composeTagResource])
	assert.Equal(t, "hypeship-otel-otelcol", ingresses[0].Input.Rules[0].Target.Instance)
	assert.Equal(t, int64(4318), ingresses[0].Input.Rules[0].Target.Port)
}

func TestValidateComposeSpecRejectsInvalidNames(t *testing.T) {
	err := validateComposeSpec(&composeSpec{
		Version: 1,
		Name:    "BadName",
		Services: map[string]composeServiceSpec{
			"api": {Image: "alpine:latest"},
		},
	})

	require.EqualError(t, err, "compose name must contain only lowercase letters, digits, and dashes")
}

func TestConflictBlockers(t *testing.T) {
	blockers := conflictBlockers([]Action{
		{Action: "create", Type: "image", Name: "alpine:latest"},
		{Action: "conflict", Type: "instance", Name: "app-api", Reason: "name exists without compose ownership"},
	})

	require.Equal(t, []string{"  instance app-api: name exists without compose ownership"}, blockers)
}
