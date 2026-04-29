package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kernel/hypeman-go"
	"github.com/kernel/hypeman-go/option"
	"github.com/kernel/hypeman-go/shared"
	"github.com/urfave/cli/v3"
)

var runCmd = cli.Command{
	Name:      "run",
	Usage:     "Create and start a new instance from an image",
	ArgsUsage: "<image>",
	Description: `Create and start a new virtual machine instance from an OCI image.

Examples:
  # Basic run
  hypeman run myimage:latest

  # Run with custom resources
  hypeman run --cpus 4 --memory 8GB myimage:latest

  # Run with vGPU
  hypeman run --gpu-profile L40S-1Q myimage:latest

  # Run with GPU passthrough
  hypeman run --device my-gpu myimage:latest

  # Run with QEMU hypervisor
  hypeman run --hypervisor qemu myimage:latest

  # Run with bandwidth limits
  hypeman run --bandwidth-down 1Gbps --bandwidth-up 500Mbps myimage:latest`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "Instance name (auto-generated if not provided)",
		},
		&cli.StringSliceFlag{
			Name:    "env",
			Aliases: []string{"e"},
			Usage:   "Set environment variable (KEY=VALUE, can be repeated)",
		},
		&cli.StringFlag{
			Name:  "credentials-json",
			Usage: "Credential policy map as JSON (keyed by guest-visible env var)",
		},
		&cli.StringFlag{
			Name:  "memory",
			Usage: `Base memory size (e.g., "1GB", "512MB")`,
			Value: "1GB",
		},
		&cli.IntFlag{
			Name:  "cpus",
			Usage: "Number of virtual CPUs",
			Value: 2,
		},
		&cli.StringFlag{
			Name:  "overlay-size",
			Usage: `Writable overlay disk size (e.g., "10GB")`,
			Value: "10GB",
		},
		&cli.StringFlag{
			Name:  "hotplug-size",
			Usage: `Additional memory for hotplug (e.g., "3GB")`,
			Value: "3GB",
		},
		&cli.BoolFlag{
			Name:  "network",
			Usage: "Enable network (default: true)",
			Value: true,
		},
		// GPU/vGPU flags
		&cli.StringFlag{
			Name:  "gpu-profile",
			Usage: `vGPU profile name (e.g., "L40S-1Q", "L40S-2Q")`,
		},
		&cli.StringSliceFlag{
			Name:  "device",
			Usage: "Device ID or name for PCI/GPU passthrough (can be repeated)",
		},
		// Hypervisor flag
		&cli.StringFlag{
			Name:  "hypervisor",
			Usage: `Hypervisor to use: "cloud-hypervisor", "firecracker", "qemu", or "vz"`,
		},
		// Resource limit flags
		&cli.StringFlag{
			Name:  "disk-io",
			Usage: `Disk I/O rate limit (e.g., "100MB/s", "500MB/s")`,
		},
		&cli.StringFlag{
			Name:  "bandwidth-down",
			Usage: `Download bandwidth limit (e.g., "1Gbps", "125MB/s")`,
		},
		&cli.StringFlag{
			Name:  "bandwidth-up",
			Usage: `Upload bandwidth limit (e.g., "1Gbps", "125MB/s")`,
		},
		&cli.BoolFlag{
			Name:  "network-egress-enabled",
			Usage: "Enable host-mediated outbound egress policy",
		},
		&cli.StringFlag{
			Name:  "network-egress-mode",
			Usage: `Egress enforcement mode: "all" or "http_https_only"`,
		},
		&cli.BoolFlag{
			Name:  "auto-standby-enabled",
			Usage: "Enable Linux-only automatic standby based on inbound TCP activity",
		},
		&cli.StringFlag{
			Name:  "auto-standby-idle-timeout",
			Usage: `How long the instance must be idle before entering standby (e.g., "10m")`,
		},
		&cli.StringSliceFlag{
			Name:  "auto-standby-ignore-destination-port",
			Usage: "TCP destination port that should not keep the instance awake (can be repeated)",
		},
		&cli.StringSliceFlag{
			Name:  "auto-standby-ignore-source-cidr",
			Usage: "Client CIDR that should not keep the instance awake (can be repeated)",
		},
		// Boot option flags
		&cli.BoolFlag{
			Name:  "skip-guest-agent",
			Usage: "Skip guest-agent installation during boot (exec and stat APIs will not work)",
		},
		&cli.BoolFlag{
			Name:  "skip-kernel-headers",
			Usage: "Skip kernel headers installation during boot for faster startup (DKMS will not work)",
		},
		&cli.BoolFlag{
			Name:  "snapshot-compression-enabled",
			Usage: "Enable snapshot memory compression for this instance policy",
		},
		&cli.StringFlag{
			Name:  "snapshot-compression-delay",
			Usage: `Delay before standby snapshot compression begins (e.g., "30s", "5m")`,
		},
		&cli.StringFlag{
			Name:  "snapshot-compression-algorithm",
			Usage: `Snapshot compression algorithm: "zstd" or "lz4"`,
		},
		&cli.IntFlag{
			Name:  "snapshot-compression-level",
			Usage: "Snapshot compression level (zstd: 1-19, lz4: 0-9)",
		},
		// Entrypoint and CMD overrides
		&cli.StringSliceFlag{
			Name:  "entrypoint",
			Usage: "Override image entrypoint (can be repeated for multiple args)",
		},
		&cli.StringSliceFlag{
			Name:  "cmd",
			Usage: "Override image CMD (can be repeated for multiple args)",
		},
		// Metadata flags
		&cli.StringSliceFlag{
			Name:    "tag",
			Aliases: []string{"metadata", "l"},
			Usage:   "Set tag key-value pair (KEY=VALUE, can be repeated)",
		},
		// Volume mount flags
		&cli.StringSliceFlag{
			Name:    "volume",
			Aliases: []string{"v"},
			Usage:   `Attach volume at creation (format: volume-id:/mount/path[:ro[:overlay=SIZE]]). Can be repeated.`,
		},
	},
	Action:          handleRun,
	HideHelpCommand: true,
}

