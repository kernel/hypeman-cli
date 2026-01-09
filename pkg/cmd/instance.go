// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/onkernel/hypeman-cli/internal/apiquery"
	"github.com/onkernel/hypeman-cli/internal/requestflag"
	"github.com/onkernel/hypeman-go"
	"github.com/onkernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var instancesCreate = requestflag.WithInnerFlags(cli.Command{
	Name:  "create",
	Usage: "Create and start instance",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "image",
			Usage:    "OCI image reference",
			Required: true,
			BodyPath: "image",
		},
		&requestflag.Flag[string]{
			Name:     "name",
			Usage:    "Human-readable name (lowercase letters, digits, and dashes only; cannot start or end with a dash)",
			Required: true,
			BodyPath: "name",
		},
		&requestflag.Flag[[]string]{
			Name:     "device",
			Usage:    "Device IDs or names to attach for GPU/PCI passthrough",
			BodyPath: "devices",
		},
		&requestflag.Flag[string]{
			Name:     "disk-io-bps",
			Usage:    `Disk I/O rate limit (e.g., "100MB/s", "500MB/s"). Defaults to proportional share based on CPU allocation if configured.`,
			BodyPath: "disk_io_bps",
		},
		&requestflag.Flag[map[string]any]{
			Name:     "env",
			Usage:    "Environment variables",
			BodyPath: "env",
		},
		&requestflag.Flag[string]{
			Name:     "hotplug-size",
			Usage:    `Additional memory for hotplug (human-readable format like "3GB", "1G")`,
			Default:  "3GB",
			BodyPath: "hotplug_size",
		},
		&requestflag.Flag[string]{
			Name:     "hypervisor",
			Usage:    "Hypervisor to use for this instance. Defaults to server configuration.",
			BodyPath: "hypervisor",
		},
		&requestflag.Flag[map[string]any]{
			Name:     "network",
			Usage:    "Network configuration for the instance",
			BodyPath: "network",
		},
		&requestflag.Flag[string]{
			Name:     "overlay-size",
			Usage:    `Writable overlay disk size (human-readable format like "10GB", "50G")`,
			Default:  "10GB",
			BodyPath: "overlay_size",
		},
		&requestflag.Flag[string]{
			Name:     "size",
			Usage:    `Base memory size (human-readable format like "1GB", "512MB", "2G")`,
			Default:  "1GB",
			BodyPath: "size",
		},
		&requestflag.Flag[int64]{
			Name:     "vcpus",
			Usage:    "Number of virtual CPUs",
			Default:  2,
			BodyPath: "vcpus",
		},
		&requestflag.Flag[[]map[string]any]{
			Name:     "volume",
			Usage:    "Volumes to attach to the instance at creation time",
			BodyPath: "volumes",
		},
	},
	Action:          handleInstancesCreate,
	HideHelpCommand: true,
}, map[string][]requestflag.HasOuterFlag{
	"network": {
		&requestflag.InnerFlag[string]{
			Name:       "network.bandwidth-download",
			Usage:      `Download bandwidth limit (external→VM, e.g., "1Gbps", "125MB/s"). Defaults to proportional share based on CPU allocation.`,
			InnerField: "bandwidth_download",
		},
		&requestflag.InnerFlag[string]{
			Name:       "network.bandwidth-upload",
			Usage:      `Upload bandwidth limit (VM→external, e.g., "1Gbps", "125MB/s"). Defaults to proportional share based on CPU allocation.`,
			InnerField: "bandwidth_upload",
		},
		&requestflag.InnerFlag[bool]{
			Name:       "network.enabled",
			Usage:      "Whether to attach instance to the default network",
			InnerField: "enabled",
		},
	},
	"volume": {
		&requestflag.InnerFlag[string]{
			Name:       "volume.mount-path",
			Usage:      "Path where volume is mounted in the guest",
			InnerField: "mount_path",
		},
		&requestflag.InnerFlag[string]{
			Name:       "volume.volume-id",
			Usage:      "Volume identifier",
			InnerField: "volume_id",
		},
		&requestflag.InnerFlag[bool]{
			Name:       "volume.overlay",
			Usage:      "Create per-instance overlay for writes (requires readonly=true)",
			InnerField: "overlay",
		},
		&requestflag.InnerFlag[string]{
			Name:       "volume.overlay-size",
			Usage:      `Max overlay size as human-readable string (e.g., "1GB"). Required if overlay=true.`,
			InnerField: "overlay_size",
		},
		&requestflag.InnerFlag[bool]{
			Name:       "volume.readonly",
			Usage:      "Whether volume is mounted read-only",
			InnerField: "readonly",
		},
	},
})

