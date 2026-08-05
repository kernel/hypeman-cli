package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/urfave/cli/v3"
)

var inspectCmd = cli.Command{
	Name:      "inspect",
	Usage:     "Get instance details by ID or name",
	ArgsUsage: "<instance>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "show-env",
			Usage: "Show environment variable values (default: hidden)",
		},
	},
	Action:          handleInspect,
	HideHelpCommand: true,
}

func handleInspect(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance ID or name required\nUsage: hypeman inspect <instance>")
	}

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	instanceID, err := ResolveInstance(ctx, &client, args[0])
	if err != nil {
		return err
	}

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	instance, err := client.Instances.Get(ctx, instanceID, opts...)
	if err != nil {
		return err
	}

	// Render from the raw server response rather than the typed struct so fields
	// the server adds ahead of an SDK bump still surface here.
	raw := instance.RawJSON()
	if raw == "" {
		return fmt.Errorf("instance response did not include a raw payload")
	}

	if !cmd.Bool("show-env") {
		raw = redactEnvValues(raw)
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	obj := gjson.Parse(raw)
	return ShowJSON(os.Stdout, "instance inspect", obj, format, transform)
}

// redactEnvValues replaces every value under the top-level "env" object with
// "[hidden]" in the raw JSON, preserving the keys. On any sjson error it falls
// back to the original payload so inspect never fails on redaction alone.
func redactEnvValues(raw string) string {
	env := gjson.Get(raw, "env")
	if !env.Exists() || !env.IsObject() {
		return raw
	}
	out := raw
	var setErr error
	env.ForEach(func(key, _ gjson.Result) bool {
		updated, err := sjson.Set(out, "env."+escapeSJSONKey(key.String()), "[hidden]")
		if err != nil {
			setErr = err
			return false
		}
		out = updated
		return true
	})
	if setErr != nil {
		return raw
	}
	return out
}

// escapeSJSONKey escapes characters sjson treats specially in a path (the '.'
// separator plus the '*', '?', '|', '#', and '@' wildcards/modifiers) so env var
// names containing them address the intended key.
func escapeSJSONKey(key string) string {
	for _, special := range []string{".", "*", "?", "|", "#", "@"} {
		key = strings.ReplaceAll(key, special, "\\"+special)
	}
	return key
}
