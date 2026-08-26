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
	Name:            "tag",
	Usage:           "Tag an existing local image without pulling or converting",
	ArgsUsage:       "<source> <target>",
	Action:          handleTag,
	HideHelpCommand: true,
}

// tagImageBody is the POST /images/{name}/tag request body, defined locally
// until the generated SDK gains a typed method for the endpoint.
type tagImageBody struct {
	Target string `json:"target"`
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
	path := fmt.Sprintf("images/%s/tag", url.PathEscape(source))
	if err := client.Post(ctx, path, tagImageBody{Target: target}, nil, opts...); err != nil {
		return err
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
