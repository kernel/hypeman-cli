package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

var pushCmd = cli.Command{
	Name:      "push",
	Aliases:   []string{"pushes"},
	Usage:     "Push an image to a registry",
	ArgsUsage: "SOURCE [TARGET]",
	Description: `Push an image from Hypeman to a remote registry.

The source image must already exist in Hypeman. TARGET is the remote registry
reference, matching Docker's push syntax as closely as possible.

Local Docker-daemon uploads remain available explicitly with "push local":
  hypeman push local IMAGE [TARGET]

Push jobs can be inspected while they run:
  hypeman push ls
  hypeman push inspect <id>

Examples:
  # Push a cached image to ECR
  hypeman push alpine:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:v1

  # Push with credentials read from stdin
  echo "$ECR_PASSWORD" | hypeman push alpine:latest registry.example.com/app:v1 \
    --username AWS --password-stdin

  # Upload a local Docker image into Hypeman
  hypeman push local nginx:latest`,
	Flags:           pushRemoteFlags(),
	Commands:        []*cli.Command{&pushLocalCmd, &pushCreateCmd, &pushListCmd, &pushGetCmd},
	Action:          handlePush,
	HideHelpCommand: true,
}

var pushLocalCmd = cli.Command{
	Name:            "local",
	Usage:           "Upload a local Docker image to Hypeman",
	ArgsUsage:       "IMAGE [TARGET]",
	Action:          handleLocalPush,
	HideHelpCommand: true,
}

func handlePush(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	switch len(args) {
	case 1:
		// Keep the old one-argument form working for existing scripts.
		return handleLocalPush(ctx, cmd)
	case 2:
		return runRemotePush(ctx, cmd, args[0], args[1])
	default:
		return fmt.Errorf("source image and target required\nUsage: hypeman push <source> <target>")
	}
}

func handleLocalPush(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("image reference required\nUsage: hypeman push local <image> [target]")
	}

	sourceImage := args[0]
	targetName := sourceImage
	if len(args) > 1 {
		targetName = args[1]
	}

	baseURL := resolveBaseURL(cmd)

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("invalid base URL %q: missing host", baseURL)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid base URL %q: scheme must be http or https", baseURL)
	}

	registryHost := parsedURL.Host

	srcRef, err := name.ParseReference(sourceImage)
	if err != nil {
		return fmt.Errorf("invalid source image: %w", err)
	}

	// Build and validate the target before opening the Docker daemon. The
	// server computes the image digest from the manifest, while the tag keeps
	// the image addressable with Docker-like image names after the push.
	targetRef := registryHost + "/" + strings.TrimPrefix(targetName, "/")
	parseOptions := []name.Option(nil)
	if parsedURL.Scheme == "http" {
		parseOptions = append(parseOptions, name.Insecure)
	}
	dstRef, err := name.ParseReference(targetRef, parseOptions...)
	if err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Loading image %s from Docker...\n", sourceImage)
	img, err := daemon.Image(srcRef)
	if err != nil {
		return fmt.Errorf("load image: %w", err)
	}

	fmt.Fprintf(os.Stderr, "The push refers to repository [%s]\n", dstRef.Context().Name())

	token := resolveAPIKey()

	// Use custom transport that always sends Basic auth header
	transport := &authTransport{
		base:  http.DefaultTransport,
		token: token,
	}

	progress := make(chan v1.Update, 32)
	progressDone := make(chan struct{})
	progressStop := make(chan struct{})
	go func() {
		defer close(progressDone)
		renderPushProgress(progress, os.Stderr, term.IsTerminal(int(os.Stderr.Fd())), progressStop)
	}()

	err = remote.Write(dstRef, img,
		remote.WithContext(ctx),
		remote.WithAuth(authn.Anonymous),
		remote.WithTransport(transport),
		remote.WithProgress(progress),
	)
	close(progressStop)
	<-progressDone
	if err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	digest, err := img.Digest()
	if err != nil {
		return fmt.Errorf("read pushed image digest: %w", err)
	}
	rawManifest, err := img.RawManifest()
	if err != nil {
		return fmt.Errorf("read pushed image manifest: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s: digest: %s size: %d\n", dstRef.Identifier(), digest, len(rawManifest))
	return nil
}

// renderPushProgress consumes go-containerregistry's aggregate byte updates.
// Keep progress on stderr so stdout remains available for shell pipelines.
func renderPushProgress(updates <-chan v1.Update, output io.Writer, interactive bool, stop <-chan struct{}) {
	printed := false
	for {
		select {
		case <-stop:
			if printed && interactive {
				fmt.Fprintln(output)
			}
			return
		case update, ok := <-updates:
			if !ok {
				if printed && interactive {
					fmt.Fprintln(output)
				}
				return
			}
			if update.Error != nil || update.Total <= 0 {
				continue
			}
			if interactive {
				fmt.Fprintf(output, "\r%s / %s", formatBytes(update.Complete), formatBytes(update.Total))
				printed = true
			}
		}
	}
}

// authTransport adds Basic auth header to all requests
type authTransport struct {
	base  http.RoundTripper
	token string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" {
		// Clone request to avoid modifying the original (RoundTripper contract)
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.base.RoundTrip(req)
}
