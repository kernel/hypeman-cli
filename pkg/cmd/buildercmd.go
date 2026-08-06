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

// The generated SDK does not expose a BuilderService yet, so these commands reach
// the /builders endpoints through the client's generic request methods. Replace
// them with client.Builders.* once that service ships.

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

type builderCreateRequest struct {
	ID         string            `json:"id,omitempty"`
	DiskSizeGB int64             `json:"disk_size_gb,omitempty"`
	Name       string            `json:"name,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
}

func handleBuilderCreate(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	body := builderCreateRequest{
		ID:   cmd.String("id"),
		Name: cmd.String("name"),
	}
	if cmd.IsSet("disk-size") {
		body.DiskSizeGB = int64(cmd.Int("disk-size"))
	}
	tags, malformedTags := parseKeyValueSpecs(cmd.StringSlice("tag"))
	for _, malformed := range malformedTags {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag: %s\n", malformed)
	}
	if len(tags) > 0 {
		body.Tags = tags
	}

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	if err := client.Post(ctx, "builders", body, nil, opts...); err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(res)
	if format == "auto" || format == "" {
		fmt.Println(obj.Get("id").String())
		return nil
	}

	return ShowJSON(os.Stdout, "builder create", obj, format, transform)
}

func handleBuilderList(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	tags, malformedTags := parseKeyValueSpecs(cmd.StringSlice("tag"))
	for _, malformed := range malformedTags {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag filter: %s\n", malformed)
	}
	for key, value := range tags {
		opts = append(opts, option.WithQuery(fmt.Sprintf("tags[%s]", key), value))
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	if err := client.Get(ctx, "builders", nil, nil, opts...); err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	builders := gjson.ParseBytes(res)

	if format != "auto" && format != "" {
		return ShowJSON(os.Stdout, "builder list", builders, format, transform)
	}

	if cmd.Bool("quiet") {
		builders.ForEach(func(_, value gjson.Result) bool {
			fmt.Println(value.Get("id").String())
			return true
		})
		return nil
	}

	return showBuilderListTable(builders)
}

func showBuilderListTable(builders gjson.Result) error {
	if !builders.IsArray() || len(builders.Array()) == 0 {
		fmt.Fprintln(os.Stderr, "No builders found.")
		return nil
	}

	table := NewTableWriter(os.Stdout, "ID", "NAME", "STATUS", "DISK", "ACTIVE BUILD", "QUEUED", "LAST USED", "CREATED")
	table.TruncOrder = []int{0, 1, 4} // ID first, then NAME, ACTIVE BUILD

	builders.ForEach(func(_, value gjson.Result) bool {
		name := value.Get("name").String()
		if name == "" {
			name = "-"
		}

		activeBuild := value.Get("active_build_id").String()
		if activeBuild == "" {
			activeBuild = "-"
		}

		table.AddRow(
			value.Get("id").String(),
			name,
			value.Get("status").String(),
			fmt.Sprintf("%d GB", value.Get("disk_size_gb").Int()),
			activeBuild,
			fmt.Sprintf("%d", len(value.Get("queued_builds").Array())),
			FormatTimeAgo(value.Get("last_used_at").Time()),
			FormatTimeAgo(value.Get("created_at").Time()),
		)
		return true
	})

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
	if err := client.Get(ctx, fmt.Sprintf("builders/%s", id), nil, nil, opts...); err != nil {
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

	opts := []option.RequestOption{option.WithHeader("Accept", "*/*")}
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	if err := client.Delete(ctx, fmt.Sprintf("builders/%s", id), nil, nil, opts...); err != nil {
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

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	if err := client.Post(ctx, fmt.Sprintf("builders/%s/prune", id), nil, nil, opts...); err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(res)
	if format == "auto" || format == "" {
		fmt.Fprintf(os.Stderr, "Pruning builder %s (status: %s)\n", id, obj.Get("status").String())
		return nil
	}

	return ShowJSON(os.Stdout, "builder prune", obj, format, transform)
}

func requireBuilderID(cmd *cli.Command, action string) (string, error) {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return "", fmt.Errorf("builder ID required\nUsage: hypeman builder %s <id>", action)
	}
	return args[0], nil
}
