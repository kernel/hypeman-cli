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

var inspectCmd = cli.Command{
	Name:            "inspect",
	Usage:           "Get instance details by ID or name",
	ArgsUsage:       "<instance>",
	Action:          handleInspect,
	HideHelpCommand: true,
}

func handleInspect(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman inspect <instance>")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	instanceID, err := ResolveInstance(ctx, &client, args[0])
	if err != nil {
		return err
	}

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var raw []byte
	opts = append(opts, option.WithResponseBodyInto(&raw))
	_, err = client.Instances.Get(ctx, instanceID, opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(raw)
	return ShowJSON(os.Stdout, "instance inspect", obj, format, transform)
}
