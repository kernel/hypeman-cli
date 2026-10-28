package cmd

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"sort"
	"strings"
	"testing"

	"github.com/kernel/hypeman-go"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

const testMachineBaseImage = "registry.example.com/macos/base@sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func parseBuildParams(t *testing.T, args ...string) (hypeman.BuildNewParams, error) {
	t.Helper()
	var params hypeman.BuildNewParams
	command := &cli.Command{
		Name:  "build",
		Flags: testBuildFlags(),
		Action: func(_ context.Context, cmd *cli.Command) error {
			var err error
			params, err = buildNewParams(cmd, []byte("source archive"), "", cmd.Int("timeout"))
			return err
		},
	}
	argv := append([]string{"build"}, args...)
	return params, command.Run(context.Background(), argv)
}

func testBuildFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"f"}},
		&cli.IntFlag{Name: "timeout", Value: 600},
		&cli.StringFlag{Name: "base-image-digest"},
		&cli.StringFlag{Name: "machine-base-image"},
		&cli.StringFlag{Name: "builder", Aliases: []string{"builder-id"}},
		&cli.StringFlag{Name: "cache-scope"},
		&cli.StringFlag{Name: "global-cache-key"},
		&cli.StringFlag{Name: "is-admin-build"},
		&cli.StringFlag{Name: "secrets"},
		&cli.StringFlag{Name: "image-name"},
		&cli.IntFlag{Name: "cpus"},
		&cli.IntFlag{Name: "memory"},
		&cli.StringSliceFlag{Name: "tag"},
	}
}

func multipartFieldNames(t *testing.T, params hypeman.BuildNewParams) []string {
	t.Helper()
	body, contentType, err := params.MarshalMultipart()
	require.NoError(t, err)
	_, parameter, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	reader := multipart.NewReader(bytes.NewReader(body), parameter["boundary"])
	var names []string
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, part.FormName())
		_ = part.Close()
	}
	sort.Strings(names)
	return names
}

func TestMachineBuildMultipartFields(t *testing.T) {
	params, err := parseBuildParams(t,
		"--machine-base-image", testMachineBaseImage,
		"--cpus", "4", "--memory", "8192", "--tag", "team=cli")
	require.NoError(t, err)

	got := multipartFieldNames(t, params)
	want := []string{"cpus", "machine_base_image", "memory_mb", "source", "tags", "timeout_seconds"}
	require.Equal(t, want, got)
}

func TestMachineBuildRejectsLinuxOptions(t *testing.T) {
	for _, option := range []string{
		"--file", "--base-image-digest", "--builder", "--cache-scope",
		"--global-cache-key", "--is-admin-build", "--secrets", "--image-name",
	} {
		t.Run(strings.TrimLeft(option, "-"), func(t *testing.T) {
			value := "value"
			if option == "--file" {
				value = "Dockerfile"
			}
			args := []string{"--machine-base-image", testMachineBaseImage, option, value}
			_, err := parseBuildParams(t, args...)
			require.ErrorContains(t, err, option+" cannot be used with --machine-base-image")
		})
	}
}

func TestMachineBuildRequiresDigestPinnedImage(t *testing.T) {
	for _, image := range []string{
		"registry.example.com/macos/base:latest", "registry.example.com/macos/base@sha256:abc",
		"@sha256:" + strings.Repeat("a", 64),
		"registry.example.com/macos/base@sha256:" + strings.Repeat("A", 64),
		"registry.example.com/macos/base@extra@sha256:" + strings.Repeat("a", 64),
	} {
		_, err := parseBuildParams(t, "--machine-base-image", image)
		require.ErrorContains(t, err, "must be pinned by a sha256 digest")
	}
}

func TestLinuxBuildMultipartFieldsRemainUnchanged(t *testing.T) {
	params, err := parseBuildParams(t, "--base-image-digest", "sha256:base")
	require.NoError(t, err)
	got := multipartFieldNames(t, params)
	want := []string{"base_image_digest", "source", "timeout_seconds"}
	require.Equal(t, want, got)
}
