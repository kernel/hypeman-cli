// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package cmd

import (
	"testing"

	"github.com/onkernel/hypeman-cli/internal/mocktest"
	"github.com/onkernel/hypeman-cli/internal/requestflag"
)

func TestIngressesCreate(t *testing.T) {
	t.Skip("Prism tests are disabled")
	mocktest.TestRunMockTestWithFlags(
		t,
		"ingresses", "create",
		"--name", "my-api-ingress",
		"--rule", "{match: {hostname: '{instance}.example.com', port: 8080}, target: {instance: '{instance}', port: 8080}, redirect_http: true, tls: true}",
	)

	// Check that inner flags have been set up correctly
	requestflag.CheckInnerFlags(ingressesCreate)

	// Alternative argument passing style using inner flags
	mocktest.TestRunMockTestWithFlags(
		t,
		"ingresses", "create",
		"--name", "my-api-ingress",
		"--rule.match", "{hostname: '{instance}.example.com', port: 8080}",
		"--rule.target", "{instance: '{instance}', port: 8080}",
		"--rule.redirect_http=true",
		"--rule.tls=true",
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
