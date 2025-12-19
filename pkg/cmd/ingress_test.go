// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"testing"

	"github.com/onkernel/hypeman-cli/internal/mocktest"
)

func TestIngressesCreate(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"ingresses", "create",
		"--name", "my-api-ingress",
		"--rule", "{match: {hostname: '{instance}.example.com', port: 8080}, target: {instance: '{instance}', port: 8080}, redirect_http: true, tls: true}",
	)
}

func TestIngressesList(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"ingresses", "list",
	)
}

func TestIngressesDelete(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"ingresses", "delete",
		"--id", "id",
	)
}

func TestIngressesGet(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"ingresses", "get",
		"--id", "id",
	)
}
