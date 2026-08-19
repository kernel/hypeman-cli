package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var tagCmd = cli.Command{
	Name:      "tag",
	Usage:     "Create a local image tag",
	ArgsUsage: "<source> <target>",
	Description: `Create a local image tag in Hypeman. If the source is not already
cached in Hypeman, fall back to the matching image in the local Docker daemon.`,
	Action: handleTag,
}

func handleTag(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) != 2 {
		return fmt.Errorf("source and target image references required\nUsage: hypeman tag <source> <target>")
	}

	source, target := args[0], args[1]
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	body := struct {
		Target string `json:"target"`
	}{Target: target}
	path := "/images/" + url.PathEscape(source) + "/tag"
	if err := client.Post(ctx, path, body, nil, opts...); err != nil {
		if !isNotFoundError(err) {
			return err
		}
		staged, stageErr := stageDockerImage(ctx, cmd, &client, source, target)
		if stageErr != nil {
			return fmt.Errorf("image %q was not found in Hypeman or Docker: %w", source, stageErr)
		}
		res = []byte(staged.RawJSON())
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	result := gjson.ParseBytes(res)
	if format != "auto" {
		return ShowJSON(os.Stdout, "tag", result, format, transform)
	}

	imageName := result.Get("name").String()
	if imageName == "" {
		imageName = target
	}
	fmt.Println(imageName)
	return nil
}
