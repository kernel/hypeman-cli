// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/kernel/hypeman-cli/internal/apiquery"
	"github.com/kernel/hypeman-cli/internal/binaryparam"
	"github.com/kernel/hypeman-cli/internal/requestflag"
	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var volumesCreate = cli.Command{
	Name:    "create",
	Usage:   "Creates a new empty volume of the specified size.",
	Suggest: true,
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "name",
			Usage:    "Volume name",
			Required: true,
			BodyPath: "name",
		},
		&requestflag.Flag[int64]{
			Name:     "size-gb",
			Usage:    "Size in gigabytes",
			Required: true,
			BodyPath: "size_gb",
		},
		&requestflag.Flag[string]{
			Name:     "id",
			Usage:    "Optional custom identifier (auto-generated if not provided)",
			BodyPath: "id",
		},
	},
	Action:          handleVolumesCreate,
	HideHelpCommand: true,
}

var volumesList = cli.Command{
	Name:            "list",
	Usage:           "List volumes",
	Suggest:         true,
	Flags:           []cli.Flag{},
	Action:          handleVolumesList,
	HideHelpCommand: true,
}

var volumesDelete = cli.Command{
	Name:    "delete",
	Usage:   "Delete volume",
	Suggest: true,
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
	},
	Action:          handleVolumesDelete,
	HideHelpCommand: true,
}

var volumesCreateFromArchive = cli.Command{
	Name:    "create-from-archive",
	Usage:   "Creates a new volume pre-populated with content from a tar.gz archive. The\narchive is streamed directly into the volume's root directory.",
	Suggest: true,
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:      "name",
			Usage:     "Volume name",
			Required:  true,
			QueryPath: "name",
		},
		&requestflag.Flag[int64]{
			Name:      "size-gb",
			Usage:     "Maximum size in GB (extraction fails if content exceeds this)",
			Required:  true,
			QueryPath: "size_gb",
		},
		&requestflag.Flag[string]{
			Name:     "body",
			Required: true,
			BodyRoot: true,
		},
		&requestflag.Flag[string]{
			Name:      "id",
			Usage:     "Optional custom volume ID (auto-generated if not provided)",
			QueryPath: "id",
		},
	},
	Action:          handleVolumesCreateFromArchive,
	HideHelpCommand: true,
}

var volumesGet = cli.Command{
	Name:    "get",
	Usage:   "Get volume details",
	Suggest: true,
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
	},
	Action:          handleVolumesGet,
	HideHelpCommand: true,
}

func handleVolumesCreate(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()

	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	params := hypeman.VolumeNewParams{}

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
	_, err = client.Volumes.New(ctx, params, options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "volumes create", obj, format, transform)
}

func handleVolumesList(ctx context.Context, cmd *cli.Command) error {
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
	_, err = client.Volumes.List(ctx, options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "volumes list", obj, format, transform)
}

func handleVolumesDelete(ctx context.Context, cmd *cli.Command) error {
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

	return client.Volumes.Delete(ctx, cmd.Value("id").(string), options...)
}

func handleVolumesCreateFromArchive(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()
	if !cmd.IsSet("body") && len(unusedArgs) > 0 {
		cmd.Set("body", unusedArgs[0])
		unusedArgs = unusedArgs[1:]
	}
	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	bodyReader, stdinInUse, err := binaryparam.FileOrStdin(os.Stdin, cmd.Value("body").(string))
	if err != nil {
		return fmt.Errorf("Failed on param '%s': %w", "body", err)
	}
	defer bodyReader.Close()

	params := hypeman.VolumeNewFromArchiveParams{}

	options, err := flagOptions(
		cmd,
		apiquery.NestedQueryFormatBrackets,
		apiquery.ArrayQueryFormatComma,
		ApplicationOctetStream,
		stdinInUse,
	)
	if err != nil {
		return err
	}

	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Volumes.NewFromArchive(
		ctx,
		bodyReader,
		params,
		options...,
	)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "volumes create-from-archive", obj, format, transform)
}

func handleVolumesGet(ctx context.Context, cmd *cli.Command) error {
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
	_, err = client.Volumes.Get(ctx, cmd.Value("id").(string), options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "volumes get", obj, format, transform)
}
