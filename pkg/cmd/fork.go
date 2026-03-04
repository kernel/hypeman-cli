package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var forkCmd = cli.Command{
	Name:      "fork",
	Usage:     "Fork an instance into a new instance",
	ArgsUsage: "<source> <name>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "from-running",
			Usage: "Allow forking from a running source by doing standby -> fork -> restore",
		},
		&cli.StringFlag{
			Name:  "target-state",
			Usage: "Target state for the forked instance: Stopped, Standby, or Running",
		},
	},
	Action:          handleFork,
	HideHelpCommand: true,
}

func handleFork(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 2 {
		return fmt.Errorf("source instance and target name required\nUsage: hypeman fork [flags] <source> <name>")
	}

	source := args[0]
	targetName := args[1]

	targetState, err := normalizeForkTargetState(cmd.String("target-state"))
	if err != nil {
		return err
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	sourceID, err := ResolveInstance(ctx, &client, source)
	if err != nil {
		return err
	}

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	params := hypeman.InstanceForkParams{
		Name: targetName,
	}
	if cmd.IsSet("from-running") {
		params.FromRunning = hypeman.Opt(cmd.Bool("from-running"))
	}
	if targetState != "" {
		params.TargetState = hypeman.InstanceForkParamsTargetState(targetState)
	}

	fmt.Fprintf(os.Stderr, "Forking %s to %s...\n", source, targetName)

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	var raw []byte
	if format != "auto" {
		opts = append(opts, option.WithResponseBodyInto(&raw))
	}

	forked, err := client.Instances.Fork(ctx, sourceID, params, opts...)
	if err != nil {
		return err
	}

	if format != "auto" {
		obj := gjson.ParseBytes(raw)
		return ShowJSON(os.Stdout, "instance fork", obj, format, transform)
	}

	fmt.Println(forked.ID)
	fmt.Fprintf(os.Stderr, "Forked %s as %s (state: %s)\n", source, forked.Name, forked.State)
	return nil
}

func normalizeForkTargetState(state string) (string, error) {
	switch strings.ToLower(state) {
	case "":
		return "", nil
	case "stopped":
		return "Stopped", nil
	case "standby":
		return "Standby", nil
	case "running":
		return "Running", nil
	default:
		return "", fmt.Errorf("invalid target state: %s (must be Stopped, Standby, or Running)", state)
	}
}
