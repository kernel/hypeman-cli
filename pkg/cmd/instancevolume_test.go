// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"testing"

	"github.com/onkernel/hypeman-cli/internal/mocktest"
)

func TestInstancesVolumesAttach(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances:volumes", "attach",
		"--id", "id",
		"--volume-id", "volumeId",
		"--mount-path", "/mnt/data",
		"--readonly",
	)
}

func TestInstancesVolumesDetach(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"instances:volumes", "detach",
		"--id", "id",
		"--volume-id", "volumeId",
	)
}