func handleRun(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) < 1 {
		return fmt.Errorf("image reference required\nUsage: hypeman run [flags] <image>")
	}

	image := args[0]

	client := hypeman.NewClient(getDefaultRequestOptions(cmd)...)

	// Check if image exists and is ready
	// URL-encode the image name to handle slashes (e.g., docker.io/library/nginx:latest)
	imgInfo, err := client.Images.Get(ctx, url.PathEscape(image))
	if err != nil {
		// Image not found, try to pull it
		var apiErr *hypeman.Error
		if ok := isNotFoundError(err, &apiErr); ok {
			fmt.Fprintf(os.Stderr, "Image not found locally. Pulling %s...\n", image)
			imgInfo, err = client.Images.New(ctx, hypeman.ImageNewParams{
				Name: image,
			})
			if err != nil {
				return fmt.Errorf("failed to pull image: %w", err)
			}
		} else {
			return fmt.Errorf("failed to check image: %w", err)
		}
	}

	// Wait for image to be ready (build is asynchronous)
	if err := waitForImageReady(ctx, &client, imgInfo); err != nil {
		return err
	}

	// Generate name if not provided
	name := cmd.String("name")
	if name == "" {
		name = GenerateInstanceName(image)
	}

	// Parse environment variables
	env := make(map[string]string)
	for _, e := range cmd.StringSlice("env") {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		} else {
			fmt.Fprintf(os.Stderr, "Warning: ignoring malformed env var: %s\n", e)
		}
	}

	// Build instance params
	params := hypeman.InstanceNewParams{
		Image:       image,
		Name:        name,
		Vcpus:       hypeman.Opt(int64(cmd.Int("cpus"))),
		Size:        hypeman.Opt(cmd.String("memory")),
		OverlaySize: hypeman.Opt(cmd.String("overlay-size")),
		HotplugSize: hypeman.Opt(cmd.String("hotplug-size")),
	}

	if len(env) > 0 {
		params.Env = env
	}
	if rawCredentials := cmd.String("credentials-json"); rawCredentials != "" {
		credentials := map[string]hypeman.InstanceNewParamsCredential{}
		if err := json.Unmarshal([]byte(rawCredentials), &credentials); err != nil {
			return fmt.Errorf("invalid credentials-json: %w", err)
		}
		params.Credentials = credentials
	}
	autoStandbyPolicy, autoStandbySet, err := buildAutoStandbyPolicy(cmd, "auto-standby-")
	if err != nil {
		return err
	}
	if autoStandbySet {
		params.AutoStandby = autoStandbyPolicy
	}

	// Network configuration
	networkEnabled := cmd.Bool("network")
	bandwidthDown := cmd.String("bandwidth-down")
	bandwidthUp := cmd.String("bandwidth-up")
	egressEnabledSet := cmd.IsSet("network-egress-enabled")
	egressMode := cmd.String("network-egress-mode")

	if !networkEnabled || bandwidthDown != "" || bandwidthUp != "" || egressEnabledSet || egressMode != "" {
		params.Network = hypeman.InstanceNewParamsNetwork{
			Enabled: hypeman.Opt(networkEnabled),
		}
		if bandwidthDown != "" {
			params.Network.BandwidthDownload = hypeman.Opt(bandwidthDown)
		}
		if bandwidthUp != "" {
			params.Network.BandwidthUpload = hypeman.Opt(bandwidthUp)
		}
		if egressEnabledSet || egressMode != "" {
			egress, err := buildNetworkEgress(cmd.Bool("network-egress-enabled"), egressEnabledSet, egressMode)
			if err != nil {
				return err
			}
			params.Network.Egress = egress
		}
	}

	// GPU configuration
	gpuProfile := cmd.String("gpu-profile")
	if gpuProfile != "" {
		params.GPU = hypeman.InstanceNewParamsGPU{
			Profile: hypeman.Opt(gpuProfile),
		}
	}

	// Device passthrough
	devices := cmd.StringSlice("device")
	if len(devices) > 0 {
		params.Devices = devices
	}

	// Hypervisor selection
	hypervisor := cmd.String("hypervisor")
	if hypervisor != "" {
		switch hypervisor {
		case "cloud-hypervisor", "ch":
			params.Hypervisor = hypeman.InstanceNewParamsHypervisorCloudHypervisor
		case "firecracker", "fc":
			params.Hypervisor = hypeman.InstanceNewParamsHypervisorFirecracker
		case "qemu":
			params.Hypervisor = hypeman.InstanceNewParamsHypervisorQemu
		case "vz":
			params.Hypervisor = hypeman.InstanceNewParamsHypervisorVz
		default:
			return fmt.Errorf("invalid hypervisor: %s (must be 'cloud-hypervisor', 'firecracker', 'qemu', or 'vz')", hypervisor)
		}
	}

	// Disk I/O limit
	diskIO := cmd.String("disk-io")
	if diskIO != "" {
		params.DiskIoBps = hypeman.Opt(diskIO)
	}

	// Boot options
	if cmd.IsSet("skip-guest-agent") {
		params.SkipGuestAgent = hypeman.Opt(cmd.Bool("skip-guest-agent"))
	}
	if cmd.IsSet("skip-kernel-headers") {
		params.SkipKernelHeaders = hypeman.Opt(cmd.Bool("skip-kernel-headers"))
	}

	// Entrypoint and CMD overrides
	if entrypoint := cmd.StringSlice("entrypoint"); len(entrypoint) > 0 {
		params.Entrypoint = entrypoint
	}
	if cmdArgs := cmd.StringSlice("cmd"); len(cmdArgs) > 0 {
		params.Cmd = cmdArgs
	}

	// Tags
	tagSpecs := cmd.StringSlice("tag")
	if len(tagSpecs) > 0 {
		tags, malformed := parseKeyValueSpecs(tagSpecs)
		for _, invalid := range malformed {
			fmt.Fprintf(os.Stderr, "Warning: ignoring malformed tag: %s\n", invalid)
		}
		if len(tags) > 0 {
			params.Tags = tags
		}
	}

	// Snapshot policy compression
	if cmd.IsSet("snapshot-compression-enabled") || cmd.IsSet("snapshot-compression-delay") || cmd.IsSet("snapshot-compression-algorithm") || cmd.IsSet("snapshot-compression-level") {
		policy := hypeman.SnapshotPolicyParam{}
		if delay := cmd.String("snapshot-compression-delay"); delay != "" {
			policy.StandbyCompressionDelay = hypeman.Opt(delay)
		}
		if cmd.IsSet("snapshot-compression-enabled") || cmd.IsSet("snapshot-compression-algorithm") || cmd.IsSet("snapshot-compression-level") {
			compression := shared.SnapshotCompressionConfigParam{
				Enabled: cmd.Bool("snapshot-compression-enabled"),
			}
			if !cmd.IsSet("snapshot-compression-enabled") {
				compression.Enabled = true
			}
			if cmd.IsSet("snapshot-compression-level") {
				compression.Level = hypeman.Opt(int64(cmd.Int("snapshot-compression-level")))
			}
			if algorithm := cmd.String("snapshot-compression-algorithm"); algorithm != "" {
				parsedAlgorithm, err := parseSnapshotCompressionAlgorithm(algorithm)
				if err != nil {
					return fmt.Errorf("invalid snapshot compression algorithm: %w", err)
				}
				compression.Algorithm = parsedAlgorithm
			}
			policy.Compression = compression
		}
		params.SnapshotPolicy = policy
	}

	// Volume mounts
	volumeSpecs := cmd.StringSlice("volume")
	if len(volumeSpecs) > 0 {
		var mounts []hypeman.VolumeMountParam
		for _, spec := range volumeSpecs {
			mount, err := parseVolumeSpec(spec)
			if err != nil {
				return fmt.Errorf("invalid volume spec %q: %w", spec, err)
			}
			mounts = append(mounts, mount)
		}
		params.Volumes = mounts
	}

	fmt.Fprintf(os.Stderr, "Creating instance %s...\n", name)

	var opts []option.RequestOption
	if cmd.Root().Bool("debug") {
		opts = append(opts, debugMiddlewareOption)
	}

	result, err := client.Instances.New(
		ctx,
		params,
		opts...,
	)
	if err != nil {
		return err
	}

	// Output instance ID (useful for scripting)
	fmt.Println(result.ID)

	return nil
}

