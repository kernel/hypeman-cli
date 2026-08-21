package cmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func pushRemoteFlags() []cli.Flag {
	flags := []cli.Flag{
		&cli.BoolFlag{
			Name:  "insecure",
			Usage: "Allow pushing to plain-HTTP registries",
		},
	}
	flags = append(flags, registryCredentialFlags()...)
	return append(flags,
		&cli.BoolFlag{
			Name:  "password-stdin",
			Usage: "Read the registry password from stdin",
		},
		&cli.BoolFlag{
			Name:    "detach",
			Aliases: []string{"d"},
			Usage:   "Return after queueing the push",
		},
	)
}

var pushCreateCmd = cli.Command{
	Name:      "create",
	Aliases:   []string{"remote"},
	Usage:     "Create a remote push job (deprecated; use push SOURCE TARGET)",
	ArgsUsage: "<image> <target>",
	Flags:     pushRemoteFlags(),
	Description: `Create a remote push job and return its ID immediately.

Use "hypeman push SOURCE TARGET" for the Docker-like flow, which waits by
default. This command is kept as a detached compatibility alias for existing
scripts.`,
	Action:          handlePushCreate,
	HideHelpCommand: true,
}

var pushListCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "List outbound image push jobs",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Only display push IDs",
		},
	},
	Action:          handlePushList,
	HideHelpCommand: true,
}

var pushGetCmd = cli.Command{
	Name:            "get",
	Aliases:         []string{"inspect"},
	Usage:           "Get push details",
	ArgsUsage:       "<id>",
	Action:          handlePushGet,
	HideHelpCommand: true,
}

func handleRemotePushTarget(ctx context.Context, cmd *cli.Command, target string) error {
	if err := validateRemotePushTarget(target); err != nil {
		return err
	}

	// The one-argument form follows Docker's local-tag flow: TARGET must be
	// present in the local Docker daemon before it can be staged and pushed.
	// Cached Hypeman images use the explicit IMAGE TARGET form instead.
	img, err := loadDockerImage(target)
	if err != nil {
		return fmt.Errorf("load local Docker image %q: %w; tag it first or use hypeman push <image> <target> for a cached Hypeman image", target, err)
	}

	fmt.Fprintf(os.Stderr, "Staging local image %s in Hypeman...\n", target)
	if err := uploadLocalImage(ctx, cmd, target, img); err != nil {
		return err
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	imported, err := waitForImageRecord(ctx, &client, target)
	if err != nil {
		return err
	}
	if err := waitForImageReady(ctx, &client, imported); err != nil {
		return err
	}

	return runRemotePush(ctx, cmd, target, target)
}

func validateRemotePushTarget(target string) error {
	if _, err := name.ParseReference(target); err != nil {
		return fmt.Errorf("invalid target %q: %w", target, err)
	}
	lastSlash := strings.LastIndex(target, "/")
	lastColon := strings.LastIndex(target, ":")
	lastAt := strings.LastIndex(target, "@")
	if lastAt > lastSlash || lastColon <= lastSlash {
		return fmt.Errorf("target %q must include an explicit tag", target)
	}
	return nil
}

func waitForImageRecord(ctx context.Context, client *hypeman.Client, imageName string) (*hypeman.Image, error) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		img, err := client.Images.Get(ctx, url.PathEscape(imageName))
		if err == nil {
			return img, nil
		}
		if !isNotFoundError(err) {
			return nil, fmt.Errorf("get staged image %s: %w", imageName, err)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func handlePushCreate(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) != 2 {
		return fmt.Errorf("image and target required\nUsage: hypeman push create <image> <target>")
	}
	return runRemotePush(ctx, cmd, args[0], args[1])
}

func runRemotePush(ctx context.Context, cmd *cli.Command, image, target string) error {
	if err := validateRemotePushReferences(image, target); err != nil {
		return err
	}

	password, err := pushPassword(cmd)
	if err != nil {
		return err
	}

	params := buildPushNewParams(
		image,
		target,
		cmd.Bool("insecure"),
		cmd.String("username"),
		password,
		cmd.String("registry-token"),
	)

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	push, err := client.Pushes.New(ctx, params, opts...)
	if err != nil {
		return err
	}

	legacyDetached := cmd.Name == "create" || cmd.Name == "remote"
	if cmd.Bool("detach") || legacyDetached {
		if format != "auto" {
			return ShowJSON(os.Stdout, "push", gjson.Parse(push.RawJSON()), format, transform)
		}
		fmt.Fprintf(os.Stderr, "push queued: %s\n", push.ID)
		fmt.Println(push.ID)
		return nil
	}

	final, err := waitForPush(ctx, &client, push, opts, format != "auto")
	if err != nil {
		return err
	}
	if format != "auto" {
		return ShowJSON(os.Stdout, "push", gjson.Parse(final.RawJSON()), format, transform)
	}
	return nil
}

func validateRemotePushReferences(image, target string) error {
	if _, err := name.ParseReference(image); err != nil {
		return fmt.Errorf("invalid source image %q: %w", image, err)
	}
	return validateRemotePushTarget(target)
}

func pushPassword(cmd *cli.Command) (string, error) {
	password := cmd.String("password")
	if !cmd.Bool("password-stdin") {
		return password, nil
	}
	if password != "" {
		return "", fmt.Errorf("--password and --password-stdin cannot be used together")
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read registry password from stdin: %w", err)
	}
	return strings.TrimRight(string(data), "\r\n"), nil
}

func waitForPush(ctx context.Context, client *hypeman.Client, push *hypeman.Push, opts []option.RequestOption, quiet bool) (*hypeman.Push, error) {
	var renderer *pushStatusRenderer
	if !quiet {
		fmt.Fprintf(os.Stderr, "The push refers to repository [%s]\n", pushRepository(push.Target))
		renderer = &pushStatusRenderer{
			output:      os.Stderr,
			interactive: term.IsTerminal(int(os.Stderr.Fd())),
		}
		defer renderer.finish()
	}

	var lastBytes int64
	return pollPush(ctx, client, push, opts, func(current *hypeman.Push) {
		if renderer == nil {
			return
		}
		message, updatedBytes := pushStatusText(current, lastBytes)
		lastBytes = updatedBytes
		if message != "" {
			renderer.update(message)
		}
	})
}

func pollPush(ctx context.Context, client *hypeman.Client, push *hypeman.Push, opts []option.RequestOption, update func(*hypeman.Push)) (*hypeman.Push, error) {
	current := push
	for {
		update(current)

		switch current.Status {
		case hypeman.PushStatusPushed:
			return current, nil
		case hypeman.PushStatusFailed:
			if current.Error != "" {
				return nil, fmt.Errorf("push %s failed: %s", push.ID, current.Error)
			}
			return nil, fmt.Errorf("push %s failed", push.ID)
		}

		ticker := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			ticker.Stop()
			return nil, ctx.Err()
		case <-ticker.C:
		}

		var err error
		current, err = client.Pushes.Get(ctx, push.ID, opts...)
		if err != nil {
			return nil, fmt.Errorf("check push %s: %w", push.ID, err)
		}
	}
}

