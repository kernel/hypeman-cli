package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/kernel/hypeman-go/shared"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var snapshotCmd = cli.Command{
	Name:  "snapshot",
	Usage: "Manage instance snapshots",
	Commands: []*cli.Command{
		&snapshotCreateCmd,
		&snapshotRestoreCmd,
		&snapshotListCmd,
		&snapshotGetCmd,
		&snapshotDeleteCmd,
		&snapshotForkCmd,
	},
	HideHelpCommand: true,
}

var snapshotCreateCmd = cli.Command{
	Name:      "create",
	Usage:     "Create a snapshot for an instance",
	ArgsUsage: "<instance>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "kind",
			Usage: `Snapshot kind: "Standby" or "Stopped" (default: Standby)`,
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "Optional snapshot name",
		},
		&cli.BoolFlag{
			Name:  "compression-enabled",
			Usage: "Enable snapshot memory compression",
		},
		&cli.StringFlag{
			Name:  "compression-algorithm",
			Usage: `Snapshot compression algorithm: "zstd" or "lz4"`,
		},
		&cli.IntFlag{
			Name:  "compression-level",
			Usage: "Snapshot compression level (zstd: 1-19, lz4: 0-9)",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "Set snapshot tag key-value pair (KEY=VALUE, can be repeated)",
		},
	},
	Action:          handleSnapshotCreate,
	HideHelpCommand: true,
}

var snapshotRestoreCmd = cli.Command{
	Name:      "restore",
	Usage:     "Restore an instance from a snapshot",
	ArgsUsage: "<instance> <snapshot-id>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "target-hypervisor",
			Usage: `Optional hypervisor override: "cloud-hypervisor", "firecracker", "qemu", or "vz"`,
		},
		&cli.StringFlag{
			Name:  "target-state",
			Usage: `Optional final state: "Stopped", "Standby", or "Running"`,
		},
	},
	Action:          handleSnapshotRestore,
	HideHelpCommand: true,
}

var snapshotListCmd = cli.Command{
	Name:  "list",
	Usage: "List snapshots",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Only display snapshot IDs",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "Filter snapshots by snapshot name",
		},
		&cli.StringFlag{
			Name:  "source-instance-id",
			Usage: "Filter snapshots by source instance ID",
		},
		&cli.StringFlag{
			Name:  "kind",
			Usage: `Filter by kind: "Standby" or "Stopped"`,
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "Filter by tag key-value pair (KEY=VALUE, can be repeated)",
		},
	},
	Action:          handleSnapshotList,
	HideHelpCommand: true,
}

var snapshotGetCmd = cli.Command{
	Name:            "get",
	Usage:           "Get snapshot details",
	ArgsUsage:       "<snapshot-id>",
	Action:          handleSnapshotGet,
	HideHelpCommand: true,
}

var snapshotDeleteCmd = cli.Command{
	Name:            "delete",
	Aliases:         []string{"rm"},
	Usage:           "Delete a snapshot",
	ArgsUsage:       "<snapshot-id>",
	Action:          handleSnapshotDelete,
	HideHelpCommand: true,
}

var snapshotForkCmd = cli.Command{
	Name:      "fork",
	Usage:     "Fork a new instance from a snapshot",
	ArgsUsage: "<snapshot-id> <name>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "target-hypervisor",
			Usage: `Optional hypervisor override: "cloud-hypervisor", "firecracker", "qemu", or "vz"`,
		},
		&cli.StringFlag{
			Name:  "target-state",
			Usage: `Optional final state: "Stopped", "Standby", or "Running"`,
		},
	},
	Action:          handleSnapshotFork,
	HideHelpCommand: true,
}

