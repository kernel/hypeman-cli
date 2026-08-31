package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushCommandStructure(t *testing.T) {
	subcommandNames := make([]string, 0, len(pushCmd.Commands))
	for _, sub := range pushCmd.Commands {
		subcommandNames = append(subcommandNames, sub.Name)
	}

	assert.Contains(t, subcommandNames, "local")
	assert.Contains(t, subcommandNames, "create")
	assert.Contains(t, subcommandNames, "list")
	assert.Contains(t, subcommandNames, "get")
	assert.Contains(t, pushListCmd.Aliases, "ls")
	assert.Contains(t, pushGetCmd.Aliases, "inspect")

	// The parent action remains reachable for the staged local and direct
	// remote-push forms.
	assert.NotNil(t, pushCmd.Action)
}

func TestPushRepository(t *testing.T) {
	assert.Equal(t, "registry.example.com/app", pushRepository("registry.example.com/app:v1"))
}

func TestWaitForImageRecordSkipsOlderDigest(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		digest := "sha256:old"
		if requests > 1 {
			digest = "sha256:new"
		}
		_, _ = fmt.Fprintf(w, `{"name":"registry.example.com/app:latest","digest":"%s","status":"ready","created_at":"2026-08-19T00:00:00Z"}`, digest)
	}))
	defer server.Close()

	client := hypeman.NewClient(option.WithBaseURL(server.URL))
	img, err := waitForImageRecord(context.Background(), &client, "registry.example.com/app:latest", "sha256:new")
	require.NoError(t, err)
	require.Equal(t, "sha256:new", img.Digest)
	require.Equal(t, 2, requests)
}

func TestPushTargetFallsBackToCachedHypemanImage(t *testing.T) {
	var pushImage, pushTarget string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/images/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:v1","digest":"sha256:test","status":"ready","created_at":"2026-08-19T00:00:00Z"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/pushes":
			var body struct {
				Image  string `json:"image"`
				Target string `json:"target"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			pushImage = body.Image
			pushTarget = body.Target
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"push-1","created_at":"2026-08-19T00:00:00Z","digest":"sha256:test","image":"123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:v1","status":"pushed","target":"123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:v1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	err := Command.Run(context.Background(), []string{
		"hypeman", "--base-url", server.URL, "push",
		"123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:v1",
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:v1", pushImage)
	assert.Equal(t, pushImage, pushTarget)
}

func TestPushTargetErrorsWhenImageMissingEverywhere(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	err := Command.Run(context.Background(), []string{
		"hypeman", "--base-url", server.URL, "push",
		"123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:v1",
	})
	require.ErrorContains(t, err, "not found in local Docker or the Hypeman cache")
}

func TestPushTargetPropagatesCachedImageFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:v1","digest":"sha256:test","status":"failed","error":"layer fetch failed","created_at":"2026-08-19T00:00:00Z"}`))
	}))
	defer server.Close()

	err := Command.Run(context.Background(), []string{
		"hypeman", "--base-url", server.URL, "push",
		"123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:v1",
	})
	require.ErrorContains(t, err, "layer fetch failed")
}

func TestValidateRemotePushReferences(t *testing.T) {
	assert.NoError(t, validateRemotePushReferences("alpine:latest", "registry.example.com/app:v1"))
	assert.ErrorContains(t, validateRemotePushReferences("alpine:latest", "registry.example.com/app"), "explicit tag")
	assert.ErrorContains(t, validateRemotePushReferences("not valid", "registry.example.com/app:v1"), "invalid source image")
	assert.Error(t, validateRemotePushReferences("alpine:latest", "registry.example.com/app@sha256:abc"))
}

func TestBuildPushNewParams(t *testing.T) {
	params := buildPushNewParams("alpine:latest", "registry.example.com/alpine:v1", false, "", "", "")

	require.Equal(t, "alpine:latest", params.CreatePushRequest.Image)
	require.Equal(t, "registry.example.com/alpine:v1", params.CreatePushRequest.Target)
	assert.False(t, params.CreatePushRequest.Insecure.Valid())

	// Omitting credentials lets the server use its own registry credentials.
	assert.False(t, params.CreatePushRequest.Credentials.Username.Valid())
	assert.False(t, params.CreatePushRequest.Credentials.Password.Valid())
	assert.False(t, params.CreatePushRequest.Credentials.RegistryToken.Valid())
}

func TestBuildPushNewParamsWithCredentials(t *testing.T) {
	params := buildPushNewParams("alpine:latest", "registry.example.com/alpine:v1", true, "alice", "s3cret", "token")

	require.True(t, params.CreatePushRequest.Insecure.Valid())
	assert.True(t, params.CreatePushRequest.Insecure.Value)

	credentials := params.CreatePushRequest.Credentials
	require.True(t, credentials.Username.Valid())
	assert.Equal(t, "alice", credentials.Username.Value)
	require.True(t, credentials.Password.Valid())
	assert.Equal(t, "s3cret", credentials.Password.Value)
	require.True(t, credentials.RegistryToken.Valid())
	assert.Equal(t, "token", credentials.RegistryToken.Value)
}

func TestBuildPushNewParamsPartialCredentials(t *testing.T) {
	params := buildPushNewParams("alpine:latest", "registry.example.com/alpine:v1", false, "", "", "token")

	credentials := params.CreatePushRequest.Credentials
	assert.False(t, credentials.Username.Valid())
	assert.False(t, credentials.Password.Valid())
	require.True(t, credentials.RegistryToken.Valid())
	assert.Equal(t, "token", credentials.RegistryToken.Value)
}
