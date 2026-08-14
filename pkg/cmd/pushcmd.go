package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

func pushRemoteFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "insecure",
			Usage: "Allow pushing to plain-HTTP registries",
		},
		&cli.StringFlag{
			Name:  "username",
			Usage: "Registry username",
		},
		&cli.StringFlag{
			Name:  "password",
			Usage: "Registry password or access token",
		},
		&cli.BoolFlag{
			Name:  "password-stdin",
			Usage: "Read the registry password from stdin",
		},
		&cli.StringFlag{
			Name:  "registry-token",
			Usage: "Bearer token for an Authorization header",
		},
		&cli.BoolFlag{
			Name:    "detach",
			Aliases: []string{"d"},
			Usage:   "Return after queueing the push",
		},
	}
}

var pushCreateCmd = cli.Command{
	Name:      "create",
	Aliases:   []string{"remote"},
	Usage:     "Create a remote push job (deprecated; use push SOURCE TARGET)",
	ArgsUsage: "<image> <target>",
	Flags:     pushRemoteFlags(),
	Description: `Create a remote push job without waiting for it to finish.

Use "hypeman push SOURCE TARGET" for the Docker-like flow. This command is
kept as a compatibility alias for existing scripts.`,
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

func handlePushCreate(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) != 2 {
		return fmt.Errorf("image and target required\nUsage: hypeman push create <image> <target>")
	}
	return runRemotePush(ctx, cmd, args[0], args[1])
}

func runRemotePush(ctx context.Context, cmd *cli.Command, image, target string) error {
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
	if format != "auto" {
		var res []byte
		opts = append(opts, option.WithResponseBodyInto(&res))
		_, err := client.Pushes.New(ctx, params, opts...)
		if err != nil {
			return err
		}
		return ShowJSON(os.Stdout, "push", gjson.ParseBytes(res), format, transform)
	}

	push, err := client.Pushes.New(ctx, params, opts...)
	if err != nil {
		return err
	}

	if cmd.Bool("detach") || cmd.Name == "create" || cmd.Name == "remote" {
		fmt.Fprintf(os.Stderr, "push queued: %s\n", push.ID)
		fmt.Println(push.ID)
		return nil
	}

	return waitForPush(ctx, &client, push, opts)
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

func waitForPush(ctx context.Context, client *hypeman.Client, push *hypeman.Push, opts []option.RequestOption) error {
	fmt.Fprintf(os.Stderr, "The push refers to repository [%s]\n", pushRepository(push.Target))
	fmt.Fprintln(os.Stderr, "queued")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	lastStatus := push.Status
	var lastBytes int64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			current, err := client.Pushes.Get(ctx, push.ID, opts...)
			if err != nil {
				return fmt.Errorf("check push %s: %w", push.ID, err)
			}
			if current.Status != lastStatus {
				fmt.Fprintln(os.Stderr, string(current.Status))
				lastStatus = current.Status
			}
			if current.Status == hypeman.PushStatusPushing && current.Bytes > lastBytes {
				fmt.Fprintf(os.Stderr, "pushing %s (%d layers)\n", formatBytes(current.Bytes), current.Layers)
				lastBytes = current.Bytes
			}

			switch current.Status {
			case hypeman.PushStatusPushed:
				fmt.Fprintf(os.Stderr, "digest: %s\n", current.Digest)
				return nil
			case hypeman.PushStatusFailed:
				if current.Error != "" {
					return fmt.Errorf("push failed: %s", current.Error)
				}
				return fmt.Errorf("push failed")
			}
		}
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

	credentials := hypeman.PushCredentialsParam{}
	haveCredentials := false
	if username != "" {
		credentials.Username = hypeman.Opt(username)
		haveCredentials = true
	}
	if password != "" {
		credentials.Password = hypeman.Opt(password)
		haveCredentials = true
	}
	if registryToken != "" {
		credentials.RegistryToken = hypeman.Opt(registryToken)
		haveCredentials = true
	}
	if haveCredentials {
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
