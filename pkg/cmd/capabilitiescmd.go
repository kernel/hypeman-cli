package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v3"
)

var capabilitiesCmd = cli.Command{
	Name:    "capabilities",
	Aliases: []string{"capability"},
	Usage:   "Show machine-readable host capabilities",
	Description: `Report server and API version, host OS/architecture, every runtime available on
this host with its per-runtime feature IDs, the configured default runtime and
whether it is available, guest networking model and host gateway, supported
image platforms, and stable server-level feature IDs.

Runtime-derived values reflect the actual host (for example, snapshot and
standby support on macOS is gated on the host OS version), so clients can gate
behavior on capabilities without hard-coding hypervisor knowledge.

Examples:
  # Show capabilities (default table format)
  hypeman capabilities

  # Show capabilities as JSON
  hypeman capabilities --format json

  # Show only the runtimes this host supports
  hypeman capabilities --transform runtimes`,
	Action:          handleCapabilities,
	HideHelpCommand: true,
}

func handleCapabilities(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	_, err := client.Capabilities.Get(ctx, opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	if format == "auto" || format == "" {
		return showCapabilities(os.Stdout, res)
	}

	obj := gjson.ParseBytes(res)
	return ShowJSON(os.Stdout, "capabilities", obj, format, transform)
}

func showCapabilities(w io.Writer, data []byte) error {
	obj := gjson.ParseBytes(data)

	server := obj.Get("server")
	fmt.Fprintln(w, "SERVER")
	fmt.Fprintf(w, "  Version:      %s\n", orDash(server.Get("version").String()))
	fmt.Fprintf(w, "  API version:  %s\n", orDash(server.Get("api_version").String()))

	host := obj.Get("host")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "HOST")
	fmt.Fprintf(w, "  OS:           %s\n", orDash(host.Get("os").String()))
	fmt.Fprintf(w, "  Arch:         %s\n", orDash(host.Get("arch").String()))

	defaultRuntime := obj.Get("default_runtime")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "DEFAULT RUNTIME")
	fmt.Fprintf(w, "  Name:         %s\n", orDash(defaultRuntime.Get("name").String()))
	fmt.Fprintf(w, "  Available:    %s\n", yesNo(defaultRuntime.Get("available").Bool()))

	runtimes := obj.Get("runtimes")
	if runtimes.IsArray() && len(runtimes.Array()) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "RUNTIMES")
		table := NewTableWriter(w, "NAME", "AVAILABLE", "FEATURES")
		table.TruncOrder = []int{2}
		runtimes.ForEach(func(_, value gjson.Result) bool {
			table.AddRow(
				value.Get("name").String(),
				yesNo(value.Get("available").Bool()),
				orDash(joinStrings(value.Get("features"))),
			)
			return true
		})
		table.Render()
	}

	images := obj.Get("images")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "IMAGES")
	fmt.Fprintf(w, "  Default platform:  %s\n", orDash(images.Get("default_platform").String()))
	fmt.Fprintf(w, "  Platforms:         %s\n", orDash(joinStrings(images.Get("platforms"))))

	network := obj.Get("network")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "NETWORK")
	fmt.Fprintf(w, "  Model:             %s\n", orDash(network.Get("model").String()))
	fmt.Fprintf(w, "  Gateway:           %s\n", orDash(network.Get("gateway").String()))
	fmt.Fprintf(w, "  Subnet:            %s\n", orDash(network.Get("subnet").String()))
	fmt.Fprintf(w, "  Guest to guest:    %s\n", yesNo(network.Get("guest_to_guest").Bool()))

	fmt.Fprintln(w)
	fmt.Fprintln(w, "SERVER FEATURES")
	fmt.Fprintf(w, "  %s\n", orDash(joinStrings(obj.Get("features"))))

	return nil
}

func joinStrings(arr gjson.Result) string {
	if !arr.IsArray() {
		return ""
	}
	values := make([]string, 0, len(arr.Array()))
	arr.ForEach(func(_, value gjson.Result) bool {
		values = append(values, value.String())
		return true
	})
	return strings.Join(values, ", ")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
