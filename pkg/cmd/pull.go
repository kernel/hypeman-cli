package cmd

import (
	"context"

	"github.com/urfave/cli/v3"
)

var pullCmd = cli.Command{
	Name:            "pull",
	Usage:           "Alias for `image create`",
	ArgsUsage:       "<image>",
	Flags:           imageCreateFlags(),
	Action:          handlePull,
	HideHelpCommand: true,
}

func handlePull(ctx context.Context, cmd *cli.Command) error {
	return handleImageCreateLike(ctx, cmd, "hypeman pull <image>", "pull")
}
