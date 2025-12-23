// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"testing"

	"github.com/onkernel/hypeman-cli/internal/mocktest"
)

func TestInstancesCreate(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances", "create",
		"--image", "docker.io/library/alpine:latest",
		"--name", "my-workload-1",
		"--device", "l4-gpu",
		"--env", "{PORT: '3000', NODE_ENV: production}",
		"--hotplug-size", "2GB",
		"--network", "{enabled: true}",
		"--overlay-size", "20GB",
		"--size", "2GB",
		"--vcpus", "2",
		"--volume", "{mount_path: /mnt/data, volume_id: vol-abc123, overlay: true, overlay_size: 1GB, readonly: true}",
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
		"--follow",
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
		"--follow-links",
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
