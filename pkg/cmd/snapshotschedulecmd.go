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

var snapshotScheduleCmd = cli.Command{
	Name:  "schedule",
	Usage: "Manage scheduled snapshots for an instance",
	Commands: []*cli.Command{
		&snapshotScheduleSetCmd,
		&snapshotScheduleGetCmd,
		&snapshotScheduleDeleteCmd,
	},
	HideHelpCommand: true,
}

var snapshotScheduleSetCmd = cli.Command{
	Name:      "set",
	Usage:     "Create or update a snapshot schedule for an instance",
	ArgsUsage: "<instance>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "interval",
			Usage:    `Snapshot interval (Go duration format, minimum "1m")`,
			Required: true,
		},
		&cli.StringFlag{
			Name:  "max-age",
			Usage: `Delete scheduled snapshots older than this duration (e.g., "24h")`,
		},
		&cli.IntFlag{
			Name:  "max-count",
			Usage: "Keep at most this many scheduled snapshots (0 disables count-based cleanup)",
		},
		&cli.StringFlag{
			Name:  "name-prefix",
			Usage: "Prefix for generated scheduled snapshot names",
		},
		&cli.StringSliceFlag{
			Name:  "metadata",
			Usage: "Set schedule metadata key-value pair (KEY=VALUE, can be repeated)",
		},
	},
	Action:          handleSnapshotScheduleSet,
	HideHelpCommand: true,
}

var snapshotScheduleGetCmd = cli.Command{
	Name:            "get",
	Usage:           "Get the snapshot schedule for an instance",
	ArgsUsage:       "<instance>",
	Action:          handleSnapshotScheduleGet,
	HideHelpCommand: true,
}

var snapshotScheduleDeleteCmd = cli.Command{
	Name:            "delete",
	Aliases:         []string{"rm"},
	Usage:           "Delete the snapshot schedule for an instance",
	ArgsUsage:       "<instance>",
	Action:          handleSnapshotScheduleDelete,
	HideHelpCommand: true,
}

func handleSnapshotScheduleSet(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman snapshot schedule set <instance> --interval <duration>")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	instanceID, err := ResolveInstance(ctx, &client, args[0])
	if err != nil {
		return err
	}

	request, malformedMetadata, err := buildSnapshotScheduleRequest(cmd)
	if err != nil {
		return err
	}
	for _, malformed := range malformedMetadata {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed metadata entry: %s\n", malformed)
	}

	params := hypeman.InstanceSnapshotScheduleUpdateParams{
		SetSnapshotScheduleRequest: request,
	}

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	_, err = client.Instances.SnapshotSchedule.Update(ctx, instanceID, params, opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(res)

	if format == "auto" {
		printSnapshotScheduleSummary(obj)
		return nil
	}

	return ShowJSON(os.Stdout, "snapshot schedule set", obj, format, transform)
}

func handleSnapshotScheduleGet(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman snapshot schedule get <instance>")
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
	_, err = client.Instances.SnapshotSchedule.Get(ctx, instanceID, opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(res)

	if format == "auto" {
		printSnapshotScheduleSummary(obj)
		return nil
	}

	return ShowJSON(os.Stdout, "snapshot schedule get", obj, format, transform)
}

func handleSnapshotScheduleDelete(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman snapshot schedule delete <instance>")
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

	if err := client.Instances.SnapshotSchedule.Delete(ctx, instanceID, opts...); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Deleted snapshot schedule for %s\n", args[0])
	return nil
}

func buildSnapshotScheduleRequest(cmd *cli.Command) (hypeman.SetSnapshotScheduleRequestParam, []string, error) {
	if !cmd.IsSet("max-age") && !cmd.IsSet("max-count") {
		return hypeman.SetSnapshotScheduleRequestParam{}, nil, fmt.Errorf("at least one of --max-age or --max-count is required")
	}

	request := hypeman.SetSnapshotScheduleRequestParam{
		Interval:  cmd.String("interval"),
		Retention: hypeman.SnapshotScheduleRetentionParam{},
	}

	if maxAge := cmd.String("max-age"); maxAge != "" {
		request.Retention.MaxAge = hypeman.Opt(maxAge)
	}
	if cmd.IsSet("max-count") {
		request.Retention.MaxCount = hypeman.Opt(int64(cmd.Int("max-count")))
	}
	if namePrefix := cmd.String("name-prefix"); namePrefix != "" {
		request.NamePrefix = hypeman.Opt(namePrefix)
	}

	metadata, malformedMetadata := parseKeyValueSpecs(cmd.StringSlice("metadata"))
	if len(metadata) > 0 {
		request.Metadata = metadata
	}

	return request, malformedMetadata, nil
}

func printSnapshotScheduleSummary(obj gjson.Result) {
	instanceID := obj.Get("instance_id").String()
	interval := obj.Get("interval").String()
	maxAge := obj.Get("retention.max_age").String()
	maxCount := obj.Get("retention.max_count").Int()
	namePrefix := obj.Get("name_prefix").String()
	nextRun := obj.Get("next_run_at").String()
	createdAt := obj.Get("created_at").String()

	if maxAge == "" {
		maxAge = "-"
	}
	if namePrefix == "" {
		namePrefix = "-"
	}
	if nextRun == "" {
		nextRun = "-"
	}

	maxCountStr := "-"
	if maxCount > 0 {
		maxCountStr = fmt.Sprintf("%d", maxCount)
	}

	fmt.Printf("%-14s %s\n", "INSTANCE", instanceID)
	fmt.Printf("%-14s %s\n", "INTERVAL", interval)
	fmt.Printf("%-14s %s\n", "MAX AGE", maxAge)
	fmt.Printf("%-14s %s\n", "MAX COUNT", maxCountStr)
	fmt.Printf("%-14s %s\n", "PREFIX", namePrefix)
	fmt.Printf("%-14s %s\n", "NEXT RUN", nextRun)
	fmt.Printf("%-14s %s\n", "CREATED", createdAt)

	metadata := obj.Get("metadata")
	if metadata.Exists() && metadata.IsObject() {
		fmt.Printf("%-14s", "METADATA")
		first := true
		metadata.ForEach(func(key, value gjson.Result) bool {
			if first {
				fmt.Printf(" %s=%s\n", key.String(), value.String())
				first = false
			} else {
				fmt.Printf("%-14s %s=%s\n", "", key.String(), value.String())
			}
			return true
		})
		if first {
			fmt.Println(" -")
		}
	}
}
