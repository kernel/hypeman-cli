package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestTagCommandPostsEscapedSourceAndTarget(t *testing.T) {
	var method, path, target string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.EscapedPath()
		var body struct {
			Target string `json:"target"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		target = body.Target
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"docker.io/library/myapp:latest"}`))
	}))
	defer server.Close()

	err := Command.Run(context.Background(), []string{
		"hypeman", "--base-url", server.URL, "--format", "json",
		"tag", "builds/job:latest", "myapp:latest",
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.MethodPost, method)
	assert.Equal(t, "/images/builds%2Fjob:latest/tag", path)
	assert.Equal(t, "myapp:latest", target)

	stdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	err = Command.Run(context.Background(), []string{
		"hypeman", "--base-url", server.URL,
		"tag", "builds/job:latest", "myapp:latest",
	})
	_ = writer.Close()
	os.Stdout = stdout
	if err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, string(output), "docker.io/library/myapp:latest")
}

func TestTagCommandFallsBackToDockerWhenHypemanMisses(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	err := Command.Run(context.Background(), []string{
		"hypeman", "--base-url", server.URL,
		"tag", "not a valid image", "myapp:latest",
	})
	require.ErrorContains(t, err, "was not found in Hypeman; stage it from Docker")
}

func TestTagCommandPropagatesNotReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"image_not_ready","message":"image is not ready"}`))
	}))
	defer server.Close()

	err := Command.Run(context.Background(), []string{
		"hypeman", "--base-url", server.URL,
		"tag", "alpine:pending", "myapp:latest",
	})
	require.ErrorContains(t, err, "image is not ready")
}
