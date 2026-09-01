package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapabilitiesCmdStructure(t *testing.T) {
	assert.Equal(t, "capabilities", capabilitiesCmd.Name)
	assert.Contains(t, capabilitiesCmd.Aliases, "capability")
	assert.NotNil(t, capabilitiesCmd.Action)
}

func TestShowCapabilities(t *testing.T) {
	payload := []byte(`{
		"default_runtime": {"available": true, "name": "cloud-hypervisor"},
		"features": ["instances", "images", "devices"],
		"host": {"arch": "amd64", "os": "linux"},
		"images": {"default_platform": "linux/amd64", "platforms": ["linux/amd64", "linux/arm64"]},
		"network": {"guest_to_guest": false, "model": "bridge", "gateway": "192.168.100.1", "subnet": "192.168.100.0/24"},
		"runtimes": [
			{"available": true, "features": ["snapshots", "standby"], "name": "cloud-hypervisor"},
			{"available": false, "features": [], "name": "qemu"}
		],
		"server": {"api_version": "1.2.3", "version": "abc1234"}
	}`)

	var buf bytes.Buffer
	require.NoError(t, showCapabilities(&buf, payload))
	out := buf.String()

	assert.Contains(t, out, "Version:      abc1234")
	assert.Contains(t, out, "API version:  1.2.3")
	assert.Contains(t, out, "OS:           linux")
	assert.Contains(t, out, "Arch:         amd64")
	assert.Contains(t, out, "Name:         cloud-hypervisor")
	assert.Contains(t, out, "cloud-hypervisor  yes")
	assert.Contains(t, out, "snapshots, standby")
	assert.Contains(t, out, "qemu              no")
	assert.Contains(t, out, "Default platform:  linux/amd64")
	assert.Contains(t, out, "Platforms:         linux/amd64, linux/arm64")
	assert.Contains(t, out, "Model:             bridge")
	assert.Contains(t, out, "Gateway:           192.168.100.1")
	assert.Contains(t, out, "Subnet:            192.168.100.0/24")
	assert.Contains(t, out, "Guest to guest:    no")
	assert.Contains(t, out, "instances, images, devices")
}

func TestShowCapabilitiesOmitsMissingOptionalFields(t *testing.T) {
	payload := []byte(`{
		"default_runtime": {"available": false, "name": "vz"},
		"features": [],
		"host": {"arch": "arm64", "os": "darwin"},
		"images": {"default_platform": "linux/arm64", "platforms": ["linux/arm64"]},
		"network": {"guest_to_guest": true, "model": "nat"},
		"runtimes": [],
		"server": {"api_version": "1.2.3", "version": "unknown"}
	}`)

	var buf bytes.Buffer
	require.NoError(t, showCapabilities(&buf, payload))
	out := buf.String()

	assert.Contains(t, out, "Gateway:           -")
	assert.Contains(t, out, "Subnet:            -")
	assert.NotContains(t, out, "RUNTIMES")
	assert.Contains(t, out, "SERVER FEATURES\n  -")
}
