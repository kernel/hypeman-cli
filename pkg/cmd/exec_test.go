package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dialExecTestServer starts a WebSocket server that runs serve on the
// accepted connection and returns a client connected to it.
func dialExecTestServer(t *testing.T, serve func(*websocket.Conn)) *websocket.Conn {
	t.Helper()
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		serve(ws)
	}))
	t.Cleanup(server.Close)

	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	t.Cleanup(func() { ws.Close() })
	return ws
}

func TestRunExecNonInteractivePropagatesExitCode(t *testing.T) {
	ws := dialExecTestServer(t, func(ws *websocket.Conn) {
		_ = ws.WriteMessage(websocket.TextMessage, []byte(`{"exitCode":3}`))
		_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	})

	exitCode, err := runExecNonInteractive(ws)
	require.NoError(t, err)
	assert.Equal(t, 3, exitCode)
}

func TestRunExecNonInteractiveRequiresExitCode(t *testing.T) {
	tests := []struct {
		name  string
		serve func(*websocket.Conn)
	}{
		{
			name: "normal close without exit code",
			serve: func(ws *websocket.Conn) {
				_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			},
		},
		{
			name: "malformed exit code",
			serve: func(ws *websocket.Conn) {
				_ = ws.WriteMessage(websocket.TextMessage, []byte(`{"exitCode":"oops"}`))
				_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			},
		},
		{
			name: "null exit code",
			serve: func(ws *websocket.Conn) {
				_ = ws.WriteMessage(websocket.TextMessage, []byte(`{"exitCode":null}`))
				_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			},
		},
		{
			name: "connection dropped before exit code",
			serve: func(ws *websocket.Conn) {
				_ = ws.NetConn().Close()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := dialExecTestServer(t, tt.serve)

			exitCode, err := runExecNonInteractive(ws)
			require.ErrorContains(t, err, "before receiving an exit code")
			assert.NotEqual(t, 0, exitCode)
		})
	}
}
