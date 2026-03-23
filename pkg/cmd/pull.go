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

var pullCmd = cli.Command{
	Name:            "pull",
	Usage:           "Alias for `image create`",
	ArgsUsage:       "<image>",
	Flags:           imageCreateFlags(),
	Action:          handlePull,
	HideHelpCommand: true,
}

func handlePull(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("image reference required\nUsage: hypeman pull <image>")
	}

	image := args[0]
	params, malformedTags := buildImageNewParams(image, cmd.StringSlice("tag"))
	for _, malformed := range malformedTags {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag: %s\n", malformed)
	}

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
		_, err := client.Images.New(ctx, params, opts...)
		if err != nil {
			return err
		}
		obj := gjson.ParseBytes(res)
		return ShowJSON(os.Stdout, "pull", obj, format, transform)
	}

	fmt.Fprintf(os.Stderr, "Pulling %s...\n", image)

	result, err := client.Images.New(ctx, params, opts...)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Status: %s\n", result.Status)
	if result.Digest != "" {
		fmt.Fprintf(os.Stderr, "Digest: %s\n", result.Digest)
	}
	fmt.Fprintf(os.Stderr, "Image: %s\n", result.Name)

	return nil
}
