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

var ingressesCreate = requestflag.WithInnerFlags(cli.Command{
	Name:  "create",
	Usage: "Create ingress",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "name",
			Usage:    "Human-readable name (lowercase letters, digits, and dashes only; cannot start or end with a dash)",
			Required: true,
			BodyPath: "name",
		},
		&requestflag.Flag[[]map[string]any]{
			Name:     "rule",
			Usage:    "Routing rules for this ingress",
			Required: true,
			BodyPath: "rules",
		},
	},
	Action:          handleIngressesCreate,
	HideHelpCommand: true,
}, map[string][]requestflag.HasOuterFlag{
	"rule": {
		&requestflag.InnerFlag[map[string]any]{
			Name:       "rule.match",
			InnerField: "match",
		},
		&requestflag.InnerFlag[map[string]any]{
			Name:       "rule.target",
			InnerField: "target",
		},
		&requestflag.InnerFlag[bool]{
			Name:       "rule.redirect-http",
			Usage:      "Auto-create HTTP to HTTPS redirect for this hostname (only applies when tls is enabled)",
			InnerField: "redirect_http",
		},
		&requestflag.InnerFlag[bool]{
			Name:       "rule.tls",
			Usage:      "Enable TLS termination (certificate auto-issued via ACME).",
			InnerField: "tls",
		},
	},
})

var ingressesList = cli.Command{
	Name:            "list",
	Usage:           "List ingresses",
	Flags:           []cli.Flag{},
	Action:          handleIngressesList,
	HideHelpCommand: true,
}

var ingressesDelete = cli.Command{
	Name:  "delete",
	Usage: "Delete ingress",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
	},
	Action:          handleIngressesDelete,
	HideHelpCommand: true,
}

var ingressesGet = cli.Command{
	Name:  "get",
	Usage: "Get ingress details",
	Flags: []cli.Flag{
		&requestflag.Flag[string]{
			Name:     "id",
			Required: true,
		},
	},
	Action:          handleIngressesGet,
	HideHelpCommand: true,
}

func handleIngressesCreate(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	unusedArgs := cmd.Args().Slice()

	if len(unusedArgs) > 0 {
		return fmt.Errorf("Unexpected extra arguments: %v", unusedArgs)
	}

	params := hypeman.IngressNewParams{}

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
	_, err = client.Ingresses.New(ctx, params, options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "ingresses create", obj, format, transform)
}

func handleIngressesList(ctx context.Context, cmd *cli.Command) error {
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
	_, err = client.Ingresses.List(ctx, options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "ingresses list", obj, format, transform)
}

func handleIngressesDelete(ctx context.Context, cmd *cli.Command) error {
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

	return client.Ingresses.Delete(ctx, cmd.Value("id").(string), options...)
}

func handleIngressesGet(ctx context.Context, cmd *cli.Command) error {
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
	_, err = client.Ingresses.Get(ctx, cmd.Value("id").(string), options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "ingresses get", obj, format, transform)
}
