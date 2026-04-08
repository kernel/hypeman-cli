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

func handleAutoStandbyStatus(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman auto-standby status <instance>")
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
	_, err = client.Instances.AutoStandby.Status(ctx, instanceID, opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "auto-standby status", gjson.ParseBytes(res), format, transform)
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
