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

var healthCmd = cli.Command{
	Name:  "health",
	Usage: "Check API server health",
	Description: `Report the health of the hypeman API server.

Examples:
  # Check health (default)
  hypeman health

  # Check health as JSON
  hypeman health --format json`,
	Action:          handleHealth,
	HideHelpCommand: true,
}

func handleHealth(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	_, err := client.Health.Check(ctx, opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.ParseBytes(res)

	if format == "auto" || format == "" {
		fmt.Println(obj.Get("status").String())
		return nil
	}

	return ShowJSON(os.Stdout, "health", obj, format, transform)
}
