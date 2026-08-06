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

var builderCmd = cli.Command{
	Name:    "builder",
	Aliases: []string{"builders"},
	Usage:   "Manage persistent build cache builders",
	Description: `Manage builders, which own a persistent cache disk that backs builds.

A builder runs one build at a time; builds targeting the same builder are
serialized. Pass a builder to 'hypeman build' with --builder to reuse its cache
across builds.

Examples:
  # Create a builder with a 100GB cache disk
  hypeman builder create --name ci --disk-size 100

  # List builders
  hypeman builder list

  # Build using a builder's cache
  hypeman build --builder ci-builder-id ./myapp

  # Reset a builder's cache disk without losing its identity
  hypeman builder prune ci-builder-id`,
	Commands: []*cli.Command{
		&builderCreateCmd,
		&builderListCmd,
		&builderGetCmd,
		&builderDeleteCmd,
		&builderPruneCmd,
	},
	HideHelpCommand: true,
}

var builderCreateCmd = cli.Command{
	Name:        "create",
	Usage:       "Create a builder and its cache disk",
	Description: "Creates a builder and its cache disk. One build at a time runs per builder.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "id",
			Usage: "Optional caller-supplied identifier (auto-generated if not provided)",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "Optional non-unique display name",
		},
		&cli.IntFlag{
			Name:    "disk-size",
			Aliases: []string{"disk-size-gb"},
			Usage:   "Cache disk size in gigabytes (omit to use the server default)",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "Set builder tag key-value pair (KEY=VALUE, can be repeated)",
		},
	},
	Action:          handleBuilderCreate,
	HideHelpCommand: true,
}

var builderListCmd = cli.Command{
	Name:  "list",
	Usage: "List builders",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Only display builder IDs",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "Filter by tag key-value pair (KEY=VALUE, can be repeated)",
		},
	},
	Action:          handleBuilderList,
	HideHelpCommand: true,
}

var builderGetCmd = cli.Command{
	Name:            "get",
	Usage:           "Get builder details",
	ArgsUsage:       "<id>",
	Action:          handleBuilderGet,
	HideHelpCommand: true,
}

var builderDeleteCmd = cli.Command{
	Name:            "delete",
	Aliases:         []string{"rm"},
	Usage:           "Delete a builder",
	Description:     "Permanently deletes a builder and its cache disk.",
	ArgsUsage:       "<id>",
	Action:          handleBuilderDelete,
	HideHelpCommand: true,
}

var builderPruneCmd = cli.Command{
	Name:            "prune",
	Usage:           "Reset a builder's cache disk",
	Description:     "Resets the builder's cache disk. The builder transitions to pruning, then ready. Builder identity is preserved.",
	ArgsUsage:       "<id>",
	Action:          handleBuilderPrune,
	HideHelpCommand: true,
}

func handleBuilderCreate(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	params := hypeman.BuilderNewParams{}
	if v := cmd.String("id"); v != "" {
		params.ID = hypeman.Opt(v)
	}
	if v := cmd.String("name"); v != "" {
		params.Name = hypeman.Opt(v)
	}
	if cmd.IsSet("disk-size") {
		params.DiskSizeGB = hypeman.Opt(int64(cmd.Int("disk-size")))
	}
	tags, malformedTags := parseKeyValueSpecs(cmd.StringSlice("tag"))
	for _, malformed := range malformedTags {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag: %s\n", malformed)
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

	// WithResponseBodyInto captures the raw body but leaves the typed result nil, so
	// only request it for the formats that render the response verbatim.
	if format != "auto" && format != "" {
		var res []byte
		opts = append(opts, option.WithResponseBodyInto(&res))
		if _, err := client.Builders.New(ctx, params, opts...); err != nil {
			return err
		}
		return ShowJSON(os.Stdout, "builder create", gjson.ParseBytes(res), format, transform)
	}

	builder, err := client.Builders.New(ctx, params, opts...)
	if err != nil {
		return err
	}

	fmt.Println(builder.ID)
	return nil
}

func handleBuilderList(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	params := hypeman.BuilderListParams{}
	tags, malformedTags := parseKeyValueSpecs(cmd.StringSlice("tag"))
	for _, malformed := range malformedTags {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag filter: %s\n", malformed)
	}
	if len(tags) > 0 {
		params.Tags = tags
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	if format != "auto" && format != "" {
		var res []byte
		opts = append(opts, option.WithResponseBodyInto(&res))
		if _, err := client.Builders.List(ctx, params, opts...); err != nil {
			return err
		}
		return ShowJSON(os.Stdout, "builder list", gjson.ParseBytes(res), format, transform)
	}

	builders, err := client.Builders.List(ctx, params, opts...)
	if err != nil {
		return err
	}

	if cmd.Bool("quiet") {
		for _, b := range *builders {
			fmt.Println(b.ID)
		}
		return nil
	}

	return showBuilderListTable(*builders)
}

func showBuilderListTable(builders []hypeman.Builder) error {
	if len(builders) == 0 {
		fmt.Fprintln(os.Stderr, "No builders found.")
		return nil
	}

	table := NewTableWriter(os.Stdout, "ID", "NAME", "STATUS", "DISK", "ACTIVE BUILD", "QUEUED", "LAST USED", "CREATED")
	table.TruncOrder = []int{0, 1, 4} // ID first, then NAME, ACTIVE BUILD

	for _, b := range builders {
		name := b.Name
		if name == "" {
			name = "-"
		}

		activeBuild := b.ActiveBuildID
		if activeBuild == "" {
			activeBuild = "-"
		}

		table.AddRow(
			b.ID,
			name,
			string(b.Status),
			fmt.Sprintf("%d GB", b.DiskSizeGB),
			activeBuild,
			fmt.Sprintf("%d", len(b.QueuedBuilds)),
			FormatTimeAgo(b.LastUsedAt),
			FormatTimeAgo(b.CreatedAt),
		)
	}

	table.Render()
	return nil
}

func handleBuilderGet(ctx context.Context, cmd *cli.Command) error {
	id, err := requireBuilderID(cmd, "get")
	if err != nil {
		return err
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	if _, err := client.Builders.Get(ctx, id, opts...); err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(res)
	return ShowJSON(os.Stdout, "builder get", obj, format, transform)
}

func handleBuilderDelete(ctx context.Context, cmd *cli.Command) error {
	id, err := requireBuilderID(cmd, "delete")
	if err != nil {
		return err
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	if err := client.Builders.Delete(ctx, id, opts...); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Deleted builder %s\n", id)
	return nil
}

func handleBuilderPrune(ctx context.Context, cmd *cli.Command) error {
	id, err := requireBuilderID(cmd, "prune")
	if err != nil {
		return err
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	if format != "auto" && format != "" {
		var res []byte
		opts = append(opts, option.WithResponseBodyInto(&res))
		if _, err := client.Builders.Prune(ctx, id, opts...); err != nil {
			return err
		}
		return ShowJSON(os.Stdout, "builder prune", gjson.ParseBytes(res), format, transform)
	}

	builder, err := client.Builders.Prune(ctx, id, opts...)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Pruning builder %s (status: %s)\n", id, builder.Status)
	return nil
}

func requireBuilderID(cmd *cli.Command, action string) (string, error) {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return "", fmt.Errorf("builder ID required\nUsage: hypeman builder %s <id>", action)
	}
	return args[0], nil
}
