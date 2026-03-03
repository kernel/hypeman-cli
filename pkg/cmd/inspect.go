package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var inspectCmd = cli.Command{
	Name:      "inspect",
	Usage:     "Get instance details by ID or name",
	ArgsUsage: "<instance>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "show-env",
			Usage: "Show environment variable values (default: hidden)",
		},
	},
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

	instance, err := client.Instances.Get(ctx, instanceID, opts...)
	if err != nil {
		return err
	}

	if !cmd.Bool("show-env") {
		instance.Env = redactEnvValues(instance.Env)
	}

	raw, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("failed to encode instance response: %w", err)
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(raw)
	return ShowJSON(os.Stdout, "instance inspect", obj, format, transform)
}

func redactEnvValues(env map[string]string) map[string]string {
	if len(env) == 0 {
		return env
	}

	redacted := make(map[string]string, len(env))
	for key := range env {
		redacted[key] = "[hidden]"
	}

	return redacted
}
