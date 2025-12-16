// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/kernel/hypeman-cli/internal/apiquery"
	"github.com/kernel/hypeman-cli/internal/requestflag"
	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var instancesCreate = cli.Command{
	Name:  "create",
	Usage: "Create and start instance",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "image",
			Usage:    "OCI image reference",
			BodyPath: "image",
		},
		&requestflag.Flag[string]{
			Name:     "name",
			Usage:    "Human-readable name (lowercase letters, digits, and dashes only; cannot start or end with a dash)",
			BodyPath: "name",
		},
		&requestflag.Flag[[]string]{
			Name:     "device",
			Usage:    "Device IDs or names to attach for GPU/PCI passthrough",
			BodyPath: "devices",
		},
		&requestflag.Flag[any]{
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
		&requestflag.Flag[any]{
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
		&requestflag.Flag[[]any]{
			Name:     "volume",
			Usage:    "Volumes to attach to the instance at creation time",
			BodyPath: "volumes",
		},
	},
	Action:          handleInstancesCreate,
	HideHelpCommand: true,
}

var instancesList = cli.Command{
	Name:            "list",
	Usage:           "List instances",
	Flags:           []cli.Flag{},
	Action:          handleInstancesList,
	HideHelpCommand: true,
}

var instancesDelete = cli.Command{
	Name:  "delete",
	Usage: "Stop and delete instance",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name: "id",
		},
	},
	Action:          handleInstancesDelete,
	HideHelpCommand: true,
}

var instancesGet = cli.Command{
	Name:  "get",
	Usage: "Get instance details",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name: "id",
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
			Name: "id",
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
			Name: "id",
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
			Name: "id",
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
			Name: "id",
		},
	},
	Action:          handleInstancesStart,
	HideHelpCommand: true,
}

var instancesStop = cli.Command{
	Name:  "stop",
	Usage: "Stop instance (graceful shutdown)",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name: "id",
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

func handleInstancesDelete(ctx context.Context, cmd *cli.Command) error {
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
	)
	if err != nil {
		return err
	}

	return client.Instances.Delete(ctx, cmd.Value("id").(string), options...)
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
	)
	if err != nil {
		return err
	}

	stream := client.Instances.LogsStreaming(
		ctx,
		cmd.Value("id").(string),
		params,
		options...,
	)
	defer stream.Close()
	for stream.Next() {
		fmt.Printf("%s\n", stream.Current())
	}
	return stream.Err()
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
	)
	if err != nil {
		return err
	}
	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Instances.Start(
		ctx, cmd.Value("id").(string),
		options...,
	)
	if err != nil {
		return err
	}

	json := gjson.Parse(string(res))
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "instances start", json, format, transform)
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
	)
	if err != nil {
		return err
	}
	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Instances.Stop(
		ctx, cmd.Value("id").(string),
		options...,
	)
	if err != nil {
		return err
	}

	json := gjson.Parse(string(res))
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "instances stop", json, format, transform)
}