func handleSnapshotCreate(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman snapshot create <instance>")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	instanceID, err := ResolveInstance(ctx, &client, args[0])
	if err != nil {
		return err
	}

	kind, err := parseSnapshotKind(cmd.String("kind"), hypeman.SnapshotKindStandby)
	if err != nil {
		return err
	}

	params := hypeman.InstanceSnapshotNewParams{
		Kind: kind,
	}
	if name := cmd.String("name"); name != "" {
		params.Name = hypeman.Opt(name)
	}

	tags, malformedTags := parseKeyValueSpecs(cmd.StringSlice("tag"))
	for _, malformed := range malformedTags {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag: %s\n", malformed)
	}
	if len(tags) > 0 {
		params.Tags = tags
	}

	if cmd.IsSet("compression-enabled") || cmd.IsSet("compression-algorithm") || cmd.IsSet("compression-level") {
		compression := shared.SnapshotCompressionConfigParam{
			Enabled: cmd.Bool("compression-enabled"),
		}
		if !cmd.IsSet("compression-enabled") {
			compression.Enabled = true
		}
		if cmd.IsSet("compression-level") {
			compression.Level = hypeman.Opt(int64(cmd.Int("compression-level")))
		}
		if algorithm := cmd.String("compression-algorithm"); algorithm != "" {
			switch strings.ToLower(algorithm) {
			case "zstd":
				compression.Algorithm = shared.SnapshotCompressionConfigAlgorithmZstd
			case "lz4":
				compression.Algorithm = shared.SnapshotCompressionConfigAlgorithmLz4
			default:
				return fmt.Errorf("invalid compression algorithm: %s (must be 'zstd' or 'lz4')", algorithm)
			}
		}
		params.Compression = compression
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
		_, err := client.Instances.Snapshots.New(ctx, instanceID, params, opts...)
		if err != nil {
			return err
		}
		obj := gjson.ParseBytes(res)
		return ShowJSON(os.Stdout, "snapshot create", obj, format, transform)
	}

	snapshot, err := client.Instances.Snapshots.New(ctx, instanceID, params, opts...)
	if err != nil {
		return err
	}
	fmt.Println(snapshot.ID)
	return nil
}

