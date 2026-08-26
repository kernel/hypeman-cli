package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagCommand_RequiresSourceAndTarget(t *testing.T) {
	err := Command.Run(context.Background(), []string{"hypeman", "tag", "only-source"})
	require.EqualError(t, err, "source and target image references required\nUsage: hypeman tag <source> <target>")
}

func TestTagCommand_PostsToTagEndpoint(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"name":"registry.example.com/app:v1","digest":"sha256:abc","status":"ready","created_at":"2026-01-01T00:00:00Z"}`)
	}))
	defer srv.Close()

	// Capture os.Stdout: handleTag prints the resolved image name directly.
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	runErr := Command.Run(context.Background(), []string{
		"hypeman", "--base-url", srv.URL,
		"tag",
		"docker.io/library/alpine:latest", "registry.example.com/app:v1",
	})

	w.Close()
	os.Stdout = origStdout
	out, readErr := io.ReadAll(r)
	require.NoError(t, readErr)

	require.NoError(t, runErr)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/images/docker.io%2Flibrary%2Falpine:latest/tag", gotPath)
	assert.JSONEq(t, `{"target":"registry.example.com/app:v1"}`, string(gotBody))
	assert.Equal(t, "registry.example.com/app:v1\n", string(out))
}

func TestTagCommand_SurfacesServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"code":"not_found","message":"image not found"}`)
	}))
	defer srv.Close()

	err := Command.Run(context.Background(), []string{
		"hypeman", "--base-url", srv.URL,
		"tag",
		"docker.io/library/alpine:missing", "registry.example.com/app:v1",
	})
	require.Error(t, err)
}
