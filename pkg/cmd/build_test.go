// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"testing"

	"github.com/kernel/hypeman-cli/internal/mocktest"
)

func TestBuildsCreate(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"builds", "create",
		"--source", "",
		"--base-image-digest", "base_image_digest",
		"--cache-scope", "cache_scope",
		"--dockerfile", "dockerfile",
		"--global-cache-key", "global_cache_key",
		"--is-admin-build", "is_admin_build",
		"--secrets", "secrets",
		"--timeout-seconds", "0",
	)
}

func TestBuildsList(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"builds", "list",
	)
}

func TestBuildsEvents(t *testing.T) {
	t.Skip("Prism doesn't support text/event-stream responses")
	mocktest.TestRunMockTestWithFlags(
		t,
		"builds", "events",
		"--id", "id",
		"--follow=true",
	)
}

func TestBuildsGet(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"builds", "get",
		"--id", "id",
	)
}