func pushStatusText(push *hypeman.Push, lastBytes int64) (string, int64) {
	switch push.Status {
	case hypeman.PushStatusQueued:
		return fmt.Sprintf("queued · %s", push.Target), lastBytes
	case hypeman.PushStatusPushing:
		if push.Bytes > lastBytes {
			lastBytes = push.Bytes
		}
		return fmt.Sprintf("pushing %s · %d layers · %s", formatBytes(lastBytes), push.Layers, push.Target), lastBytes
	case hypeman.PushStatusPushed:
		return fmt.Sprintf("pushed · digest: %s", push.Digest), lastBytes
	case hypeman.PushStatusFailed:
		message := push.Error
		if message == "" {
			message = "unknown error"
		}
		return "failed · " + message, lastBytes
	default:
		return "", lastBytes
	}
}

type pushStatusRenderer struct {
	output      io.Writer
	interactive bool
	last        string
}

func (r *pushStatusRenderer) update(message string) {
	if message == r.last {
		return
	}
	r.last = message
	if r.interactive {
		fmt.Fprintf(r.output, "\r\033[K%s", message)
		return
	}
	fmt.Fprintln(r.output, message)
}

func (r *pushStatusRenderer) finish() {
	if r.interactive && r.last != "" {
		fmt.Fprintln(r.output)
	}
}

func pushRepository(target string) string {
	ref, err := name.ParseReference(target)
	if err != nil {
		return target
	}
	return ref.Context().Name()
}

// buildPushNewParams assembles the outbound push request. Credentials are only
// sent when at least one is supplied, so that the server falls back to its own
// registry credentials otherwise.
func buildPushNewParams(image, target string, insecure bool, username, password, registryToken string) hypeman.PushNewParams {
	params := hypeman.PushNewParams{
		CreatePushRequest: hypeman.CreatePushRequestParam{
			Image:  image,
			Target: target,
		},
	}
	if insecure {
		params.CreatePushRequest.Insecure = hypeman.Opt(true)
	}

	if credentials, ok := buildRegistryCredentials(username, password, registryToken); ok {
		params.CreatePushRequest.Credentials = credentials
	}

	return params
}

func handlePushList(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	if format != "auto" {
		var res []byte
		opts = append(opts, option.WithResponseBodyInto(&res))
		_, err := client.Pushes.List(ctx, opts...)
		if err != nil {
			return err
		}
		obj := gjson.ParseBytes(res)
		return ShowJSON(os.Stdout, "push list", obj, format, transform)
	}

	pushes, err := client.Pushes.List(ctx, opts...)
	if err != nil {
		return err
	}

	if cmd.Bool("quiet") {
		for _, p := range *pushes {
			fmt.Println(p.ID)
		}
		return nil
	}

	if len(*pushes) == 0 {
		fmt.Fprintln(os.Stderr, "No pushes found.")
		return nil
	}

	table := NewTableWriter(os.Stdout, "ID", "STATUS", "IMAGE", "TARGET", "SIZE", "CREATED")
	table.TruncOrder = []int{3, 2, 0, 5} // TARGET first, then IMAGE, ID, CREATED
	for _, p := range *pushes {
		size := "-"
		if p.Bytes > 0 {
			size = formatBytes(p.Bytes)
		}

		table.AddRow(
			TruncateID(p.ID),
			string(p.Status),
			p.Image,
			p.Target,
			size,
			FormatTimeAgo(p.CreatedAt),
		)
	}
	table.Render()

	return nil
}

func handlePushGet(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("push ID required\nUsage: hypeman push get <id>")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	_, err := client.Pushes.Get(ctx, args[0], opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(res)
	return ShowJSON(os.Stdout, "push get", obj, format, transform)
}
