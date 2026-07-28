package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var autoStandbyCmd = cli.Command{
	Name:    "auto-standby",
	Aliases: []string{"autostandby"},
	Usage:   "Inspect auto-standby configuration and status",
	Commands: []*cli.Command{
		&autoStandbyStatusCmd,
		&autoStandbyHoldCmd,
	},
	HideHelpCommand: true,
}

var autoStandbyStatusCmd = cli.Command{
	Name:            "status",
	Usage:           "Get auto-standby status for an instance",
	ArgsUsage:       "<instance>",
	Action:          handleAutoStandbyStatus,
	HideHelpCommand: true,
}

var autoStandbyHoldCmd = cli.Command{
	Name:      "hold",
	Usage:     "Hold off auto-standby for an instance",
	ArgsUsage: "<instance>",
	Description: `Place a hold that prevents the auto-standby controller from putting the instance
into standby before the returned HOLD UNTIL time, and cancel any queued standby attempt.

Use this before opening a connection to a candidate-idle instance: success means it
is safe to connect until HOLD UNTIL, while a conflict error means the instance is in
standby (or irrevocably entering it) and must be restored first.

Instances where auto-standby is disabled, unconfigured, or unsupported succeed and
report their current status, because no auto-standby will occur.

Examples:
  hypeman auto-standby hold my-instance
  hypeman auto-standby hold my-instance --format json`,
	Action:          handleAutoStandbyHold,
	HideHelpCommand: true,
}

func handleAutoStandbyStatus(ctx context.Context, cmd *cli.Command) error {
	return runAutoStandbyAction(ctx, cmd, "status", func(ctx context.Context, client *hypeman.Client, instanceID string, opts []option.RequestOption) error {
		_, err := client.Instances.AutoStandby.Status(ctx, instanceID, opts...)
		return err
	})
}

func handleAutoStandbyHold(ctx context.Context, cmd *cli.Command) error {
	return runAutoStandbyAction(ctx, cmd, "hold", func(ctx context.Context, client *hypeman.Client, instanceID string, opts []option.RequestOption) error {
		_, err := client.Instances.AutoStandby.Hold(ctx, instanceID, opts...)
		return err
	})
}

// runAutoStandbyAction resolves the instance argument, invokes an auto-standby
// endpoint that returns an AutoStandbyStatus, and renders the response.
func runAutoStandbyAction(
	ctx context.Context,
	cmd *cli.Command,
	subcommand string,
	call func(ctx context.Context, client *hypeman.Client, instanceID string, opts []option.RequestOption) error,
) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman auto-standby %s <instance>", subcommand)
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

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	if err := call(ctx, &client, instanceID, opts); err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(res)

	if format == "auto" {
		status := obj.Get("status").String()
		enabled := obj.Get("enabled").Bool()
		configured := obj.Get("configured").Bool()
		supported := obj.Get("supported").Bool()
		idleTimeout := obj.Get("idle_timeout").String()
		reason := obj.Get("reason").String()
		trackingMode := obj.Get("tracking_mode").String()
		connections := obj.Get("active_inbound_connections").Int()

		if idleTimeout == "" {
			idleTimeout = "-"
		}
		if reason == "" {
			reason = "-"
		}

		fmt.Printf("%-14s %s\n", "STATUS", status)
		fmt.Printf("%-14s %t\n", "ENABLED", enabled)
		fmt.Printf("%-14s %t\n", "CONFIGURED", configured)
		fmt.Printf("%-14s %t\n", "SUPPORTED", supported)
		fmt.Printf("%-14s %s\n", "IDLE TIMEOUT", idleTimeout)
		fmt.Printf("%-14s %s\n", "REASON", reason)
		fmt.Printf("%-14s %s\n", "TRACKING", trackingMode)
		fmt.Printf("%-14s %d\n", "CONNECTIONS", connections)

		if idleSince := obj.Get("idle_since").String(); idleSince != "" {
			fmt.Printf("%-14s %s\n", "IDLE SINCE", idleSince)
		}
		if nextStandby := obj.Get("next_standby_at").String(); nextStandby != "" {
			fmt.Printf("%-14s %s\n", "NEXT STANDBY", nextStandby)
		}
		if holdUntil := obj.Get("hold_until").String(); holdUntil != "" {
			fmt.Printf("%-14s %s\n", "HOLD UNTIL", holdUntil)
		}
		return nil
	}

	return ShowJSON(os.Stdout, "auto-standby "+subcommand, obj, format, transform)
}

func buildAutoStandbyPolicy(cmd *cli.Command, prefix string) (hypeman.AutoStandbyPolicyParam, bool, error) {
	var policy hypeman.AutoStandbyPolicyParam

	enabledFlag := prefix + "enabled"
	idleTimeoutFlag := prefix + "idle-timeout"
	ignoreDestinationPortFlag := prefix + "ignore-destination-port"
	ignoreSourceCIDRFlag := prefix + "ignore-source-cidr"

	enabledSet := cmd.IsSet(enabledFlag)
	idleTimeout := cmd.String(idleTimeoutFlag)
	ignoreSourceCIDRs := cleanStringValues(cmd.StringSlice(ignoreSourceCIDRFlag))
	ignoreDestinationPorts, err := parseAutoStandbyPorts(cmd.StringSlice(ignoreDestinationPortFlag), ignoreDestinationPortFlag)
	if err != nil {
		return hypeman.AutoStandbyPolicyParam{}, false, err
	}

	if !enabledSet && idleTimeout == "" && len(ignoreDestinationPorts) == 0 && len(ignoreSourceCIDRs) == 0 {
		return hypeman.AutoStandbyPolicyParam{}, false, nil
	}

	if enabledSet {
		policy.Enabled = hypeman.Opt(cmd.Bool(enabledFlag))
	} else {
		policy.Enabled = hypeman.Opt(true)
	}

	if idleTimeout != "" {
		policy.IdleTimeout = hypeman.Opt(idleTimeout)
	}
	if len(ignoreDestinationPorts) > 0 {
		policy.IgnoreDestinationPorts = ignoreDestinationPorts
	}
	if len(ignoreSourceCIDRs) > 0 {
		policy.IgnoreSourceCidrs = ignoreSourceCIDRs
	}

	return policy, true, nil
}

func parseAutoStandbyPorts(rawPorts []string, flagName string) ([]int64, error) {
	ports := make([]int64, 0, len(rawPorts))
	for _, rawPort := range rawPorts {
		value := strings.TrimSpace(rawPort)
		if value == "" {
			continue
		}

		port, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid %s value %q: %w", flagName, rawPort, err)
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("%s must be between 1 and 65535: %q", flagName, rawPort)
		}

		ports = append(ports, port)
	}

	return ports, nil
}

func cleanStringValues(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		cleaned = append(cleaned, value)
	}

	return cleaned
}