var instancesList = cli.Command{
	Name:            "list",
	Usage:           "List instances",
	Flags:           []cli.Flag{},
	Action:          handleInstancesList,
	HideHelpCommand: true,
}

var instancesGet = cli.Command{
	Name:  "get",
	Usage: "Get instance details",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
	},
	Action:          handleInstancesGet,
	HideHelpCommand: true,
}

var instancesLogs = cli.Command{
	Name:  "logs",
	Usage: "Streams instance logs as Server-Sent Events. Use the `source` parameter to\nselect which log to stream:",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
		&requestflag.Flag[bool]{
			Name:      "follow",
			Usage:     "Continue streaming new lines after initial output",
			QueryPath: "follow",
		},
		&requestflag.Flag[string]{
			Name:      "source",
			Usage:     "Log source to stream:\n- app: Guest application logs (serial console output)\n- vmm: Cloud Hypervisor VMM logs (hypervisor stdout+stderr)\n- hypeman: Hypeman operations log (actions taken on this instance)\n",
			Default:   "app",
			QueryPath: "source",
		},
		&requestflag.Flag[int64]{
			Name:      "tail",
			Usage:     "Number of lines to return from end",
			Default:   100,
			QueryPath: "tail",
		},
	},
	Action:          handleInstancesLogs,
	HideHelpCommand: true,
}

var instancesRestore = cli.Command{
	Name:  "restore",
	Usage: "Restore instance from standby",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
	},
	Action:          handleInstancesRestore,
	HideHelpCommand: true,
}

var instancesStandby = cli.Command{
	Name:  "standby",
	Usage: "Put instance in standby (pause, snapshot, delete VMM)",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
	},
	Action:          handleInstancesStandby,
	HideHelpCommand: true,
}

var instancesStart = cli.Command{
	Name:  "start",
	Usage: "Start a stopped instance",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
	},
	Action:          handleInstancesStart,
	HideHelpCommand: true,
}

var instancesStat = cli.Command{
	Name:  "stat",
	Usage: "Returns information about a path in the guest filesystem. Useful for checking if\na path exists, its type, and permissions before performing file operations.",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
		&requestflag.Flag[string]{
			Name:      "path",
			Usage:     "Path to stat in the guest filesystem",
			Required:  true,
			QueryPath: "path",
		},
		&requestflag.Flag[bool]{
			Name:      "follow-links",
			Usage:     "Follow symbolic links (like stat vs lstat)",
			QueryPath: "follow_links",
		},
	},
	Action:          handleInstancesStat,
	HideHelpCommand: true,
}

var instancesStop = cli.Command{
	Name:  "stop",
	Usage: "Stop instance (graceful shutdown)",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
	},
	Action:          handleInstancesStop,
	HideHelpCommand: true,
}

func handleInstancesCreate(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()

	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	params := hypeman.InstanceNewParams{}

	options, err := flagOptions(
		cmd,
		apiquery.NestedQueryFormatBrackets,
		apiquery.ArrayQueryFormatComma,
		ApplicationJSON,
		false,
	)
	if err != nil {
		return err
	}

	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Instances.New(ctx, params, options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "instances create", obj, format, transform)
}

func handleInstancesList(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()

	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	options, err := flagOptions(
		cmd,
		apiquery.NestedQueryFormatBrackets,
		apiquery.ArrayQueryFormatComma,
		EmptyBody,
		false,
	)
	if err != nil {
		return err
	}

	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Instances.List(ctx, options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "instances list", obj, format, transform)
}

