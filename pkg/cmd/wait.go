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

var waitCmd = cli.Command{
	Name:      "wait",
	Usage:     "Wait for an instance to reach a target state",
	ArgsUsage: "<instance>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "state",
			Usage:    `Target state: "Created", "Initializing", "Running", "Paused", "Shutdown", "Stopped", "Standby", or "Unknown"`,
			Required: true,
		},
		&cli.StringFlag{
			Name:  "timeout",
			Usage: `Maximum duration to wait (e.g., "30s", "2m")`,
		},
	},
	Action:          handleWait,
	HideHelpCommand: true,
}

func handleWait(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman wait <instance> --state <state>")
	}

	targetState, err := parseInstanceWaitState(cmd.String("state"))
	if err != nil {
		return err
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	instanceID, err := ResolveInstance(ctx, &client, args[0])
	if err != nil {
		return err
	}

	params := hypeman.InstanceWaitParams{
		State: targetState,
	}
	if timeout := cmd.String("timeout"); timeout != "" {
		params.Timeout = hypeman.Opt(timeout)
	}

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	_, err = client.Instances.Wait(ctx, instanceID, params, opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(res)

	if format == "auto" {
		state := obj.Get("state").String()
		timedOut := obj.Get("timed_out").Bool()
		stateError := obj.Get("state_error").String()

		if timedOut {
			fmt.Fprintf(os.Stderr, "Timed out waiting (last state: %s)\n", state)
			return fmt.Errorf("timed out waiting for instance to reach state %s", cmd.String("state"))
		}
		if stateError != "" {
			fmt.Printf("%-14s %s\n", "STATE", state)
			fmt.Printf("%-14s %s\n", "STATE ERROR", stateError)
		} else {
			fmt.Println(state)
		}
		return nil
	}

	return ShowJSON(os.Stdout, "instance wait", obj, format, transform)
}

func parseInstanceWaitState(raw string) (hypeman.InstanceWaitParamsState, error) {
	switch strings.ToLower(raw) {
	case "created":
		return hypeman.InstanceWaitParamsStateCreated, nil
	case "initializing":
		return hypeman.InstanceWaitParamsStateInitializing, nil
	case "running":
		return hypeman.InstanceWaitParamsStateRunning, nil
	case "paused":
		return hypeman.InstanceWaitParamsStatePaused, nil
	case "shutdown":
		return hypeman.InstanceWaitParamsStateShutdown, nil
	case "stopped":
		return hypeman.InstanceWaitParamsStateStopped, nil
	case "standby":
		return hypeman.InstanceWaitParamsStateStandby, nil
	case "unknown":
		return hypeman.InstanceWaitParamsStateUnknown, nil
	default:
		return "", fmt.Errorf("invalid state: %s (must be Created, Initializing, Running, Paused, Shutdown, Stopped, Standby, or Unknown)", raw)
	}
}
