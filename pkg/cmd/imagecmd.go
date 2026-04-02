package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var imageCmd = cli.Command{
	Name:    "image",
	Aliases: []string{"images"},
	Usage:   "Manage images",
	Commands: []*cli.Command{
		&imageCreateCmd,
		&imageListCmd,
		&imageGetCmd,
		&imageDeleteCmd,
	},
	HideHelpCommand: true,
}

var imageCreateCmd = cli.Command{
	Name:            "create",
	Usage:           "Pull and convert an OCI image",
	ArgsUsage:       "<name>",
	Flags:           imageCreateFlags(),
	Action:          handleImageCreate,
	HideHelpCommand: true,
}

var imageListCmd = cli.Command{
	Name:  "list",
	Usage: "List images",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Only display image names",
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "Filter by tag key-value pair (KEY=VALUE, can be repeated)",
		},
	},
	Action:          handleImageList,
	HideHelpCommand: true,
}

var imageGetCmd = cli.Command{
	Name:            "get",
	Usage:           "Get image details",
	ArgsUsage:       "<name>",
	Action:          handleImageGet,
	HideHelpCommand: true,
}

var imageDeleteCmd = cli.Command{
	Name:            "delete",
	Aliases:         []string{"rm"},
	Usage:           "Delete an image",
	ArgsUsage:       "<name>",
	Action:          handleImageDelete,
	HideHelpCommand: true,
}

func handleImageList(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	params := hypeman.ImageListParams{}
	tags, malformedTags := parseKeyValueSpecs(cmd.StringSlice("tag"))
	for _, malformed := range malformedTags {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag filter: %s\n", malformed)
	}
	if len(tags) > 0 {
		params.Tags = tags
	}

	if format != "auto" {
		var res []byte
		opts = append(opts, option.WithResponseBodyInto(&res))
		_, err := client.Images.List(ctx, params, opts...)
		if err != nil {
			return err
		}
		obj := gjson.ParseBytes(res)
		return ShowJSON(os.Stdout, "image list", obj, format, transform)
	}

	images, err := client.Images.List(ctx, params, opts...)
	if err != nil {
		return err
	}

	quietMode := cmd.Bool("quiet")

	if quietMode {
		for _, img := range *images {
			fmt.Println(img.Name)
		}
		return nil
	}

	if len(*images) == 0 {
		fmt.Fprintln(os.Stderr, "No images found.")
		return nil
	}

	table := NewTableWriter(os.Stdout, "NAME", "STATUS", "DIGEST", "SIZE", "CREATED")
	table.TruncOrder = []int{0, 2, 4} // NAME first, then DIGEST, CREATED
	for _, img := range *images {
		digest := img.Digest
		if len(digest) > 19 {
			digest = digest[:19]
		}

		size := "-"
		if img.SizeBytes > 0 {
			size = formatBytes(img.SizeBytes)
		}

		table.AddRow(
			img.Name,
			string(img.Status),
			digest,
			size,
			FormatTimeAgo(img.CreatedAt),
		)
	}
	table.Render()

	return nil
}

func handleImageCreate(ctx context.Context, cmd *cli.Command) error {
	return handleImageCreateLike(ctx, cmd, "hypeman image create <name>", "image create")
}

func handleImageCreateLike(ctx context.Context, cmd *cli.Command, usageLine, outputLabel string) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("image name required\nUsage: %s", usageLine)
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	params, malformedTags := buildImageNewParams(args[0], cmd.StringSlice("tag"))
	for _, malformed := range malformedTags {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag: %s\n", malformed)
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
		_, err := client.Images.New(ctx, params, opts...)
		if err != nil {
			return err
		}
		obj := gjson.ParseBytes(res)
		return ShowJSON(os.Stdout, outputLabel, obj, format, transform)
	}

	result, err := client.Images.New(ctx, params, opts...)
	if err != nil {
		return err
	}
	fmt.Println(result.Name)
	return nil
}

func imageCreateFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: "Set image tag key-value pair (KEY=VALUE, can be repeated)",
		},
	}
}

func buildImageNewParams(name string, tagSpecs []string) (hypeman.ImageNewParams, []string) {
	params := hypeman.ImageNewParams{Name: name}

	tags, malformedTags := parseKeyValueSpecs(tagSpecs)
	if len(tags) > 0 {
		params.Tags = tags
	}

	return params, malformedTags
}

func handleImageGet(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("image name required\nUsage: hypeman image get <name>")
	}

	name := args[0]

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	_, err := client.Images.Get(ctx, url.PathEscape(name), opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(res)
	return ShowJSON(os.Stdout, "image get", obj, format, transform)
}

func handleImageDelete(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("image name required\nUsage: hypeman image delete <name>")
	}

	name := args[0]

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	err := client.Images.Delete(ctx, url.PathEscape(name), opts...)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Deleted image %s\n", name)
	return nil
}