func handleInstancesGet(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()
	if !cmd.IsSet("id") && len(unusedArgs) > 0 {
		cmd.Set("id", unusedArgs[0])
		unusedArgs = unusedArgs[1:]
	}
	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	options, err := flagOptions(
		cmd,
		apiquery.NestedQueryFormatBrackets,
		apiquery.ArrayQueryFormatComma,
		EmptyBody,
		false,
	)
	if err != nil {
		return err
	}

	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Instances.Get(ctx, cmd.Value("id").(string), options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "instances get", obj, format, transform)
}

func handleInstancesLogs(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()
	if !cmd.IsSet("id") && len(unusedArgs) > 0 {
		cmd.Set("id", unusedArgs[0])
		unusedArgs = unusedArgs[1:]
	}
	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	params := hypeman.InstanceLogsParams{}

	options, err := flagOptions(
		cmd,
		apiquery.NestedQueryFormatBrackets,
		apiquery.ArrayQueryFormatComma,
		EmptyBody,
		false,
	)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	stream := client.Instances.LogsStreaming(
		ctx,
		cmd.Value("id").(string),
		params,
		options...,
	)
	return ShowJSONIterator(os.Stdout, "instances logs", stream, format, transform)
}

func handleInstancesRestore(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()
	if !cmd.IsSet("id") && len(unusedArgs) > 0 {
		cmd.Set("id", unusedArgs[0])
		unusedArgs = unusedArgs[1:]
	}
	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	options, err := flagOptions(
		cmd,
		apiquery.NestedQueryFormatBrackets,
		apiquery.ArrayQueryFormatComma,
		EmptyBody,
		false,
	)
	if err != nil {
		return err
	}

	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Instances.Restore(ctx, cmd.Value("id").(string), options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "instances restore", obj, format, transform)
}

func handleInstancesStandby(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()
	if !cmd.IsSet("id") && len(unusedArgs) > 0 {
		cmd.Set("id", unusedArgs[0])
		unusedArgs = unusedArgs[1:]
	}
	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	options, err := flagOptions(
		cmd,
		apiquery.NestedQueryFormatBrackets,
		apiquery.ArrayQueryFormatComma,
		EmptyBody,
		false,
	)
	if err != nil {
		return err
	}

	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Instances.Standby(ctx, cmd.Value("id").(string), options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "instances standby", obj, format, transform)
}

func handleInstancesStart(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()
	if !cmd.IsSet("id") && len(unusedArgs) > 0 {
		cmd.Set("id", unusedArgs[0])
		unusedArgs = unusedArgs[1:]
	}
	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	options, err := flagOptions(
		cmd,
		apiquery.NestedQueryFormatBrackets,
		apiquery.ArrayQueryFormatComma,
		EmptyBody,
		false,
	)
	if err != nil {
		return err
	}

	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Instances.Start(ctx, cmd.Value("id").(string), options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "instances start", obj, format, transform)
}

func handleInstancesStat(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()
	if !cmd.IsSet("id") && len(unusedArgs) > 0 {
		cmd.Set("id", unusedArgs[0])
		unusedArgs = unusedArgs[1:]
	}
	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	params := hypeman.InstanceStatParams{}

	options, err := flagOptions(
		cmd,
		apiquery.NestedQueryFormatBrackets,
		apiquery.ArrayQueryFormatComma,
		EmptyBody,
		false,
	)
	if err != nil {
		return err
	}

	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Instances.Stat(
		ctx,
		cmd.Value("id").(string),
		params,
		options...,
	)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "instances stat", obj, format, transform)
}

func handleInstancesStop(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()
	if !cmd.IsSet("id") && len(unusedArgs) > 0 {
		cmd.Set("id", unusedArgs[0])
		unusedArgs = unusedArgs[1:]
	}
	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	options, err := flagOptions(
		cmd,
		apiquery.NestedQueryFormatBrackets,
		apiquery.ArrayQueryFormatComma,
		EmptyBody,
		false,
	)
	if err != nil {
		return err
	}

	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Instances.Stop(ctx, cmd.Value("id").(string), options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "instances stop", obj, format, transform)
}
