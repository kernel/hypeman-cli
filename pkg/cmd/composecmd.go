package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kernel/hypeman-cli/lib/compose"
	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var composeCmd = cli.Command{
	Name:  "compose",
	Usage: "Apply a small declarative workload file",
	Commands: []*cli.Command{
		&composePlanCmd,
		&composeUpCmd,
		&composeDownCmd,
	},
	HideHelpCommand: true,
}

var composePlanCmd = cli.Command{
	Name:  "plan",
	Usage: "Show compose changes without applying them",
	Flags: composeFileFlags(),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		runner, err := newComposeRunner(cmd)
		if err != nil {
			return err
		}
		result, err := runner.Plan(ctx)
		if err != nil {
			return err
		}
		return showComposeResult(cmd, "compose plan", result)
	},
	HideHelpCommand: true,
}

var composeUpCmd = cli.Command{
	Name:  "up",
	Usage: "Create or update compose resources",
	Flags: append(composeFileFlags(),
		&cli.BoolFlag{
			Name:  "replace",
			Usage: "Recreate resources when immutable fields changed",
		},
		&cli.BoolFlag{
			Name:  "wait",
			Usage: "Wait for newly created instances to reach Running",
			Value: true,
		},
		&cli.StringFlag{
			Name:  "wait-timeout",
			Usage: `Maximum wait per instance (e.g. "30s", "2m")`,
			Value: "2m",
		},
	),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		runner, err := newComposeRunner(cmd)
		if err != nil {
			return err
		}
		result, err := runner.Up(ctx, compose.UpOptions{
			Replace:     cmd.Bool("replace"),
			Wait:        cmd.Bool("wait"),
			WaitTimeout: cmd.String("wait-timeout"),
			Verbose:     cmd.Root().String("format") == "auto",
		})
		if err != nil {
			return err
		}
		if cmd.Root().String("format") != "auto" {
			return showComposeResult(cmd, "compose up", result)
		}
		printComposeDone(result)
		return nil
	},
	HideHelpCommand: true,
}

var composeDownCmd = cli.Command{
	Name:  "down",
	Usage: "Delete resources owned by a compose file",
	Flags: composeFileFlags(),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		runner, err := newComposeRunner(cmd)
		if err != nil {
			return err
		}
		result, err := runner.Down(ctx, cmd.Root().String("format") == "auto")
		if err != nil {
			return err
		}
		if cmd.Root().String("format") != "auto" {
			return showComposeResult(cmd, "compose down", result)
		}
		printComposeDone(result)
		return nil
	},
	HideHelpCommand: true,
}

func composeFileFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "file",
			Aliases: []string{"f"},
			Usage:   "Compose file to apply",
			Value:   "hypeman.compose.yaml",
		},
	}
}

func newComposeRunner(cmd *cli.Command) (*compose.Runner, error) {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)
	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}
	return compose.NewRunner(cmd.String("file"), client, opts...)
}

func showComposeResult(cmd *cli.Command, title string, result compose.Plan) error {
	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")
	if format == "auto" {
		printComposePlan(result)
		return nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return ShowJSON(os.Stdout, title, gjson.ParseBytes(data), format, transform)
}

func printComposePlan(result compose.Plan) {
	fmt.Fprintf(os.Stdout, "Compose file: %s\n", result.File)
	fmt.Fprintf(os.Stdout, "Name: %s\n\n", result.Name)
	if len(result.Actions) == 0 {
		fmt.Fprintln(os.Stdout, "No resources found.")
		fmt.Fprintln(os.Stdout)
		printComposeSummary(result.Summary)
		return
	}
	table := NewTableWriter(os.Stdout, "ACTION", "TYPE", "NAME", "REASON")
	table.TruncOrder = []int{2, 3}
	for _, action := range result.Actions {
		table.AddRow(action.Action, action.Type, action.Name, action.Reason)
	}
	table.Render()
	fmt.Fprintln(os.Stdout)
	printComposeSummary(result.Summary)
}

func printComposeDone(result compose.Plan) {
	fmt.Fprintln(os.Stderr)
	printComposeSummaryTo(os.Stderr, "Done", result.Summary)
}

func printComposeSummary(summary compose.Summary) {
	printComposeSummaryTo(os.Stdout, "Summary", summary)
}

func printComposeSummaryTo(out *os.File, label string, summary compose.Summary) {
	parts := []string{}
	for _, part := range []struct {
		count int
		label string
	}{
		{summary.Create, "to create"},
		{summary.Replace, "to replace"},
		{summary.Delete, "to delete"},
		{summary.Unchanged, "unchanged"},
		{summary.Skip, "skipped"},
		{summary.Conflict, "conflicts"},
	} {
		if part.count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", part.count, part.label))
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "no changes")
	}
	fmt.Fprintf(out, "%s: %s\n", label, strings.Join(parts, ", "))
}
