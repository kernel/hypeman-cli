package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/kernel/hypeman-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// TestSubcommandsAreListedInHelp guards against SDK surface that is wired up but
// unreachable in practice: urfave/cli silently renders the no-subcommand help
// template for a command that has exactly one subcommand and a hidden help
// command, which omits the COMMANDS section entirely.
func TestSubcommandsAreListedInHelp(t *testing.T) {
	var walk func(t *testing.T, path []string, parent *cli.Command)
	walk = func(t *testing.T, path []string, parent *cli.Command) {
		subs := visibleSubcommands(parent)
		if len(subs) == 0 {
			return
		}

		help := renderHelp(t, path)
		for _, sub := range subs {
			assert.Contains(t, help, sub.Name,
				"`hypeman %s --help` does not list subcommand %q",
				strings.Join(path, " "), sub.Name)
		}

		for _, sub := range subs {
			walk(t, append(append([]string{}, path...), sub.Name), sub)
		}
	}

	walk(t, nil, Command)
}

func visibleSubcommands(parent *cli.Command) []*cli.Command {
	var subs []*cli.Command
	for _, sub := range parent.VisibleCommands() {
		// Skip the generated "@manpages"/"@completion" style helper commands.
		if strings.HasPrefix(sub.Name, "@") {
			continue
		}
		subs = append(subs, sub)
	}
	return subs
}

func renderHelp(t *testing.T, path []string) string {
	t.Helper()

	var buf bytes.Buffer
	origWriter, origErrWriter := Command.Writer, Command.ErrWriter
	Command.Writer, Command.ErrWriter = &buf, &buf
	defer func() {
		Command.Writer, Command.ErrWriter = origWriter, origErrWriter
	}()

	args := append([]string{"hypeman"}, path...)
	args = append(args, "--help")
	require.NoError(t, Command.Run(context.Background(), args))

	return buf.String()
}

func TestParseInstanceWaitState(t *testing.T) {
	t.Run("accepts mixed-case state names", func(t *testing.T) {
		state, err := parseInstanceWaitState("rUnNiNg")
		require.NoError(t, err)
		assert.Equal(t, hypeman.InstanceWaitParamsStateRunning, state)
	})

	t.Run("rejects unsupported state names", func(t *testing.T) {
		_, err := parseInstanceWaitState("booting")
		require.EqualError(t, err, "invalid state: booting (must be Created, Initializing, Running, Paused, Shutdown, Stopped, Standby, or Unknown)")
	})
}

func TestParseAutoStandbyPorts(t *testing.T) {
	t.Run("parses valid port values", func(t *testing.T) {
		ports, err := parseAutoStandbyPorts([]string{"80", " 443 "}, "ignore-destination-port")
		require.NoError(t, err)
		assert.Equal(t, []int64{80, 443}, ports)
	})

	t.Run("rejects out-of-range ports", func(t *testing.T) {
		_, err := parseAutoStandbyPorts([]string{"70000"}, "ignore-destination-port")
		require.EqualError(t, err, `ignore-destination-port must be between 1 and 65535: "70000"`)
	})
}
