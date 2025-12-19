// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"testing"

	"github.com/onkernel/hypeman-cli/internal/mocktest"
)

func TestVolumesCreate(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"volumes", "create",
		"--name", "my-data-volume",
		"--size-gb", "10",
		"--id", "vol-data-1",
	)
}

func TestVolumesList(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"volumes", "list",
	)
}

func TestVolumesDelete(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"volumes", "delete",
		"--id", "id",
	)
}

func TestVolumesGet(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"volumes", "get",
		"--id", "id",
	)
}
