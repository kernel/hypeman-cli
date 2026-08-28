package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
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
	for _, ref := range []struct{ label, value string }{{"source", source}, {"target", target}} {
		if _, err := name.ParseReference(ref.value); err != nil {
			return fmt.Errorf("invalid %s %q: %w", ref.label, ref.value, err)
		}
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	// TODO: switch to a typed Images.Tag once the tag API from kernel/hypeman#453
	// is generated into hypeman-go.
	body := struct {
		Target string `json:"target"`
	}{Target: target}
	if err := client.Post(ctx, "/images/"+url.PathEscape(source)+"/tag", body, nil, opts...); err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	if format != "auto" {
		return ShowJSON(os.Stdout, "tag", gjson.ParseBytes(res), format, transform)
	}

	imageName := gjson.GetBytes(res, "name").String()
	if imageName == "" {
		imageName = target
	}
	fmt.Println(imageName)
	return nil
}
