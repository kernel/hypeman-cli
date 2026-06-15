package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/kernel/hypeman-cli/lib/compose"
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
  hypeman update auto-standby <instance> --enabled --idle-timeout 10m
  hypeman update egress-credentials <instance> --env KEY=VALUE
  hypeman update health-check <instance> --type http --http-port 8080
  hypeman update restart-policy <instance> --policy on_failure --max-attempts 5`,
	Commands: []*cli.Command{
		&updateAutoStandbyCmd,
		&updateEgressCredentialsCmd,
		&updateHealthCheckCmd,
		&updateRestartPolicyCmd,
	},
	HideHelpCommand: true,
}

var updateAutoStandbyCmd = cli.Command{
	Name:      "auto-standby",
	Usage:     "Update the auto-standby policy for an instance",
	ArgsUsage: "<instance>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "enabled",
			Usage: "Enable Linux-only automatic standby based on inbound TCP activity",
		},
		&cli.StringFlag{
			Name:  "idle-timeout",
			Usage: `How long the instance must be idle before entering standby (e.g., "10m")`,
		},
		&cli.StringSliceFlag{
			Name:  "ignore-destination-port",
			Usage: "TCP destination port that should not keep the instance awake (can be repeated)",
		},
		&cli.StringSliceFlag{
			Name:  "ignore-source-cidr",
			Usage: "Client CIDR that should not keep the instance awake (can be repeated)",
		},
	},
	Action:          handleUpdateAutoStandby,
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

var updateHealthCheckCmd = cli.Command{
	Name:            "health-check",
	Usage:           "Update the workload health check policy for an instance",
	ArgsUsage:       "<instance>",
	Flags:           healthCheckFlags(""),
	Action:          handleUpdateHealthCheck,
	HideHelpCommand: true,
}

var updateRestartPolicyCmd = cli.Command{
	Name:            "restart-policy",
	Usage:           "Update the restart supervision policy for an instance",
	ArgsUsage:       "<instance>",
	Flags:           restartPolicyFlags(""),
	Action:          handleUpdateRestartPolicy,
	HideHelpCommand: true,
}

func handleUpdateAutoStandby(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman update auto-standby <instance> [flags]")
	}

	policy, policySet, err := buildAutoStandbyPolicy(cmd, "")
	if err != nil {
		return err
	}
	if !policySet {
		return fmt.Errorf("at least one auto-standby flag is required")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	instanceID, err := ResolveInstance(ctx, &client, args[0])
	if err != nil {
		return err
	}

	params := hypeman.InstanceUpdateParams{
		AutoStandby: policy,
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
		return ShowJSON(os.Stdout, "update auto-standby", obj, format, transform)
	}

	fmt.Fprintf(os.Stderr, "Updating auto-standby for %s...\n", args[0])

	instance, err := client.Instances.Update(ctx, instanceID, params, opts...)
	if err != nil {
		return err
	}
	fmt.Println(instance.ID)
	return nil
}

func handleUpdateHealthCheck(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman update health-check <instance> [flags]")
	}

	input, set, err := parseHealthCheckInput(cmd, "")
	if err != nil {
		return err
	}
	if !set {
		return fmt.Errorf("at least one health-check flag is required")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	instanceID, err := ResolveInstance(ctx, &client, args[0])
	if err != nil {
		return err
	}

	params := hypeman.InstanceUpdateParams{
		HealthCheck: compose.BuildHealthCheckParam(input),
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
		return ShowJSON(os.Stdout, "update health-check", obj, format, transform)
	}

	fmt.Fprintf(os.Stderr, "Updating health-check for %s...\n", args[0])

	instance, err := client.Instances.Update(ctx, instanceID, params, opts...)
	if err != nil {
		return err
	}
	fmt.Println(instance.ID)
	return nil
}

func handleUpdateRestartPolicy(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman update restart-policy <instance> [flags]")
	}

	input, set := parseRestartPolicyInput(cmd, "")
	if !set {
		return fmt.Errorf("at least one restart-policy flag is required")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	instanceID, err := ResolveInstance(ctx, &client, args[0])
	if err != nil {
		return err
	}

	params := hypeman.InstanceUpdateParams{
		RestartPolicy: compose.BuildRestartPolicyParam(input),
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
		return ShowJSON(os.Stdout, "update restart-policy", obj, format, transform)
	}

	fmt.Fprintf(os.Stderr, "Updating restart-policy for %s...\n", args[0])

	instance, err := client.Instances.Update(ctx, instanceID, params, opts...)
	if err != nil {
		return err
	}
	fmt.Println(instance.ID)
	return nil
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
