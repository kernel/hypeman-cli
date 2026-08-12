package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var pushCreateCmd = cli.Command{
	Name:      "create",
	Usage:     "Push a hypeman image to a remote registry",
	ArgsUsage: "<image> <target>",
	Description: `Create a push job that exports a hypeman image to a remote registry.

Only images in the ready state can be pushed. The push runs asynchronously;
use "hypeman push get <id>" to poll its progress.

Examples:
  # Push a cached image to ECR using the server's registry credentials
  hypeman push create alpine:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:v1

  # Push with credentials borrowed for this push only
  hypeman push create alpine:latest registry.example.com/myapp:v1 --username alice --password s3cret`,
	Flags: []cli.Flag{
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
		&cli.StringFlag{
			Name:  "registry-token",
			Usage: "Bearer token for an Authorization header",
		},
	},
	Action:          handlePushCreate,
	HideHelpCommand: true,
}

var pushListCmd = cli.Command{
	Name:  "list",
	Usage: "List outbound image push jobs",
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
	Usage:           "Get push details",
	ArgsUsage:       "<id>",
	Action:          handlePushGet,
	HideHelpCommand: true,
}

func handlePushCreate(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 2 {
		return fmt.Errorf("image and target required\nUsage: hypeman push create <image> <target>")
	}

	params := buildPushNewParams(
		args[0],
		args[1],
		cmd.Bool("insecure"),
		cmd.String("username"),
		cmd.String("password"),
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
		obj := gjson.ParseBytes(res)
		return ShowJSON(os.Stdout, "push create", obj, format, transform)
	}

	push, err := client.Pushes.New(ctx, params, opts...)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Pushing %s to %s...\n", push.Image, push.Target)
	fmt.Println(push.ID)
	return nil
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
