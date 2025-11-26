package cmd

import (
	"context"
	"fmt"

	"github.com/onkernel/hypeman-go"
	"github.com/onkernel/hypeman-go/option"
	"github.com/urfave/cli/v3"
)

var logsCmd = cli.Command{
	Name:      "logs",
	Usage:     "Fetch the logs of an instance",
	ArgsUsage: "<instance>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "follow",
			Aliases: []string{"f"},
			Usage:   "Follow log output",
		},
		&cli.IntFlag{
			Name:  "tail",
			Usage: "Number of lines to show from the end of the logs",
			Value: 100,
		},
	},
	Action:          handleLogs,
	HideHelpCommand: true,
}

func handleLogs(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID required\nUsage: hypeman logs [flags] <instance>")
	}

	instanceID := args[0]

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	params := hypeman.InstanceStreamLogsParams{}
	if cmd.IsSet("follow") {
		params.Follow = hypeman.Opt(cmd.Bool("follow"))
	}
	if cmd.IsSet("tail") {
		params.Tail = hypeman.Opt(int64(cmd.Int("tail")))
	}

	stream := client.Instances.StreamLogsStreaming(
		ctx,
		instanceID,
		params,
		option.WithMiddleware(debugMiddleware(cmd.Root().Bool("debug"))),
	)
	defer stream.Close()

	for stream.Next() {
		fmt.Printf("%s\n", stream.Current())
	}

	return stream.Err()
}