func buildNetworkEgress(enabled bool, enabledSet bool, mode string) (hypeman.InstanceNewParamsNetworkEgress, error) {
	egress := hypeman.InstanceNewParamsNetworkEgress{}
	if enabledSet {
		egress.Enabled = hypeman.Opt(enabled)
	} else if mode != "" {
		egress.Enabled = hypeman.Opt(true)
	}

	if mode != "" {
		switch mode {
		case "all", "http_https_only":
			egress.Enforcement = hypeman.InstanceNewParamsNetworkEgressEnforcement{
				Mode: mode,
			}
		default:
			return hypeman.InstanceNewParamsNetworkEgress{}, fmt.Errorf("invalid network-egress-mode: %s (must be 'all' or 'http_https_only')", mode)
		}
	}

	return egress, nil
}

// isNotFoundError checks if err is a 404 not found error
func isNotFoundError(err error, target **hypeman.Error) bool {
	if apiErr, ok := err.(*hypeman.Error); ok {
		*target = apiErr
		return apiErr.Response != nil && apiErr.Response.StatusCode == 404
	}
	return false
}

// waitForImageReady polls image status until it becomes ready or failed
func waitForImageReady(ctx context.Context, client *hypeman.Client, img *hypeman.Image) error {
	if img.Status == hypeman.ImageStatusReady {
		return nil
	}
	if img.Status == hypeman.ImageStatusFailed {
		if img.Error != "" {
			return fmt.Errorf("image build failed: %s", img.Error)
		}
		return fmt.Errorf("image build failed")
	}

	// Poll until ready using the normalized image name from the API response
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	// Show initial status
	showImageStatus(img)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			updated, err := client.Images.Get(ctx, url.PathEscape(img.Name))
			if err != nil {
				return fmt.Errorf("failed to check image status: %w", err)
			}

			// Show status update if changed
			if updated.Status != img.Status {
				showImageStatus(updated)
				img = updated
			}

			switch updated.Status {
			case hypeman.ImageStatusReady:
				return nil
			case hypeman.ImageStatusFailed:
				if updated.Error != "" {
					return fmt.Errorf("image build failed: %s", updated.Error)
				}
				return fmt.Errorf("image build failed")
			}
		}
	}
}

