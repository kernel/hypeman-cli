// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"testing"

	"github.com/onkernel/hypeman-cli/internal/mocktest"
)

func TestImagesCreate(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"images", "create",
		"--name", "docker.io/library/nginx:latest",
	)
}

func TestImagesList(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"images", "list",
	)
}

func TestImagesDelete(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"images", "delete",
		"--name", "name",
	)
}

func TestImagesGet(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"images", "get",
		"--name", "name",
	)
}
