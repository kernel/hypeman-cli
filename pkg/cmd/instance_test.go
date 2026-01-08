// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"testing"

	"github.com/onkernel/hypeman-cli/internal/mocktest"
	"github.com/onkernel/hypeman-cli/internal/requestflag"
)

func TestInstancesCreate(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "create",
		"--image", "docker.io/library/alpine:latest",
		"--name", "my-workload-1",
		"--device", "l4-gpu",
		"--disk-io-bps", "100MB/s",
		"--env", "{PORT: '3000', NODE_ENV: production}",
		"--hotplug-size", "2GB",
		"--hypervisor", "cloud-hypervisor",
		"--network", "{bandwidth_download: 1Gbps, bandwidth_upload: 1Gbps, enabled: true}",
		"--overlay-size", "20GB",
		"--size", "2GB",
		"--vcpus", "2",
		"--volume", "{mount_path: /mnt/data, volume_id: vol-abc123, overlay: true, overlay_size: 1GB, readonly: true}",
	)

	// Check that inner flags have been set up correctly
	requestflag.CheckInnerFlags(instancesCreate)

	// Alternative argument passing style using inner flags
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "create",
		"--image", "docker.io/library/alpine:latest",
		"--name", "my-workload-1",
		"--device", "l4-gpu",
		"--disk-io-bps", "100MB/s",
		"--env", "{PORT: '3000', NODE_ENV: production}",
		"--hotplug-size", "2GB",
		"--hypervisor", "cloud-hypervisor",
		"--network.bandwidth_download", "1Gbps",
		"--network.bandwidth_upload", "1Gbps",
		"--network.enabled=true",
		"--overlay-size", "20GB",
		"--size", "2GB",
		"--vcpus", "2",
		"--volume.mount_path", "/mnt/data",
		"--volume.volume_id", "vol-abc123",
		"--volume.overlay=true",
		"--volume.overlay_size", "1GB",
		"--volume.readonly=true",
	)
}

func TestInstancesList(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "list",
	)
}

func TestInstancesDelete(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "delete",
		"--id", "id",
	)
}

func TestInstancesGet(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "get",
		"--id", "id",
	)
}

func TestInstancesLogs(t *testing.T) {
	t.Skip("Prism doesn't support text/event-stream responses")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "logs",
		"--id", "id",
		"--follow=true",
		"--source", "app",
		"--tail", "0",
	)
}

func TestInstancesRestore(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "restore",
		"--id", "id",
	)
}

func TestInstancesStandby(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "standby",
		"--id", "id",
	)
}

func TestInstancesStart(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "start",
		"--id", "id",
	)
}

func TestInstancesStat(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "stat",
		"--id", "id",
		"--path", "path",
		"--follow-links=true",
	)
}

func TestInstancesStop(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "stop",
		"--id", "id",
	)
}
