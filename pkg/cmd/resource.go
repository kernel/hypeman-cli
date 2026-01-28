// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/onkernel/hypeman-cli/internal/apiquery"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var resourcesGet = cli.Command{
	Name:            "get",
	Usage:           "Returns current host resource capacity, allocation status, and per-instance\nbreakdown. Resources include CPU, memory, disk, and network. Oversubscription\nratios are applied to calculate effective limits.",
	Suggest:         true,
	Flags:           []cli.Flag{},
	Action:          handleResourcesGet,
	HideHelpCommand: true,
}

func handleResourcesGet(ctx context.Context, cmd *cli.Command) error {
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
	_, err = client.Resources.Get(ctx, options...)
	if err != nil {
		return err
	}

	obj := gjson.ParseBytes(res)
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	return ShowJSON(os.Stdout, "resources get", obj, format, transform)
}
