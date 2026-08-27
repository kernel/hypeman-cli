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
	Name:        "tag",
	Usage:       "Create a local image tag",
	ArgsUsage:   "<source> <target>",
	Description: "Create a local image tag in Hypeman without pulling or converting the image.",
	Action:      handleTag,
}

func handleTag(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) != 2 {
		return fmt.Errorf("source and target image references required\nUsage: hypeman tag <source> <target>")
	}

	source, target := args[0], args[1]
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	res, err := tagImage(ctx, cmd, &client, source, target)
	if err != nil {
		return err
	}

	return printTagResult(cmd, target, res)
}

func tagImage(ctx context.Context, cmd *cli.Command, client *hypeman.Client, source, target string) ([]byte, error) {
	var res []byte
	opts := []option.RequestOption{option.WithResponseBodyInto(&res)}
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	path := "/images/" + url.PathEscape(source) + "/tag"
	body := struct {
		Target string `json:"target"`
	}{Target: target}
	if err := client.Post(ctx, path, body, nil, opts...); err != nil {
		return nil, err
	}
	return res, nil
}

func printTagResult(cmd *cli.Command, target string, res []byte) error {
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
