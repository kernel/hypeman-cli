package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/urfave/cli/v3"
)

var psCmd = cli.Command{
	Name:  "ps",
	Usage: "List instances",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Show all instances (default: running only)",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Only display instance IDs",
		},
		&cli.StringFlag{
			Name:  "state",
			Usage: "Filter instances by state (e.g., Running, Stopped, Standby)",
		},
		&cli.StringSliceFlag{
			Name:  "metadata",
			Usage: "Filter by metadata key-value pair (KEY=VALUE, can be repeated)",
		},
	},
	Action:          handlePs,
	HideHelpCommand: true,
}

func handlePs(ctx context.Context, cmd *cli.Command) error {
	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	params := hypeman.InstanceListParams{}
	stateFilter := cmd.String("state")
	if stateFilter != "" {
		params.State = hypeman.InstanceListParamsState(stateFilter)
	}

	metadataFilters, malformedMetadata := parseMetadataFilters(cmd.StringSlice("metadata"))
	for _, malformed := range malformedMetadata {
		fmt.Fprintf(os.Stderr, "Warning: ignoring malformed metadata filter: %s\n", malformed)
	}
	if len(metadataFilters) > 0 {
		params.Metadata = metadataFilters
	}

	instances, err := client.Instances.List(
		ctx,
		params,
		opts...,
	)
	if err != nil {
		return err
	}

	showAll := cmd.Bool("all")
	quietMode := cmd.Bool("quiet")
	serverSideFilterActive := stateFilter != "" || len(metadataFilters) > 0

	// Filter instances client-side only when no server-side filter is active
	var filtered []hypeman.Instance
	for _, inst := range *instances {
		if showAll || serverSideFilterActive || inst.State == "Running" {
			filtered = append(filtered, inst)
		}
	}

	// Quiet mode - just IDs
	if quietMode {
		for _, inst := range filtered {
			fmt.Println(inst.ID)
		}
		return nil
	}

	// Table output
	if len(filtered) == 0 {
		if !showAll && !serverSideFilterActive {
			fmt.Fprintln(os.Stderr, "No running instances. Use -a to show all.")
		}
		return nil
	}

	table := NewTableWriter(os.Stdout, "INSTANCE ID", "NAME", "IMAGE", "STATE", "GPU", "HV", "CREATED")
	table.TruncOrder = []int{2, 4, 6, 1} // IMAGE first, then GPU, CREATED, NAME
	for _, inst := range filtered {
		table.AddRow(
			TruncateID(inst.ID),
			inst.Name,
			inst.Image,
			string(inst.State),
			formatGPU(inst.GPU),
			formatHypervisor(inst.Hypervisor),
			FormatTimeAgo(inst.CreatedAt),
		)
	}
	table.Render()

	return nil
}

// formatGPU returns a short representation of GPU configuration
func formatGPU(gpu hypeman.InstanceGPU) string {
	// Check if GPU profile is set
	if gpu.Profile != "" {
		return gpu.Profile
	}
	// Check if mdev UUID is set (indicates vGPU without profile name shown)
	if gpu.MdevUuid != "" {
		return "vgpu"
	}
	return "-"
}

// formatHypervisor returns a short abbreviation for the hypervisor
func formatHypervisor(hv hypeman.InstanceHypervisor) string {
	switch hv {
	case hypeman.InstanceHypervisorCloudHypervisor:
		return "ch"
	case hypeman.InstanceHypervisorQemu:
		return "qemu"
	case hypeman.InstanceHypervisorFirecracker:
		return "fc"
	case hypeman.InstanceHypervisorVz:
		return "vz"
	default:
		if hv == "" {
			return "ch" // default
		}
		return string(hv)
	}
}

func parseMetadataFilters(specs []string) (map[string]string, []string) {
	metadata := make(map[string]string)
	var malformed []string

	for _, spec := range specs {
		parts := strings.SplitN(spec, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			malformed = append(malformed, spec)
			continue
		}
		metadata[parts[0]] = parts[1]
	}

	return metadata, malformed
}
