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
	renderPushProgress(updates, &output, true, make(chan struct{}))

	assert.Equal(t, "\r1.0 KB / 2.0 KB\r2.0 KB / 2.0 KB\n", output.String())
}

func TestRenderPushProgressNonInteractive(t *testing.T) {
	updates := make(chan v1.Update, 1)
	updates <- v1.Update{Total: 1024, Complete: 1024}
	close(updates)

	var output bytes.Buffer
	renderPushProgress(updates, &output, false, make(chan struct{}))

	assert.Empty(t, output.String())
}

func TestPushStatusRenderer(t *testing.T) {
	var output bytes.Buffer
	renderer := &pushStatusRenderer{output: &output, interactive: true}

	renderer.update("queued")
	renderer.update("queued")
	renderer.update("pushing 1.0 MB")
	renderer.finish()

	assert.Equal(t, "\r\033[Kqueued\r\033[Kpushing 1.0 MB\n", output.String())
}

func TestPushStatusRendererNonInteractive(t *testing.T) {
	var output bytes.Buffer
	renderer := &pushStatusRenderer{output: &output}

	renderer.update("queued")
	renderer.update("pushing 1.0 MB")
	renderer.finish()

	assert.Equal(t, "queued\npushing 1.0 MB\n", output.String())
}
