// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/onkernel/hypeman-cli/internal/apiquery"
	"github.com/onkernel/hypeman-cli/internal/requestflag"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var buildsCreate = cli.Command{
	Name:    "create",
	Usage:   "Creates a new build job. Source code should be uploaded as a tar.gz archive in\nthe multipart form data.",
	Suggest: true,
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "source",
			Usage:    "Source tarball (tar.gz) containing application code and optionally a Dockerfile",
			Required: true,
			BodyPath: "source",
		},
		&requestflag.Flag[string]{
			Name:     "base-image-digest",
			Usage:    "Optional pinned base image digest",
			BodyPath: "base_image_digest",
		},
		&requestflag.Flag[string]{
			Name:     "cache-scope",
			Usage:    "Tenant-specific cache key prefix",
			BodyPath: "cache_scope",
		},
		&requestflag.Flag[string]{
			Name:     "dockerfile",
			Usage:    "Dockerfile content. Required if not included in the source tarball.",
			BodyPath: "dockerfile",
		},
		&requestflag.Flag[string]{
			Name:     "global-cache-key",
			Usage:    "Global cache identifier (e.g., \"node\", \"python\", \"ubuntu\", \"browser\").\nWhen specified, the build will import from cache/global/{key}.\nAdmin builds will also export to this location.\n",
			BodyPath: "global_cache_key",
		},
		&requestflag.Flag[string]{
			Name:     "is-admin-build",
			Usage:    "Set to \"true\" to grant push access to global cache (operator-only).\nAdmin builds can populate the shared global cache that all tenant builds read from.\n",
			BodyPath: "is_admin_build",
		},
		&requestflag.Flag[string]{
			Name:     "secrets",
			Usage:    "JSON array of secret references to inject during build.\nEach object has \"id\" (required) for use with --mount=type=secret,id=...\nExample: [{\"id\": \"npm_token\"}, {\"id\": \"github_token\"}]\n",
			BodyPath: "secrets",
		},
		&requestflag.Flag[int64]{
			Name:     "timeout-seconds",
			Usage:    "Build timeout (default 600)",
			BodyPath: "timeout_seconds",
		},
	},
	Action:          handleBuildsCreate,
	HideHelpCommand: true,
}

var buildsList = cli.Command{
	Name:            "list",
	Usage:           "List builds",
	Suggest:         true,
	Flags:           []cli.Flag{},
	Action:          handleBuildsList,
	HideHelpCommand: true,
}

var buildsEvents = cli.Command{
	Name:    "events",
	Usage:   "Streams build events as Server-Sent Events. Events include:",
	Suggest: true,
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
		&requestflag.Flag[bool]{
			Name:      "follow",
			Usage:     "Continue streaming new events after initial output",
			QueryPath: "follow",
		},
	},
	Action:          handleBuildsEvents,
	HideHelpCommand: true,
}

var buildsGet = cli.Command{
	Name:    "get",
	Usage:   "Get build details",
	Suggest: true,
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
	},
	Action:          handleBuildsGet,
	HideHelpCommand: true,
}

func handleBuildsCreate(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()

	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	params := hypeman.BuildNewParams{}

	options, err := flagOptions(
		cmd,
		apiquery.NestedQueryFormatBrackets,
		apiquery.ArrayQueryFormatComma,
		MultipartFormEncoded,
		false,
	)
	if err != nil {
		return err
	}

	var res []byte
	options = append(options, option.WithResponseBodyInto(&res))
	_, err = client.Builds.New(ctx, params, options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "builds create", obj, format, transform)
}

func handleBuildsList(ctx context.Context, cmd *cli.Command) error {
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
	_, err = client.Builds.List(ctx, options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "builds list", obj, format, transform)
}

func handleBuildsEvents(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()
	if !cmd.IsSet("id") && len(unusedArgs) > 0 {
		cmd.Set("id", unusedArgs[0])
		unusedArgs = unusedArgs[1:]
	}
	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	params := hypeman.BuildEventsParams{}

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
	stream := client.Builds.EventsStreaming(
		ctx,
		cmd.Value("id").(string),
		params,
		options...,
	)
	return ShowJSONIterator(os.Stdout, "builds events", stream, format, transform)
}

func handleBuildsGet(ctx context.Context, cmd *cli.Command) error {
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
	_, err = client.Builds.Get(ctx, cmd.Value("id").(string), options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "builds get", obj, format, transform)
}
