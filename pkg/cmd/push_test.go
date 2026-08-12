package cmd

import (
	"bytes"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
)

func TestRenderPushProgress(t *testing.T) {
	updates := make(chan v1.Update, 3)
	updates <- v1.Update{Total: 2048, Complete: 1024}
	updates <- v1.Update{Total: 2048, Complete: 2048}
	updates <- v1.Update{Error: assert.AnError}
	close(updates)

	var output bytes.Buffer
	renderPushProgress(updates, &output, true)

	assert.Equal(t, "\r1.0 KB / 2.0 KB\r2.0 KB / 2.0 KB\n", output.String())
}

func TestRenderPushProgressNonInteractive(t *testing.T) {
	updates := make(chan v1.Update, 1)
	updates <- v1.Update{Total: 1024, Complete: 1024}
	close(updates)

	var output bytes.Buffer
	renderPushProgress(updates, &output, false)

	assert.Empty(t, output.String())
}