// parseVolumeSpec parses a volume mount specification string.
// Format: volume-id:/mount/path[:ro[:overlay=SIZE]]
// Examples:
//
//	my-vol:/data
//	my-vol:/data:ro
//	my-vol:/data:ro:overlay=10GB
func parseVolumeSpec(spec string) (hypeman.VolumeMountParam, error) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) < 2 {
		return hypeman.VolumeMountParam{}, fmt.Errorf("expected format volume-id:/mount/path[:ro[:overlay=SIZE]]")
	}

	volumeID := parts[0]
	if volumeID == "" {
		return hypeman.VolumeMountParam{}, fmt.Errorf("volume ID cannot be empty")
	}

	remaining := parts[1]
	// Split remaining by colon to get mount path and options
	segments := strings.Split(remaining, ":")
	mountPath := segments[0]
	if mountPath == "" {
		return hypeman.VolumeMountParam{}, fmt.Errorf("mount path cannot be empty")
	}

	mount := hypeman.VolumeMountParam{
		VolumeID:  volumeID,
		MountPath: mountPath,
	}

	// Parse optional flags
	for _, seg := range segments[1:] {
		switch {
		case seg == "ro":
			mount.Readonly = hypeman.Opt(true)
		case strings.HasPrefix(seg, "overlay="):
			mount.Overlay = hypeman.Opt(true)
			mount.OverlaySize = hypeman.Opt(strings.TrimPrefix(seg, "overlay="))
		default:
			return hypeman.VolumeMountParam{}, fmt.Errorf("unknown option %q", seg)
		}
	}

	return mount, nil
}

// showImageStatus prints image build status to stderr
func showImageStatus(img *hypeman.Image) {
	switch img.Status {
	case hypeman.ImageStatusPending:
		if img.QueuePosition > 0 {
			fmt.Fprintf(os.Stderr, "Queued (position %d)...\n", img.QueuePosition)
		} else {
			fmt.Fprintf(os.Stderr, "Queued...\n")
		}
	case hypeman.ImageStatusPulling:
		fmt.Fprintf(os.Stderr, "Pulling image...\n")
	case hypeman.ImageStatusConverting:
		fmt.Fprintf(os.Stderr, "Converting to disk image...\n")
	case hypeman.ImageStatusReady:
		fmt.Fprintf(os.Stderr, "Image ready.\n")
	}
}