func handleSnapshotRestore(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 2 {
		return fmt.Errorf("instance and snapshot ID required\nUsage: hypeman snapshot restore <instance> <snapshot-id>")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	instanceID, err := ResolveInstance(ctx, &client, args[0])
	if err != nil {
		return err
	}
	snapshotID := args[1]

	params := hypeman.InstanceSnapshotRestoreParams{
		ID: instanceID,
	}
	if value := cmd.String("target-hypervisor"); value != "" {
		hypervisor, err := parseSnapshotTargetHypervisor(value)
		if err != nil {
			return err
		}
		params.TargetHypervisor = hypervisor
	}
	if value := cmd.String("target-state"); value != "" {
		state, err := parseSnapshotTargetState(value)
		if err != nil {
			return err
		}
		params.TargetState = state
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
		_, err := client.Instances.Snapshots.Restore(ctx, snapshotID, params, opts...)
		if err != nil {
			return err
		}
		obj := gjson.ParseBytes(res)
		return ShowJSON(os.Stdout, "snapshot restore", obj, format, transform)
	}

	instance, err := client.Instances.Snapshots.Restore(ctx, snapshotID, params, opts...)
	if err != nil {
		return err
	}
	fmt.Println(instance.ID)
	return nil
}

func handleSnapshotList(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	params := hypeman.SnapshotListParams{}
	if name := cmd.String("name"); name != "" {
		params.Name = hypeman.Opt(name)
	}
	if sourceInstanceID := cmd.String("source-instance-id"); sourceInstanceID != "" {
		params.SourceInstanceID = hypeman.Opt(sourceInstanceID)
	}
	if kindInput := cmd.String("kind"); kindInput != "" {
		kind, err := parseSnapshotKind(kindInput, "")
		if err != nil {
			return err
		}
		params.Kind = kind
	}
	tags, malformedTags := parseKeyValueSpecs(cmd.StringSlice("tag"))
	for _, malformed := range malformedTags {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag filter: %s\n", malformed)
	}
	if len(tags) > 0 {
		params.Tags = tags
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
		_, err := client.Snapshots.List(ctx, params, opts...)
		if err != nil {
			return err
		}
		obj := gjson.ParseBytes(res)
		return ShowJSON(os.Stdout, "snapshot list", obj, format, transform)
	}

	snapshots, err := client.Snapshots.List(ctx, params, opts...)
	if err != nil {
		return err
	}

	if cmd.Bool("quiet") {
		for _, snapshot := range *snapshots {
			fmt.Println(snapshot.ID)
		}
		return nil
	}

	if len(*snapshots) == 0 {
		fmt.Fprintln(os.Stderr, "No snapshots found.")
		return nil
	}

	table := NewTableWriter(os.Stdout, "ID", "NAME", "KIND", "SOURCE", "CREATED")
	table.TruncOrder = []int{0, 3}
	for _, snapshot := range *snapshots {
		name := snapshot.Name
		if name == "" {
			name = "-"
		}
		table.AddRow(
			TruncateID(snapshot.ID),
			name,
			string(snapshot.Kind),
			snapshot.SourceInstanceName,
			FormatTimeAgo(snapshot.CreatedAt),
		)
	}
	table.Render()
	return nil
}

func handleSnapshotGet(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("snapshot ID required\nUsage: hypeman snapshot get <snapshot-id>")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	_, err := client.Snapshots.Get(ctx, args[0], opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	obj := gjson.ParseBytes(res)
	return ShowJSON(os.Stdout, "snapshot get", obj, format, transform)
}

func handleSnapshotDelete(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("snapshot ID required\nUsage: hypeman snapshot delete <snapshot-id>")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	if err := client.Snapshots.Delete(ctx, args[0], opts...); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Deleted snapshot %s\n", args[0])
	return nil
}

func handleSnapshotFork(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 2 {
		return fmt.Errorf("snapshot ID and target name required\nUsage: hypeman snapshot fork <snapshot-id> <name>")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	snapshotID := args[0]
	targetName := args[1]

	params := hypeman.SnapshotForkParams{
		Name: targetName,
	}
	if value := cmd.String("target-hypervisor"); value != "" {
		hypervisor, err := parseSnapshotForkTargetHypervisor(value)
		if err != nil {
			return err
		}
		params.TargetHypervisor = hypervisor
	}
	if value := cmd.String("target-state"); value != "" {
		state, err := parseSnapshotForkTargetState(value)
		if err != nil {
			return err
		}
		params.TargetState = state
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
		_, err := client.Snapshots.Fork(ctx, snapshotID, params, opts...)
		if err != nil {
			return err
		}
		obj := gjson.ParseBytes(res)
		return ShowJSON(os.Stdout, "snapshot fork", obj, format, transform)
	}

	instance, err := client.Snapshots.Fork(ctx, snapshotID, params, opts...)
	if err != nil {
		return err
	}
	fmt.Println(instance.ID)
	return nil
}

func parseSnapshotKind(raw string, fallback hypeman.SnapshotKind) (hypeman.SnapshotKind, error) {
	switch strings.ToLower(raw) {
	case "":
		if fallback == "" {
			return "", nil
		}
		return fallback, nil
	case "standby":
		return hypeman.SnapshotKindStandby, nil
	case "stopped":
		return hypeman.SnapshotKindStopped, nil
	default:
		return "", fmt.Errorf("invalid snapshot kind: %s (must be Standby or Stopped)", raw)
	}
}

func parseSnapshotTargetState(raw string) (hypeman.InstanceSnapshotRestoreParamsTargetState, error) {
	switch strings.ToLower(raw) {
	case "stopped":
		return hypeman.InstanceSnapshotRestoreParamsTargetStateStopped, nil
	case "standby":
		return hypeman.InstanceSnapshotRestoreParamsTargetStateStandby, nil
	case "running":
		return hypeman.InstanceSnapshotRestoreParamsTargetStateRunning, nil
	default:
		return "", fmt.Errorf("invalid target state: %s (must be Stopped, Standby, or Running)", raw)
	}
}

func parseSnapshotTargetHypervisor(raw string) (hypeman.InstanceSnapshotRestoreParamsTargetHypervisor, error) {
	switch strings.ToLower(raw) {
	case "cloud-hypervisor", "ch":
		return hypeman.InstanceSnapshotRestoreParamsTargetHypervisorCloudHypervisor, nil
	case "firecracker", "fc":
		return hypeman.InstanceSnapshotRestoreParamsTargetHypervisorFirecracker, nil
	case "qemu":
		return hypeman.InstanceSnapshotRestoreParamsTargetHypervisorQemu, nil
	case "vz":
		return hypeman.InstanceSnapshotRestoreParamsTargetHypervisorVz, nil
	default:
		return "", fmt.Errorf("invalid target hypervisor: %s (must be cloud-hypervisor, firecracker, qemu, or vz)", raw)
	}
}

func parseSnapshotForkTargetState(raw string) (hypeman.SnapshotForkParamsTargetState, error) {
	switch strings.ToLower(raw) {
	case "stopped":
		return hypeman.SnapshotForkParamsTargetStateStopped, nil
	case "standby":
		return hypeman.SnapshotForkParamsTargetStateStandby, nil
	case "running":
		return hypeman.SnapshotForkParamsTargetStateRunning, nil
	default:
		return "", fmt.Errorf("invalid target state: %s (must be Stopped, Standby, or Running)", raw)
	}
}

func parseSnapshotForkTargetHypervisor(raw string) (hypeman.SnapshotForkParamsTargetHypervisor, error) {
	switch strings.ToLower(raw) {
	case "cloud-hypervisor", "ch":
		return hypeman.SnapshotForkParamsTargetHypervisorCloudHypervisor, nil
	case "firecracker", "fc":
		return hypeman.SnapshotForkParamsTargetHypervisorFirecracker, nil
	case "qemu":
		return hypeman.SnapshotForkParamsTargetHypervisorQemu, nil
	case "vz":
		return hypeman.SnapshotForkParamsTargetHypervisorVz, nil
	default:
		return "", fmt.Errorf("invalid target hypervisor: %s (must be cloud-hypervisor, firecracker, qemu, or vz)", raw)
	}
}
