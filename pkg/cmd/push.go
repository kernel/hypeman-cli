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
	Usage:     "Push an image to hypeman",
	ArgsUsage: "NAME[:TAG] [TARGET]",
	Description: `Push an image from the local Docker daemon into the hypeman image cache.

The command follows Docker's push flow: the source image is read from the
local daemon, uploaded to hypeman, and reported with its manifest digest. If
TARGET is omitted, the source name and tag are used.

Subcommands manage outbound pushes, which export a cached hypeman image to a
remote registry (e.g. AWS ECR, Docker Hub):
  hypeman push create <image> <target>  Push a hypeman image to a remote registry
  hypeman push list                     List outbound image push jobs
  hypeman push get <id>                 Get push details

Examples:
  # Push the local nginx:latest image
  hypeman push nginx:latest

  # Push using a different repository or tag
  hypeman push nginx:latest myapp/nginx:v1

  # Export a cached hypeman image to a remote registry
  hypeman push create nginx:latest registry.example.com/nginx:latest`,
	Commands: []*cli.Command{
		&pushCreateCmd,
		&pushListCmd,
		&pushGetCmd,
	},
	Action:          handlePush,
	HideHelpCommand: true,
}

func handlePush(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("image reference required\nUsage: hypeman push <image>")
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

	fmt.Fprintf(os.Stderr, "Loading image %s from Docker...\n", sourceImage)

	srcRef, err := name.ParseReference(sourceImage)
	if err != nil {
		return fmt.Errorf("invalid source image: %w", err)
	}

	img, err := daemon.Image(srcRef)
	if err != nil {
		return fmt.Errorf("load image: %w", err)
	}

	// Build the target reference. The server computes the image digest from
	// the manifest, while the tag keeps the image addressable with Docker-like
	// image names after the push.
	targetRef := registryHost + "/" + strings.TrimPrefix(targetName, "/")
	parseOptions := []name.Option(nil)
	if parsedURL.Scheme == "http" {
		parseOptions = append(parseOptions, name.Insecure)
	}
	dstRef, err := name.ParseReference(targetRef, parseOptions...)
	if err != nil {
		return fmt.Errorf("invalid target: %w", err)
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
	go func() {
		defer close(progressDone)
		renderPushProgress(progress, os.Stderr, term.IsTerminal(int(os.Stderr.Fd())))
	}()

	err = remote.Write(dstRef, img,
		remote.WithContext(ctx),
		remote.WithAuth(authn.Anonymous),
		remote.WithTransport(transport),
		remote.WithProgress(progress),
	)
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
func renderPushProgress(updates <-chan v1.Update, output io.Writer, interactive bool) {
	if !interactive {
		for range updates {
		}
		return
	}

	printed := false
	for update := range updates {
		if update.Error != nil || update.Total <= 0 {
			continue
		}
		fmt.Fprintf(output, "\r%s / %s", formatBytes(update.Complete), formatBytes(update.Total))
		printed = true
	}
	if printed {
		fmt.Fprintln(output)
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
