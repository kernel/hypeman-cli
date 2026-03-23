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

var updateCmd = cli.Command{
	Name:  "update",
	Usage: "Update specific mutable instance configuration",
	Description: `Update mutable instance settings that have dedicated update flows.

Currently supported:
  hypeman update egress-credentials <instance> --env KEY=VALUE`,
	Commands: []*cli.Command{
		&updateEgressCredentialsCmd,
	},
	HideHelpCommand: true,
}

var updateEgressCredentialsCmd = cli.Command{
	Name:      "egress-credentials",
	Usage:     "Rotate env-backed credentials for existing mediated egress bindings",
	ArgsUsage: "<instance>",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "env",
			Aliases: []string{"e"},
			Usage:   "Update a bound credential env value (KEY=VALUE, can be repeated)",
		},
	},
	Action:          handleUpdate,
	HideHelpCommand: true,
}

func handleUpdate(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman update egress-credentials <instance> --env KEY=VALUE")
	}

	env, malformed := parseKeyValueSpecs(cmd.StringSlice("env"))
	for _, invalid := range malformed {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed env entry: %s\n", invalid)
	}
	if len(env) == 0 {
		return fmt.Errorf("at least one bound credential --env KEY=VALUE entry is required")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	instanceID, err := ResolveInstance(ctx, &client, args[0])
	if err != nil {
		return err
	}

	params := hypeman.InstanceUpdateParams{
		Env: env,
	}

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	if format != "auto" {
		var res []byte
		opts = append(opts, option.WithResponseBodyInto(&res))
		_, err := client.Instances.Update(ctx, instanceID, params, opts...)
		if err != nil {
			return err
		}
		obj := gjson.ParseBytes(res)
		return ShowJSON(os.Stdout, "instance update egress-credentials", obj, format, transform)
	}

	instance, err := client.Instances.Update(ctx, instanceID, params, opts...)
	if err != nil {
		return err
	}
	fmt.Println(instance.ID)
	return nil
}
