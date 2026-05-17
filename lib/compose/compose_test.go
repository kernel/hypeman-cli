package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kernel/hypeman-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadComposeSpecInterpolatesFilesAndEnv(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "otelcol.yaml"), []byte("endpoint: https://${env:OTEL_COLLECTOR_VM_HOSTNAME}\ntoken: ${file:token.txt}\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "token.txt"), []byte("${env:OTEL_COLLECTOR_VM_TOKEN}"), 0644))
	t.Setenv("COMPOSE_NAME", "hypeship-otel")
	t.Setenv("OTEL_IMAGE", "otel/opentelemetry-collector-contrib:0.108.0")
	t.Setenv("OTELCOL_ENV_NAME", "OTELCOL_CONFIG")
	t.Setenv("OTEL_COLLECTOR_VM_HOSTNAME", "otel.example.com")
	t.Setenv("OTEL_COLLECTOR_VM_TOKEN", "collector-token")
	t.Setenv("SIGNOZ_ACCESS_TOKEN", "secret-token")

	composePath := filepath.Join(dir, "hypeman.compose.yaml")
	require.NoError(t, os.WriteFile(composePath, []byte(`
version: 1
name: ${env:COMPOSE_NAME}
services:
  otelcol:
    image: ${env:OTEL_IMAGE}
    cmd: ["--config=env:${env:OTELCOL_ENV_NAME}"]
    env:
      OTELCOL_CONFIG: ${file:otelcol.yaml}
      SIGNOZ_ACCESS_TOKEN: ${env:SIGNOZ_ACCESS_TOKEN}
    ingress:
      - hostname: ${env:OTEL_COLLECTOR_VM_HOSTNAME}
        target_port: 4318
`), 0644))

	spec, err := loadComposeSpec(composePath)
	require.NoError(t, err)

	service := spec.Services["otelcol"]
	assert.Equal(t, "hypeship-otel", spec.Name)
	assert.Equal(t, "otel/opentelemetry-collector-contrib:0.108.0", service.Image)
	assert.Equal(t, []string{"--config=env:OTELCOL_CONFIG"}, service.Cmd)
	assert.Equal(t, "endpoint: https://otel.example.com\ntoken: collector-token\n", service.Env["OTELCOL_CONFIG"])
	assert.Equal(t, "secret-token", service.Env["SIGNOZ_ACCESS_TOKEN"])
	require.Len(t, service.Ingress, 1)
	assert.Equal(t, "otel.example.com", service.Ingress[0].Hostname)
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

	_, instances, ingresses, images, err := runner.desiredResources()
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

func TestValidateComposeSpecRejectsImageAndDockerfile(t *testing.T) {
	err := validateComposeSpec(&composeSpec{
		Version: 1,
		Name:    "worker-stack",
		Services: map[string]composeServiceSpec{
			"worker": {Image: "alpine:latest", Dockerfile: "./Dockerfile"},
		},
	})

	require.EqualError(t, err, `service "worker" cannot include both image and dockerfile`)
}

func TestDesiredResourcesBuildsDockerfileService(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine:latest\nCOPY worker /worker\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "worker"), []byte("echo ok\n"), 0644))

	composePath := filepath.Join(dir, "hypeman.compose.yaml")
	require.NoError(t, os.WriteFile(composePath, []byte(`
version: 1
name: worker-stack
services:
  worker:
    dockerfile: ./Dockerfile
    cmd: ["./worker"]
`), 0644))

	spec, err := loadComposeSpec(composePath)
	require.NoError(t, err)

	runner := Runner{file: composePath, spec: spec}
	builds, instances, _, images, err := runner.desiredResources()
	require.NoError(t, err)

	require.Empty(t, images)
	require.Len(t, builds, 1)
	require.Len(t, instances, 1)
	assert.Equal(t, "worker", builds[0].Service)
	assert.Regexp(t, `^compose/worker-stack/worker:[a-f0-9]{12}$`, builds[0].Image)
	assert.Equal(t, builds[0].Image, instances[0].Input.Image)

	again, _, _, _, err := runner.desiredResources()
	require.NoError(t, err)
	require.Equal(t, builds[0].Image, again[0].Image)

	require.NoError(t, os.WriteFile(filepath.Join(dir, ".dockerignore"), []byte("*.tmp\n"), 0644))
	dockerignoreChanged, _, _, _, err := runner.desiredResources()
	require.NoError(t, err)
	require.NotEqual(t, builds[0].Image, dockerignoreChanged[0].Image)

	require.NoError(t, os.Remove(filepath.Join(dir, ".dockerignore")))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "worker"), []byte("echo changed\n"), 0644))
	changed, _, _, _, err := runner.desiredResources()
	require.NoError(t, err)
	require.NotEqual(t, builds[0].Image, changed[0].Image)
}

func TestRunnableBuildImage(t *testing.T) {
	assert.Equal(t, "builds/build-id", runnableBuildImage(&hypeman.Build{
		ID:       "build-id",
		ImageRef: "builds/build-id",
	}))
	assert.Equal(t, "docker.io/builds/build-id:latest", runnableBuildImage(&hypeman.Build{
		ID: "build-id",
	}))
}

func TestUpdateDesiredInstanceImageRehashesTags(t *testing.T) {
	instances := []desiredInstance{{
		Service: "worker",
		Input: hypeman.InstanceNewParams{
			Name:  "worker-stack-worker",
			Image: "compose/worker-stack/worker:original",
			Tags:  composeTags("worker-stack", "worker", composeResourceInstance, "old-hash"),
		},
	}}

	require.NoError(t, updateDesiredInstanceImage(instances, "worker-stack", "worker", "builds/build-id"))

	assert.Equal(t, "builds/build-id", instances[0].Input.Image)
	tags := instances[0].Input.Tags
	assert.Equal(t, composeResourceInstance, tags[composeTagResource])
	assert.NotEqual(t, "old-hash", tags[composeTagHash])
	assert.Equal(t, instances[0].Hash, tags[composeTagHash])
}

func TestConflictBlockers(t *testing.T) {
	blockers := conflictBlockers([]Action{
		{Action: "create", Type: "image", Name: "alpine:latest"},
		{Action: "conflict", Type: "instance", Name: "app-api", Reason: "name exists without compose ownership"},
	})

	require.Equal(t, []string{"  instance app-api: name exists without compose ownership"}, blockers)
}
