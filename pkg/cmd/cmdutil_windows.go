<<<<<<< HEAD
//go:build windows

package cmd

import (
	"errors"
	"os"
)

// createSocketPair is not supported on Windows, so we return an error
// which causes createPagerFiles to fall back to using pipes.
func createSocketPair() (*os.File, *os.File, bool, error) {
	return nil, nil, false, errors.New("socket pairs not supported on Windows")
}
||||||| parent of fe526a1 (fix(cli): fix compilation on Windows)
=======
//go:build windows

package cmd

import "os"

func streamOutputOSSpecific(label string, generateOutput func(w *os.File) error) error {
	// We have a trick with sockets that we use when possible on Unix-like systems. Those APIs aren't
	// available on Windows, so we fall back to using pipes.
	return streamToPagerWithPipe(label, generateOutput)
}
>>>>>>> fe526a1 (fix(cli): fix compilation on Windows)
