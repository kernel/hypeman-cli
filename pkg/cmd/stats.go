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

var statsCmd = cli.Command{
	Name:      "stats",
	Usage:     "Show real-time resource utilization for a running instance",
	ArgsUsage: "<instance>",
	Description: `Display real-time CPU, memory, and network statistics for a running VM instance.

Examples:
  # Show stats for an instance
  hypeman stats my-instance

  # Show stats as JSON
  hypeman stats --format json my-instance`,
	Action:          handleStats,
	HideHelpCommand: true,
}

func handleStats(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("instance name or ID required\nUsage: hypeman stats <instance>")
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

	var res []byte
	opts = append(opts, option.WithResponseBodyInto(&res))
	_, err = client.Instances.Stats(ctx, instanceID, opts...)
	if err != nil {
		return err
	}

	format := cmd.Root().String("format")
	transform := cmd.Root().String("transform")

	if format == "auto" || format == "" {
		return showStatsTable(res)
	}

	obj := gjson.ParseBytes(res)
	return ShowJSON(os.Stdout, "instance stats", obj, format, transform)
}

func showStatsTable(data []byte) error {
	obj := gjson.ParseBytes(data)

	name := obj.Get("instance_name").String()
	id := obj.Get("instance_id").String()

	fmt.Printf("Instance: %s (%s)\n\n", name, TruncateID(id))

	fmt.Println("METRIC                 VALUE")
	fmt.Println("─────────────────────  ──────────────")

	vcpus := obj.Get("allocated_vcpus").Int()
	cpuSecs := obj.Get("cpu_seconds").Float()
	fmt.Printf("%-22s %d\n", "Allocated vCPUs", vcpus)
	fmt.Printf("%-22s %.2f s\n", "CPU Time", cpuSecs)

	allocMem := obj.Get("allocated_memory_bytes").Int()
	rssMem := obj.Get("memory_rss_bytes").Int()
	vmsMem := obj.Get("memory_vms_bytes").Int()
	fmt.Printf("%-22s %s\n", "Allocated Memory", formatBytes(allocMem))
	fmt.Printf("%-22s %s\n", "Memory RSS", formatBytes(rssMem))
	fmt.Printf("%-22s %s\n", "Memory VMS", formatBytes(vmsMem))

	utilRatio := obj.Get("memory_utilization_ratio")
	if utilRatio.Exists() && utilRatio.Type != gjson.Null {
		fmt.Printf("%-22s %.1f%%\n", "Memory Utilization", utilRatio.Float()*100)
	}

	rxBytes := obj.Get("network_rx_bytes").Int()
	txBytes := obj.Get("network_tx_bytes").Int()
	fmt.Printf("%-22s %s\n", "Network RX", formatBytes(rxBytes))
	fmt.Printf("%-22s %s\n", "Network TX", formatBytes(txBytes))

	return nil
}
